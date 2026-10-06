package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"voxlab/pkg/audio"
	"voxlab/pkg/config"
	"voxlab/pkg/engine"
	"voxlab/pkg/intent"
	"voxlab/pkg/statemachine"
	"voxlab/pkg/tts"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local test bench development
	},
}

// Server is the main VoxLab web and voice API server.
type Server struct {
	cfg          *config.AppConfig
	httpServer   *http.Server
	dsp          *audio.DSPProcessor
	ringBuffer   *audio.RingBuffer
	stateMachine *statemachine.StateMachine
	engine       engine.SpeechEngine
	matcher      *intent.IntentMatcher
	ttsMgr       *tts.TTSManager
	webMicSource *audio.WebSocketSource
	hostMicSource *audio.HostSimulatedSource
	activeSource string
	audioInChan  chan []float32

	clientsMu sync.RWMutex
	clients   map[*websocket.Conn]bool

	dictationDraft string
	dictationTarget string
}

// NewServer initializes the VoxLab server with all pipelines.
func NewServer(cfg *config.AppConfig) (*Server, error) {
	dsp := audio.NewDSPProcessor(
		cfg.Audio.SampleRate,
		cfg.Audio.HighPassCutoffHz,
		cfg.Audio.NoiseGateThresholdDBFS,
		cfg.Audio.AGCEnabled,
		cfg.Audio.TargetRMS,
	)

	ringBuf := audio.NewRingBuffer(int(float64(cfg.Audio.SampleRate) * cfg.KWS.PreRollDurationSec))
	sm := statemachine.NewStateMachine()

	catalog, err := intent.LoadCatalog(cfg.Intent.CatalogPath)
	if err != nil {
		log.Printf("[Warning] Failed loading catalog from %s, using defaults: %v", cfg.Intent.CatalogPath, err)
		catalog = intent.DefaultCatalog()
	}
	matcher := intent.NewIntentMatcher(catalog, cfg.Intent.ConfidenceThreshold, cfg.Intent.MarginThreshold)

	var eng engine.SpeechEngine
	if cfg.Engine.Mode == "sherpa" {
		eng = engine.NewSherpaRunner(cfg)
	} else {
		eng = engine.NewSimulatorEngine()
	}

	ttsMgr := tts.NewTTSManager(eng, dsp)
	webMic := audio.NewWebSocketSource()
	hostMic := audio.NewHostSimulatedSource(cfg.Audio.SampleRate, cfg.Audio.ChunkSamples)

	s := &Server{
		cfg:           cfg,
		dsp:           dsp,
		ringBuffer:    ringBuf,
		stateMachine:  sm,
		engine:        eng,
		matcher:       matcher,
		ttsMgr:        ttsMgr,
		webMicSource:  webMic,
		hostMicSource: hostMic,
		activeSource:  "web_ui_mic",
		audioInChan:   make(chan []float32, 200),
		clients:       make(map[*websocket.Conn]bool),
	}

	// Listen for state transitions and broadcast them
	sm.AddListener(func(ev statemachine.StateEvent) {
		s.BroadcastJSON(map[string]interface{}{
			"event": "voice.state",
			"data": map[string]interface{}{
				"from_state": ev.FromState,
				"to_state":   ev.ToState,
				"trigger":    ev.Trigger,
				"timestamp":  ev.Timestamp.Format(time.RFC3339Nano),
				"extra":      ev.Data,
			},
		})
	})

	return s, nil
}

// Start begins listening and serving HTTP & WebSocket requests.
func (s *Server) Start() error {
	// Start audio processing loop
	ctx, cancel := context.WithCancel(context.Background())
	_ = cancel
	go s.audioPipelineLoop(ctx)

	// Connect default web mic source
	_ = s.webMicSource.Start(ctx, s.audioInChan)

	mux := http.NewServeMux()

	// Static Web UI
	mux.Handle("/", http.FileServer(http.Dir(s.cfg.StaticDir)))

	// WebSocket IPC
	mux.HandleFunc("/ws", s.handleWebSocket)

	// REST APIs
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/catalog", s.handleCatalog)
	mux.HandleFunc("/api/voices", s.handleVoices)
	mux.HandleFunc("/api/tts", s.handleTTS)

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("[VoxLab] Server listening on http://%s (Engine: %s, UI: %s)", addr, s.engine.Name(), s.cfg.StaticDir)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// audioPipelineLoop processes incoming PCM audio chunks through DSP, KWS, and ASR.
func (s *Server) audioPipelineLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case chunk, ok := <-s.audioInChan:
			if !ok {
				return
			}
			s.processIncomingAudio(chunk)
		}
	}
}

