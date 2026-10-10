package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
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
	cfg           *config.AppConfig
	httpServer    *http.Server
	dsp           *audio.DSPProcessor
	ringBuffer    *audio.RingBuffer
	stateMachine  *statemachine.StateMachine
	engine        engine.SpeechEngine
	matcher       *intent.IntentMatcher
	ttsMgr        *tts.TTSManager
	webMicSource  *audio.WebSocketSource
	hostMicSource *audio.HostNativeSource
	wavSource     *audio.WavFileSource
	activeSource  string
	sourceMu      sync.Mutex
	audioInChan   chan []float32

	asrEngineMode string // "sherpa", "browser", "sim"
	engineMu      sync.RWMutex
	sherpaEngine  engine.SpeechEngine
	simEngine     engine.SpeechEngine

	ctx    context.Context
	cancel context.CancelFunc

	clientsMu sync.RWMutex
	clients   map[*websocket.Conn]*sync.Mutex

	monitorMu      sync.RWMutex
	monitorEnabled bool

	dictationDraft  string
	dictationTarget string

	meterMu       sync.Mutex
	lastMeterTime time.Time
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

	sherpaRunner := engine.NewSherpaRunner(cfg)
	simRunner := engine.NewSimulatorEngine()

	asrMode := "sim"
	var eng engine.SpeechEngine = simRunner
	if cfg.Engine.Mode == "sherpa" {
		asrMode = "sherpa"
		eng = sherpaRunner
	}

	ttsMgr := tts.NewTTSManager(sherpaRunner, dsp)
	webMic := audio.NewWebSocketSource()
	hostMic := audio.NewHostNativeSource(cfg.Audio.SampleRate, cfg.Audio.ChunkSamples)

	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		cfg:           cfg,
		dsp:           dsp,
		ringBuffer:    ringBuf,
		stateMachine:  sm,
		engine:        eng,
		sherpaEngine:  sherpaRunner,
		simEngine:     simRunner,
		asrEngineMode: asrMode,
		matcher:       matcher,
		ttsMgr:        ttsMgr,
		webMicSource:  webMic,
		hostMicSource: hostMic,
		activeSource:  "web_ui_mic",
		audioInChan:   make(chan []float32, 200),
		ctx:           ctx,
		cancel:        cancel,
		clients:       make(map[*websocket.Conn]*sync.Mutex),
	}

	// Listen for state transitions and broadcast them
	sm.AddListener(func(ev statemachine.StateEvent) {
		ttsModel, isNeural := s.ttsMgr.ActiveModel()
		s.BroadcastJSON(map[string]interface{}{
			"event": "voice.state",
			"data": map[string]interface{}{
				"from_state":      ev.FromState,
				"to_state":        ev.ToState,
				"trigger":         ev.Trigger,
				"timestamp":       ev.Timestamp.Format(time.RFC3339Nano),
				"extra":           ev.Data,
				"engine_mode":     s.engine.Name(),
				"asr_engine":      s.ASREngineMode(),
				"asr_engine_name": s.engineDisplayName(),
				"tts_model":       ttsModel,
				"tts_is_neural":   isNeural,
			},
		})
	})

	return s, nil
}