// processIncomingAudio handles a single 30ms audio chunk through all active filters and gates.
func (s *Server) processIncomingAudio(rawChunk []float32) {
	// Step 1: Preprocessing & Noise Gating
	cleanChunk, rms, dbfs, passedGate := s.dsp.ProcessChunk(rawChunk)

	// Broadcast meter update (throttled every ~90ms to save WS bandwidth)
	now := time.Now().UnixNano()
	if (now/int64(time.Millisecond))%90 < 30 {
		s.BroadcastJSON(map[string]interface{}{
			"event": "audio.meter",
			"data": map[string]interface{}{
				"rms":         rms,
				"dbfs":        dbfs,
				"passed_gate": passedGate,
				"echo_muted":  s.dsp.IsEchoMuted(),
			},
		})
	}

	// Always write clean chunk to circular pre-roll buffer
	s.ringBuffer.Write(cleanChunk)

	state := s.stateMachine.Current()

	// If echo muted or silent chunk below gate, skip ASR inference when idle
	if s.dsp.IsEchoMuted() {
		return
	}

	// Step 2: Wake Word Detection when in IDLE
	if (state == statemachine.StateIdle) && s.cfg.KWS.Enabled && passedGate {
		detected, kw, conf := s.engine.DetectWakeWord(cleanChunk)
		if detected && conf >= s.cfg.KWS.Threshold {
			log.Printf("[VoxLab] Wake-word '%s' detected (confidence: %.2f)", kw, conf)
			s.stateMachine.WakeWordTriggered(kw, conf)
			s.BroadcastJSON(map[string]interface{}{
				"event": "wake_word_detected",
				"data": map[string]interface{}{
					"phrase":     kw,
					"confidence": conf,
				},
			})
			return
		}
	}

	// Step 3: Speech Recognition when in listening states
	if state == statemachine.StateCommandListen {
		asrRes, err := s.engine.ProcessASRChunk(cleanChunk, false)
		if err != nil {
			log.Printf("[Error] Command ASR error: %v", err)
			return
		}
		if asrRes != nil {
			if !asrRes.IsFinal {
				s.BroadcastJSON(map[string]interface{}{
					"event": "transcript.partial",
					"data": map[string]interface{}{
						"mode":       "command",
						"transcript": asrRes.Transcript,
						"tokens":     asrRes.Tokens,
					},
				})
			} else {
				// Final utterance recognized!
				s.BroadcastJSON(map[string]interface{}{
					"event": "transcript.final",
					"data": map[string]interface{}{
						"mode":       "command",
						"transcript": asrRes.Transcript,
					},
				})
				s.handleFinalCommandTranscript(asrRes.Transcript)
			}
		}
	} else if state == statemachine.StateAnnotationListen {
		asrRes, err := s.engine.ProcessASRChunk(cleanChunk, true)
		if err != nil {
			log.Printf("[Error] Dictation ASR error: %v", err)
			return
		}
		if asrRes != nil {
			if !asrRes.IsFinal {
				s.dictationDraft = asrRes.Transcript
				s.BroadcastJSON(map[string]interface{}{
					"event": "transcript.partial",
					"data": map[string]interface{}{
						"mode":       "dictation",
						"transcript": asrRes.Transcript,
						"tokens":     asrRes.Tokens,
					},
				})
			} else {
				s.dictationDraft = asrRes.Transcript
				s.stateMachine.SpeechEndpointed(asrRes.Transcript)
				words := len(asrRes.Tokens)
				s.BroadcastJSON(map[string]interface{}{
					"event": "annotation.draft",
					"data": map[string]interface{}{
						"target_id":  s.dictationTarget,
						"draft_text": asrRes.Transcript,
						"word_count": words,
					},
				})
			}
		}
	}
}

// handleFinalCommandTranscript evaluates a finalized phrase through Intent Matcher and State Machine.
func (s *Server) handleFinalCommandTranscript(transcript string) {
	start := time.Now()
	s.stateMachine.SpeechEndpointed(transcript)

	decision := s.matcher.Match(transcript)
	intentLatency := time.Since(start).Milliseconds()

	s.BroadcastJSON(map[string]interface{}{
		"event": "telemetry.benchmark",
		"data": map[string]interface{}{
			"intent_latency_ms": intentLatency,
		},
	})

	switch decision.Decision {
	case "ACCEPT":
		s.BroadcastJSON(map[string]interface{}{
			"event": "intent_matched",
			"data": map[string]interface{}{
				"intent_id":      decision.IntentID,
				"confidence":     decision.Confidence,
				"margin":         decision.Margin,
				"raw_transcript": decision.RawTranscript,
				"slots":          decision.Slots,
				"top_candidates": decision.TopCandidates,
			},
		})
		s.stateMachine.FinishProcess()

	case "REJECT":
		s.BroadcastJSON(map[string]interface{}{
			"event": "utterance_rejected",
			"data": map[string]interface{}{
				"reason":         decision.Reason,
				"confidence":     decision.Confidence,
				"margin":         decision.Margin,
				"raw_transcript": decision.RawTranscript,
				"top_candidates": decision.TopCandidates,
			},
		})
		s.stateMachine.FinishProcess()

	case "CLARIFY":
		s.stateMachine.RequestConfirmation(decision.IntentID)
		s.BroadcastJSON(map[string]interface{}{
			"event": "intent_clarification_required",
			"data": map[string]interface{}{
				"intent_id":      decision.IntentID,
				"reason":         decision.Reason,
				"confidence":     decision.Confidence,
				"raw_transcript": decision.RawTranscript,
				"missing_slots":  decision.MissingSlots,
			},
		})
	}
}