// Start begins listening and serving HTTP & WebSocket requests.
func (s *Server) Start() error {
	// Start audio processing loop
	go s.audioPipelineLoop(s.ctx)

	// Connect default web mic source
	_ = s.webMicSource.Start(s.ctx, s.audioInChan)

	mux := http.NewServeMux()

	// Static Web UI
	mux.Handle("/", http.FileServer(http.Dir(s.cfg.StaticDir)))

	// WebSocket IPC
	mux.HandleFunc("/ws", s.handleWebSocket)

	// REST APIs
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/catalog", s.handleCatalog)
	mux.HandleFunc("/api/voices", s.handleVoices)
	mux.HandleFunc("/api/models", s.handleModels)
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
	if s.cancel != nil {
		s.cancel()
	}
	if s.hostMicSource != nil {
		_ = s.hostMicSource.Stop()
	}
	if s.wavSource != nil {
		_ = s.wavSource.Stop()
	}
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// SwitchSource changes the active audio input source.
func (s *Server) SwitchSource(source string) {
	s.sourceMu.Lock()
	defer s.sourceMu.Unlock()

	if s.activeSource == source {
		return
	}

	// Stop previous non-web sources
	if s.activeSource == "host_native_mic" && s.hostMicSource != nil {
		_ = s.hostMicSource.Stop()
	} else if s.activeSource == "wav_file_injection" && s.wavSource != nil {
		_ = s.wavSource.Stop()
	}

	s.activeSource = source
	devName := ""

	switch source {
	case "host_native_mic":
		if s.hostMicSource != nil {
			_ = s.hostMicSource.Start(s.ctx, s.audioInChan)
			devName = s.hostMicSource.DeviceName()
		}
	case "wav_file_injection":
		wavPath := filepath.Join("data", "test_samples", "voice_test.wav")
		s.wavSource = audio.NewWavFileSource(wavPath, s.cfg.Audio.SampleRate, s.cfg.Audio.ChunkSamples, true)
		_ = s.wavSource.Start(s.ctx, s.audioInChan)
		devName = "voice_test.wav"
	default:
		// web_ui_mic
		devName = "browser_mic"
	}

	log.Printf("[VoxLab] Active audio source switched to: %s (%s)", source, devName)

	s.BroadcastJSON(map[string]interface{}{
		"event": "audio.source_changed",
		"data": map[string]interface{}{
			"source":      source,
			"device_name": devName,
		},
	})
}

// SetASREngineMode updates the active ASR engine between 'sherpa', 'browser', and 'sim'.
func (s *Server) SetASREngineMode(mode string) error {
	s.engineMu.Lock()
	defer s.engineMu.Unlock()

	switch mode {
	case "sherpa":
		s.asrEngineMode = "sherpa"
		s.engine = s.sherpaEngine
	case "browser":
		s.asrEngineMode = "browser"
	case "sim":
		s.asrEngineMode = "sim"
		s.engine = s.simEngine
	default:
		return fmt.Errorf("unknown ASR engine mode: %s", mode)
	}

	log.Printf("[VoxLab] Active ASR engine switched to: %s (%s)", s.asrEngineMode, s.engineDisplayName())

	s.BroadcastJSON(map[string]interface{}{
		"event": "asr.engine_changed",
		"data": map[string]interface{}{
			"engine": s.asrEngineMode,
			"name":   s.engineDisplayName(),
		},
	})
	return nil
}

// ASREngineMode returns the current ASR engine mode.
func (s *Server) ASREngineMode() string {
	s.engineMu.RLock()
	defer s.engineMu.RUnlock()
	return s.asrEngineMode
}

func (s *Server) engineDisplayName() string {
	switch s.asrEngineMode {
	case "sherpa":
		return "Sherpa-ONNX (Offline Neural)"
	case "browser":
		return "Browser Web Speech API"
	case "sim":
		return "Testbench Simulator"
	default:
		return s.asrEngineMode
	}
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

// downsampleWave extracts representative points across the chunk for UI rendering.
func downsampleWave(samples []float32, targetLen int) []float32 {
	if len(samples) == 0 {
		return nil
	}
	if len(samples) <= targetLen {
		return samples
	}
	out := make([]float32, targetLen)
	step := float64(len(samples)) / float64(targetLen)
	for i := 0; i < targetLen; i++ {
		idx := int(float64(i) * step)
		if idx >= len(samples) {
			idx = len(samples) - 1
		}
		out[i] = samples[idx]
	}
	return out
}

// processIncomingAudio handles a single 30ms audio chunk through all active filters and gates.
func (s *Server) processIncomingAudio(rawChunk []float32) {
	// Step 1: Preprocessing & Noise Gating
	cleanChunk, rms, dbfs, passedGate := s.dsp.ProcessChunk(rawChunk)

	// Stream clean audio chunk to clients for live headphone monitoring if remote input active
	s.sourceMu.Lock()
	currSource := s.activeSource
	s.sourceMu.Unlock()
	if currSource != "web_ui_mic" && s.IsMonitorEnabled() {
		pcmBytes := audio.Float32ToBytesPCM(cleanChunk)
		s.BroadcastBinary(pcmBytes)
	}

	// Broadcast meter update with downsampled waveform at a steady ~30Hz (33ms)
	s.meterMu.Lock()
	now := time.Now()
	shouldBroadcast := now.Sub(s.lastMeterTime) >= 33*time.Millisecond
	if shouldBroadcast {
		s.lastMeterTime = now
	}
	s.meterMu.Unlock()

	if shouldBroadcast {
		// Use raw acoustic chunk for visualizer so soft speech and background noise are visible,
		// while passed_gate indicates when noise floor gating is active.
		wave := downsampleWave(rawChunk, 64)
		s.BroadcastJSON(map[string]interface{}{
			"event": "audio.meter",
			"data": map[string]interface{}{
				"rms":         rms,
				"dbfs":        dbfs,
				"passed_gate": passedGate,
				"echo_muted":  s.dsp.IsEchoMuted(),
				"source":      s.activeSource,
				"wave":        wave,
				"clean_wave":  downsampleWave(cleanChunk, 64),
				"agc_gain":    s.dsp.CurrentGain(),
				"highpass_on": s.dsp.IsHighPassEnabled(),
				"agc_on":      s.dsp.IsAGCEnabled(),
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
	s.engineMu.RLock()
	currentASRMode := s.asrEngineMode
	s.engineMu.RUnlock()

	// If browser Web Speech API is active, the browser performs recognition directly in client;
	// Host daemon skips ASR chunk decoding to avoid duplicate/competing transcripts.
	if currentASRMode == "browser" {
		return
	}

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
				s.BroadcastJSON(map[string]interface{}{
					"event": "transcript.partial",
					"data": map[string]interface{}{
						"mode":       "dictation",
						"transcript": asrRes.Transcript,
						"tokens":     asrRes.Tokens,
					},
				})
			} else {
				// Phrase segment completed during a pause in speech.
				// Dictation mode continues until explicitly stopped by user!
				s.dictationDraft = AppendDictationPhrase(s.dictationDraft, asrRes.Transcript)
				words := len(intent.Tokenize(s.dictationDraft))
				s.BroadcastJSON(map[string]interface{}{
					"event": "annotation.draft",
					"data": map[string]interface{}{
						"target_id":  s.dictationTarget,
						"draft_text": s.dictationDraft,
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

	connMu := &sync.Mutex{}
	s.clientsMu.Lock()
	s.clients[conn] = connMu
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()
	}()

	// Send initial state snapshot safely under connMu
	ttsModel, isNeural := s.ttsMgr.ActiveModel()
	connMu.Lock()
	_ = conn.WriteJSON(map[string]interface{}{
		"event": "voice.state",
		"data": map[string]interface{}{
			"to_state":        s.stateMachine.Current(),
			"timestamp":       time.Now().Format(time.RFC3339),
			"active_source":   s.activeSource,
			"engine_mode":     s.engine.Name(),
			"asr_engine":      s.ASREngineMode(),
			"asr_engine_name": s.engineDisplayName(),
			"tts_model":       ttsModel,
			"tts_is_neural":   isNeural,
		},
	})
	connMu.Unlock()

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
					s.dictationDraft = ""
					s.stateMachine.StartDictation(target)
				} else {
					s.stateMachine.StartCommand()
				}

			case "cancel":
				s.SetPlaybackActive(false)
				s.stateMachine.Cancel()
				s.engine.ResetASR()

			case "playback_status":
				playing, _ := req["playing"].(bool)
				s.SetPlaybackActive(playing)

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
					streaming := true
					if st, ok := p["streaming"].(bool); ok {
						streaming = st
					}
					go s.executeTTS(text, voice, speed, streaming)
				}

			case "set_source":
				source, _ := req["source"].(string)
				s.SwitchSource(source)

			case "set_asr_engine":
				engName, _ := req["engine"].(string)
				if engName != "" {
					if err := s.SetASREngineMode(engName); err != nil {
						log.Printf("[Error] Failed setting ASR engine %s: %v", engName, err)
					}
				}

			case "set_noise_gate":
				if th, ok := req["threshold_dbfs"].(float64); ok {
					s.dsp.SetGateThreshold(th)
					log.Printf("[DSP] Noise gate threshold set to %.1f dBFS", th)
				}

			case "set_kws_threshold":
				if th, ok := req["threshold"].(float64); ok {
					s.cfg.KWS.Threshold = th
					log.Printf("[KWS] Wake-word threshold set to %.2f", th)
				}

			case "set_kws_enable":
				if en, ok := req["enabled"].(bool); ok {
					s.cfg.KWS.Enabled = en
					log.Printf("[KWS] Continuous wake-word enabled: %v", en)
				}

			case "set_kws_keyword":
				if kw, ok := req["keyword"].(string); ok {
					s.cfg.KWS.Keyword = kw
					log.Printf("[KWS] Wake-word phrase set to: %q", kw)
				}

			case "set_agc":
				if en, ok := req["enabled"].(bool); ok {
					s.dsp.SetAGCEnabled(en)
					log.Printf("[DSP] AGC enabled: %v", en)
				}

			case "set_highpass":
				if en, ok := req["enabled"].(bool); ok {
					s.dsp.SetHighPassEnabled(en)
					log.Printf("[DSP] High-pass filter (80Hz de-rumble) enabled: %v", en)
				}

			case "set_monitor":
				if en, ok := req["enabled"].(bool); ok {
					s.SetMonitorEnabled(en)
					log.Printf("[Server] Headphone audio monitor: enabled=%v", en)
				}

			case "inject_text":
				text, _ := req["text"].(string)
				mode, _ := req["mode"].(string)
				if mode == "dictation" {
					if text == "" {
						s.dictationDraft = ""
					} else {
						s.dictationDraft = AppendDictationPhrase(s.dictationDraft, text)
					}
					s.BroadcastJSON(map[string]interface{}{
						"event": "annotation.draft",
						"data": map[string]interface{}{
							"target_id":  "injected",
							"draft_text": s.dictationDraft,
							"word_count": len(intent.Tokenize(s.dictationDraft)),
						},
					})
				} else {
					s.BroadcastJSON(map[string]interface{}{
						"event": "transcript.final",
						"data": map[string]interface{}{
							"mode":       "command",
							"transcript": text,
						},
					})
					s.handleFinalCommandTranscript(text)
				}

			case "set_tts_model":
				model, _ := req["model"].(string)
				if model != "" {
					if err := s.ttsMgr.SetModel(model); err != nil {
						log.Printf("[Error] Failed setting TTS model %s: %v", model, err)
					} else {
						activeModel, isNeural := s.ttsMgr.ActiveModel()
						voices := s.ttsMgr.Voices()
						installed := s.ttsMgr.InstalledModels()
						s.BroadcastJSON(map[string]interface{}{
							"event": "tts.model_changed",
							"data": map[string]interface{}{
								"model":            activeModel,
								"model_id":         model,
								"is_neural":        isNeural,
								"installed_models": installed,
								"voices":           voices,
							},
						})
					}
				}
			}
		}
	}
}

// SetPlaybackActive coordinates half-duplex echo suppression with actual speaker playback.
func (s *Server) SetPlaybackActive(playing bool) {
	s.ttsMgr.SetSpeaking(playing)
	s.stateMachine.SetSpeaking(playing)
	log.Printf("[TTS] Speaker playback status changed: playing=%v (echo_muted=%v)", playing, s.dsp.IsEchoMuted())
}

// executeTTS executes Kokoro TTS and broadcasts audio waveform and metrics.
func (s *Server) executeTTS(text, voice string, speed float64, streaming bool) {
	if text == "" {
		return
	}
	if voice == "" {
		voice = s.cfg.TTS.DefaultVoice
	}

	// Synthesis is model computation only (no sound is playing from speakers yet).
	// We broadcast tts.started so the UI can show progress, but mic remains unmuted.
	ttsModel, isNeural := s.ttsMgr.ActiveModel()
	s.BroadcastJSON(map[string]interface{}{
		"event": "tts.started",
		"data": map[string]interface{}{
			"text":      text,
			"voice":     voice,
			"speed":     speed,
			"model":     ttsModel,
			"is_neural": isNeural,
			"streaming": streaming,
		},
	})

	var (
		firstChunkLatency int64
		chunkCount        int
	)

	onChunkCb := func(chunk engine.TTSChunk) error {
		chunkCount++
		if chunkCount == 1 {
			firstChunkLatency = chunk.LatencyMs
		}
		chunkWavB64 := base64.StdEncoding.EncodeToString(chunk.WAVBytes)
		s.BroadcastJSON(map[string]interface{}{
			"event": "tts.chunk",
			"data": map[string]interface{}{
				"index":        chunk.Index,
				"is_last":      chunk.IsLast,
				"duration_sec": chunk.DurationSec,
				"latency_ms":   chunk.LatencyMs,
				"sample_rate":  chunk.SampleRate,
				"wav_base64":   chunkWavB64,
			},
		})
		return nil
	}

	var (
		ttsRes *engine.TTSResult
		err    error
	)

	req := engine.TTSRequest{
		Text:       text,
		Voice:      voice,
		Speed:      speed,
		SampleRate: s.cfg.TTS.SampleRate,
	}

	if streaming {
		ttsRes, err = s.ttsMgr.SynthesizeStream(req, onChunkCb)
	} else {
		ttsRes, err = s.ttsMgr.Synthesize(req)
	}

	if err != nil {
		log.Printf("[Error] TTS error: %v", err)
		return
	}

	wavB64 := base64.StdEncoding.EncodeToString(ttsRes.WAVBytes)

	s.BroadcastJSON(map[string]interface{}{
		"event": "tts.finished",
		"data": map[string]interface{}{
			"text":                   text,
			"voice":                  voice,
			"model":                  ttsModel,
			"is_neural":              isNeural,
			"duration_sec":           ttsRes.DurationSec,
			"latency_ms":             ttsRes.LatencyMs,
			"first_chunk_latency_ms": firstChunkLatency,
			"chunks_count":           chunkCount,
			"sample_rate":            ttsRes.SampleRate,
			"streaming":              streaming,
			"wav_base64":             wavB64,
		},
	})

	// Failsafe timer: if client starts playback but drops connection or fails to report ended,
	// ensure echo suppression is automatically cleared after audio duration + margin.
	go func(durSec float64) {
		time.Sleep(time.Duration((durSec+4.0)*1000) * time.Millisecond)
		if s.ttsMgr.IsSpeaking() {
			log.Printf("[TTS] Failsafe timer: automatically clearing echo suppression after %.1fs", durSec)
			s.SetPlaybackActive(false)
		}
	}(ttsRes.DurationSec)
}

// SetMonitorEnabled toggles streaming raw PCM audio chunks to clients for headphone monitoring.
func (s *Server) SetMonitorEnabled(enabled bool) {
	s.monitorMu.Lock()
	defer s.monitorMu.Unlock()
	s.monitorEnabled = enabled
}

// IsMonitorEnabled returns whether client headphone monitoring is currently active.
func (s *Server) IsMonitorEnabled() bool {
	s.monitorMu.RLock()
	defer s.monitorMu.RUnlock()
	return s.monitorEnabled
}

// BroadcastJSON sends a JSON event to all connected WebSocket clients.
func (s *Server) BroadcastJSON(v interface{}) {
	s.clientsMu.RLock()
	type clientTarget struct {
		conn *websocket.Conn
		mu   *sync.Mutex
	}
	targets := make([]clientTarget, 0, len(s.clients))
	for client, mu := range s.clients {
		targets = append(targets, clientTarget{conn: client, mu: mu})
	}
	s.clientsMu.RUnlock()

	for _, t := range targets {
		t.mu.Lock()
		_ = t.conn.WriteJSON(v)
		t.mu.Unlock()
	}
}

// BroadcastBinary sends raw binary message (e.g. PCM audio) to all connected WebSocket clients.
func (s *Server) BroadcastBinary(data []byte) {
	s.clientsMu.RLock()
	type clientTarget struct {
		conn *websocket.Conn
		mu   *sync.Mutex
	}
	targets := make([]clientTarget, 0, len(s.clients))
	for client, mu := range s.clients {
		targets = append(targets, clientTarget{conn: client, mu: mu})
	}
	s.clientsMu.RUnlock()

	for _, t := range targets {
		t.mu.Lock()
		_ = t.conn.WriteMessage(websocket.BinaryMessage, data)
		t.mu.Unlock()
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
	reqModel := r.URL.Query().Get("model")
	var voices []engine.VoiceProfile
	if reqModel != "" {
		voices = s.ttsMgr.VoicesForModel(reqModel)
	} else {
		voices = s.ttsMgr.Voices()
	}
	modelName, isNeural := s.ttsMgr.ActiveModel()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"model":            modelName,
		"is_neural":        isNeural,
		"installed_models": s.ttsMgr.InstalledModels(),
		"voices":           voices,
	})
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost {
		var body struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body.Model != "" {
			if err := s.ttsMgr.SetModel(body.Model); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			activeModel, isNeural := s.ttsMgr.ActiveModel()
			voices := s.ttsMgr.Voices()
			installed := s.ttsMgr.InstalledModels()
			s.BroadcastJSON(map[string]interface{}{
				"event": "tts.model_changed",
				"data": map[string]interface{}{
					"model":            activeModel,
					"model_id":         body.Model,
					"is_neural":        isNeural,
					"installed_models": installed,
					"voices":           voices,
				},
			})
		}
	}
	modelName, isNeural := s.ttsMgr.ActiveModel()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"model":            modelName,
		"is_neural":        isNeural,
		"installed_models": s.ttsMgr.InstalledModels(),
	})
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