// handleWebSocket manages client connections and control messaging.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[Error] WS Upgrade: %v", err)
		return
	}
	defer conn.Close()

	s.clientsMu.Lock()
	s.clients[conn] = true
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()
	}()

	// Send initial state snapshot
	_ = conn.WriteJSON(map[string]interface{}{
		"event": "voice.state",
		"data": map[string]interface{}{
			"to_state":  s.stateMachine.Current(),
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})

	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}

		if messageType == websocket.BinaryMessage {
			// Binary PCM audio chunk from Web UI mic (16-bit PCM little endian)
			if s.activeSource == "web_ui_mic" {
				samples := audio.BytesToFloat32PCM(payload)
				s.webMicSource.PushChunk(samples)
			}
		} else if messageType == websocket.TextMessage {
			var req map[string]interface{}
			if err := json.Unmarshal(payload, &req); err != nil {
				continue
			}

			action, _ := req["action"].(string)
			switch action {
			case "activate":
				mode, _ := req["mode"].(string)
				if mode == "dictation" {
					target, _ := req["target_id"].(string)
					s.dictationTarget = target
					s.stateMachine.StartDictation(target)
				} else {
					s.stateMachine.StartCommand()
				}

			case "cancel":
				s.stateMachine.Cancel()
				s.engine.ResetASR()

			case "confirm":
				approved, _ := req["approved"].(bool)
				s.stateMachine.ConfirmAction(approved)

			case "speak":
				if p, ok := req["payload"].(map[string]interface{}); ok {
					text, _ := p["text"].(string)
					voice, _ := p["voice"].(string)
					speed, _ := p["speed"].(float64)
					if speed <= 0 {
						speed = 1.0
					}
					go s.executeTTS(text, voice, speed)
				}

			case "set_source":
				source, _ := req["source"].(string)
				s.activeSource = source
				log.Printf("[VoxLab] Active audio source set to: %s", source)

			case "inject_text":
				text, _ := req["text"].(string)
				mode, _ := req["mode"].(string)
				if mode == "dictation" {
					s.dictationDraft = text
					s.BroadcastJSON(map[string]interface{}{
						"event": "annotation.draft",
						"data": map[string]interface{}{
							"target_id":  "injected",
							"draft_text": text,
							"word_count": len(intent.Tokenize(text)),
						},
					})
				} else {
					s.handleFinalCommandTranscript(text)
				}
			}
		}
	}
}

// executeTTS executes Kokoro TTS and broadcasts audio waveform and metrics.
func (s *Server) executeTTS(text, voice string, speed float64) {
	if text == "" {
		return
	}
	if voice == "" {
		voice = s.cfg.TTS.DefaultVoice
	}

	s.stateMachine.SetSpeaking(true)
	defer s.stateMachine.SetSpeaking(false)

	s.BroadcastJSON(map[string]interface{}{
		"event": "tts.started",
		"data": map[string]interface{}{
			"text":  text,
			"voice": voice,
			"speed": speed,
		},
	})

	ttsRes, err := s.ttsMgr.Synthesize(engine.TTSRequest{
		Text:       text,
		Voice:      voice,
		Speed:      speed,
		SampleRate: s.cfg.TTS.SampleRate,
	})
	if err != nil {
		log.Printf("[Error] TTS error: %v", err)
		return
	}

	wavB64 := base64.StdEncoding.EncodeToString(ttsRes.WAVBytes)

	s.BroadcastJSON(map[string]interface{}{
		"event": "tts.finished",
		"data": map[string]interface{}{
			"text":         text,
			"duration_sec": ttsRes.DurationSec,
			"latency_ms":   ttsRes.LatencyMs,
			"sample_rate":  ttsRes.SampleRate,
			"wav_base64":   wavB64,
		},
	})
}

// BroadcastJSON sends a JSON event to all connected WebSocket clients.
func (s *Server) BroadcastJSON(v interface{}) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for client := range s.clients {
		_ = client.WriteJSON(v)
	}
}

// REST Handlers
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.cfg)
}

func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	cat, _ := intent.LoadCatalog(s.cfg.Intent.CatalogPath)
	_ = json.NewEncoder(w).Encode(cat)
}

func (s *Server) handleVoices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.ttsMgr.Voices())
}

func (s *Server) handleTTS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req engine.TTSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := s.ttsMgr.Synthesize(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "audio/wav")
	_, _ = w.Write(res.WAVBytes)
}
