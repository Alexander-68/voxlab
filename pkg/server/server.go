package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"voxlab/pkg/audio"
	"voxlab/pkg/config"
	"voxlab/pkg/engine"
	"voxlab/pkg/evaluation"
	"voxlab/pkg/intent"
	"voxlab/pkg/statemachine"
	"voxlab/pkg/tts"
)

// upgrader keeps gorilla's default CheckOrigin: browsers may only connect from the same origin.
var upgrader = websocket.Upgrader{}

// Server is the main VoxLab web and voice API server.
type Server struct {
	cfg           *config.AppConfig
	httpServer    *http.Server
	dsp           *audio.DSPProcessor
	enhancer      audio.SpeechEnhancer
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
	asrChan       chan asrWorkItem

	// turn identifies the current listening turn. Bumped before the recognizer is reset, so
	// audio and results from an abandoned turn can be recognized and dropped.
	turn      atomic.Uint64
	asrTurnMu sync.Mutex // serializes recognizer calls with turn resets

	asrEngineMode string // "sherpa", "browser", "sim"
	engineMu      sync.RWMutex
	sherpaEngine  engine.SpeechEngine
	simEngine     engine.SpeechEngine

	ctx    context.Context
	cancel context.CancelFunc

	clientsMu sync.RWMutex
	clients   map[*wsClient]struct{}

	monitorMu      sync.RWMutex
	monitorEnabled bool

	dictMu          sync.Mutex // guards dictation draft/target (written by WS handlers and the ASR loop)
	dictationDraft  string
	dictationTarget string

	inputLangMu   sync.RWMutex
	inputLanguage string

	meterMu       sync.Mutex
	lastMeterTime time.Time

	evalMu       sync.Mutex
	evalRunning  bool
	evalCancel   context.CancelFunc
	evalTier     string
	evalProgress int
	evalTotal    int
	lastEvalItem *evaluation.ItemResult
	lastReport   *evaluation.SuiteReport
	evalRunner   *evaluation.EvalRunner
}

type asrWorkItem struct {
	chunk       []float32
	turn        uint64
	state       statemachine.State
	isDictation bool
	flush       bool // finalize pending speech and end the dictation session
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
	enhancer := audio.NewSpeechEnhancer(filepath.Join("bin"), cfg.Engine.ModelDir, cfg.Audio.Enhancer)

	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		cfg:           cfg,
		dsp:           dsp,
		enhancer:      enhancer,
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
		asrChan:       make(chan asrWorkItem, 200),
		inputLanguage: cfg.Audio.InputLanguage,
		ctx:           ctx,
		cancel:        cancel,
		clients:       make(map[*wsClient]struct{}),
	}

	// Surface each second-pass correction so the bench shows first-pass vs refined text and its cost.
	sherpaRunner.OnRefine = func(isDictation bool, firstPass, refined string, latencyMs int64) {
		mode := "command"
		if isDictation {
			mode = "dictation"
		}
		log.Printf("[ASR] Second pass (%d ms): %q -> %q", latencyMs, firstPass, refined)
		s.BroadcastJSON(map[string]interface{}{
			"event": "transcript.refined",
			"data": map[string]interface{}{
				"mode":       mode,
				"first_pass": firstPass,
				"transcript": refined,
				"latency_ms": latencyMs,
			},
		})
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
				"engine_mode":     s.currentEngine().Name(),
				"asr_engine":      s.ASREngineMode(),
				"asr_engine_name": s.engineDisplayName(),
				"tts_model":       ttsModel,
				"tts_is_neural":   isNeural,
				"enhancer_on":     s.enhancer != nil && s.enhancer.IsEnabled(),
				"enhancer_type":   s.enhancerModelType(),
				"enhancer_name":   s.enhancerName(),
			},
		})
	})

	s.evalRunner = evaluation.NewEvalRunner(s.HandleWebSocket)
	s.evalRunner.Describe = s.evalRunConfig
	return s, nil
}

// StartBackgroundLoops starts the decoupled audio and ASR processing goroutines (used in tests and evaluation).
func (s *Server) StartBackgroundLoops() {
	go s.audioPipelineLoop(s.ctx)
	go s.asrPipelineLoop(s.ctx)
	_ = s.webMicSource.Start(s.ctx, s.audioInChan)
}

func isLoopbackName(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

// loopbackHostGuard refuses non-loopback Host headers while bound to loopback. Same-origin checks
// alone pass under DNS rebinding, where evil.example resolves to 127.0.0.1 and is its own origin.
func (s *Server) loopbackHostGuard(next http.Handler) http.Handler {
	if !isLoopbackName(s.cfg.Host) {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if !isLoopbackName(host) {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Start begins listening and serving HTTP & WebSocket requests.
func (s *Server) Start() error {
	s.StartBackgroundLoops()

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.routes(),
	}

	if !isLoopbackName(s.cfg.Host) {
		log.Printf("[Warning] Bound to %s: the host microphone and all controls are reachable from the network without authentication", s.cfg.Host)
	}
	log.Printf("[VoxLab] Server listening on http://%s (Engine: %s, UI: %s)", addr, s.engine.Name(), s.cfg.StaticDir)
	return s.httpServer.ListenAndServe()
}

// routes builds the HTTP handler with its cross-origin and Host protections.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	// Static Web UI & Evaluation Assets
	mux.Handle("/data/", http.StripPrefix("/data/", http.FileServer(http.Dir("data"))))
	mux.Handle("/", http.FileServer(http.Dir(s.cfg.StaticDir)))

	// WebSocket IPC
	mux.HandleFunc("/ws", s.handleWebSocket)

	// REST APIs
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/catalog", s.handleCatalog)
	mux.HandleFunc("/api/voices", s.handleVoices)
	mux.HandleFunc("/api/models", s.handleModels)
	mux.HandleFunc("/api/tts", s.handleTTS)
	mux.HandleFunc("/api/eval/status", s.handleEvalStatus)
	mux.HandleFunc("/api/eval/start", s.handleEvalStart)
	mux.HandleFunc("/api/eval/cancel", s.handleEvalCancel)
	mux.HandleFunc("/api/eval/report", s.handleEvalReport)

	return s.loopbackHostGuard(http.NewCrossOriginProtection().Handler(mux))
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
	if s.enhancer != nil {
		_ = s.enhancer.Close()
	}
	// Kill the warm ASR daemon; child processes outlive us on Windows.
	if c, ok := s.sherpaEngine.(io.Closer); ok {
		_ = c.Close()
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

	if s.enhancer != nil {
		s.enhancer.Reset()
	}

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
	old := s.engine
	switch mode {
	case "sherpa":
		s.engine = s.sherpaEngine
	case "browser":
	case "sim":
		s.engine = s.simEngine
	default:
		s.engineMu.Unlock()
		return fmt.Errorf("unknown ASR engine mode: %s", mode)
	}
	s.asrEngineMode = mode
	s.engineMu.Unlock()
	old.ResetASR()

	name := s.engineDisplayName()
	log.Printf("[VoxLab] Active ASR engine switched to: %s (%s)", mode, name)

	s.BroadcastJSON(map[string]interface{}{
		"event": "asr.engine_changed",
		"data": map[string]interface{}{
			"engine": mode,
			"name":   name,
		},
	})
	return nil
}

func (s *Server) currentEngine() engine.SpeechEngine {
	s.engineMu.RLock()
	defer s.engineMu.RUnlock()
	return s.engine
}

func (s *Server) currentSource() string {
	s.sourceMu.Lock()
	defer s.sourceMu.Unlock()
	return s.activeSource
}

// ASREngineMode returns the current ASR engine mode.
func (s *Server) ASREngineMode() string {
	s.engineMu.RLock()
	defer s.engineMu.RUnlock()
	return s.asrEngineMode
}

func (s *Server) engineDisplayName() string {
	switch mode := s.ASREngineMode(); mode {
	case "sherpa":
		return "Sherpa-ONNX (Offline Neural)"
	case "browser":
		return "Browser Web Speech API"
	case "sim":
		return "Testbench Simulator"
	default:
		return mode
	}
}

// asrModelInfo describes the sherpa first/second-pass model setup (nil without a sherpa runner).
func (s *Server) asrModelInfo() map[string]interface{} {
	if r, ok := s.sherpaEngine.(*engine.SherpaRunner); ok {
		return r.ASRModelInfo()
	}
	return nil
}

func (s *Server) broadcastASRModelInfo() {
	s.BroadcastJSON(map[string]interface{}{"event": "asr.models_changed", "data": s.asrModelInfo()})
}

func (s *Server) enhancerModelType() string {
	if s.enhancer == nil {
		return "none"
	}
	return s.enhancer.ModelType()
}

func (s *Server) enhancerName() string {
	if s.enhancer == nil {
		return "None"
	}
	return s.enhancer.Name()
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
	// Step 0: Neural Speech Enhancement (if enabled)
	var enhancedChunk []float32
	if s.enhancer != nil && s.enhancer.IsEnabled() {
		enhancedChunk = s.enhancer.ProcessChunk(rawChunk)
	} else {
		enhancedChunk = rawChunk
	}

	// Step 1: Preprocessing & Noise Gating
	cleanChunk, rms, dbfs, passedGate := s.dsp.ProcessChunk(enhancedChunk)

	// Stream clean audio chunk to clients for live headphone monitoring (all sources, so the monitor always plays what ASR hears)
	currSource := s.currentSource()
	if s.IsMonitorEnabled() {
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
				"rms":           rms,
				"dbfs":          dbfs,
				"passed_gate":   passedGate,
				"gate_disabled": !s.dsp.IsGateEnabled(),
				"echo_muted":    s.dsp.IsEchoMuted(),
				"source":        currSource,
				"wave":          wave,
				"clean_wave":    downsampleWave(cleanChunk, 64),
				"agc_gain":      s.dsp.CurrentGain(),
				"highpass_on":   s.dsp.IsHighPassEnabled(),
				"agc_on":        s.dsp.IsAGCEnabled(),
				"enhancer_on":   s.enhancer != nil && s.enhancer.IsEnabled(),
				"enhancer_type": s.enhancerModelType(),
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
		detected, kw, conf := s.currentEngine().DetectWakeWord(cleanChunk)
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

	// Step 3: Speech Recognition when in listening states.
	// In browser mode the Web Speech API recognizes in the client; skip host ASR.
	if s.ASREngineMode() == "browser" {
		return
	}
	if state != statemachine.StateCommandListen && state != statemachine.StateAnnotationListen {
		return
	}

	// Forward to the decoupled ASR worker so metering is never blocked by inference.
	select {
	case s.asrChan <- asrWorkItem{chunk: cleanChunk, turn: s.turn.Load(), state: state, isDictation: state == statemachine.StateAnnotationListen}:
	default:
		log.Printf("[Warning] ASR queue full, dropping audio chunk")
		s.BroadcastJSON(map[string]interface{}{"event": "asr.queue_overflow", "data": map[string]interface{}{}})
	}
}

// newTurn abandons the current turn: its queued audio and in-flight results are dropped.
func (s *Server) newTurn() {
	s.turn.Add(1)
	s.asrTurnMu.Lock()
	defer s.asrTurnMu.Unlock()
	s.currentEngine().ResetASR()
}

func (s *Server) asrPipelineLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case item, ok := <-s.asrChan:
			if !ok {
				return
			}
			s.handleASRWorkItem(item)
		}
	}
}

func (s *Server) handleASRWorkItem(item asrWorkItem) {
	// Chunks queued before a state change (endpoint, cancel, stop) belong to a finished turn.
	// The state check alone misses a cancel followed by a new turn in the same state.
	if s.stateMachine.Current() != item.state || item.turn != s.turn.Load() {
		return
	}

	s.asrTurnMu.Lock()
	if item.turn != s.turn.Load() {
		s.asrTurnMu.Unlock()
		return
	}
	eng := s.currentEngine()
	var (
		flushed string
		res     *engine.ASRResult
		err     error
	)
	if item.flush {
		flushed = eng.FlushASR()
	} else {
		res, err = eng.ProcessASRChunk(item.chunk, item.isDictation)
	}
	s.asrTurnMu.Unlock()

	// Recognition can take seconds with a second pass; the turn may have been cancelled meanwhile.
	if item.turn != s.turn.Load() {
		return
	}

	if item.flush {
		s.appendDictation(flushed)
		s.stateMachine.FinishReview(true)
		return
	}

	mode := "command"
	if item.isDictation {
		mode = "dictation"
	}
	if err != nil {
		log.Printf("[Error] %s ASR error: %v", mode, err)
		return
	}
	if res == nil {
		return
	}

	switch {
	case !res.IsFinal:
		s.BroadcastJSON(map[string]interface{}{
			"event": "transcript.partial",
			"data": map[string]interface{}{
				"mode":       mode,
				"transcript": res.Transcript,
				"tokens":     res.Tokens,
			},
		})
	case item.isDictation:
		// Segment endpointed on a pause; dictation continues until explicitly stopped.
		s.appendDictation(res.Transcript)
	default:
		s.BroadcastJSON(map[string]interface{}{
			"event": "transcript.final",
			"data": map[string]interface{}{
				"mode":       "command",
				"transcript": res.Transcript,
			},
		})
		s.handleFinalCommandTranscript(res.Transcript)
	}
}

// appendDictation adds a recognized phrase to the draft and broadcasts it.
// Empty text (noise, typing) just clears the live partial so the UI returns to listening.
func (s *Server) appendDictation(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		s.BroadcastJSON(map[string]interface{}{
			"event": "transcript.partial",
			"data":  map[string]interface{}{"mode": "dictation", "transcript": ""},
		})
		return
	}
	s.dictMu.Lock()
	s.dictationDraft = AppendDictationPhrase(s.dictationDraft, text)
	draft, target := s.dictationDraft, s.dictationTarget
	s.dictMu.Unlock()
	s.BroadcastJSON(map[string]interface{}{
		"event": "annotation.draft",
		"data": map[string]interface{}{
			"target_id":  target,
			"draft_text": draft,
			"word_count": len(intent.Tokenize(draft)),
		},
	})
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

// HandleWebSocket exposes WebSocket handling for integration and evaluation harnesses.
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	s.handleWebSocket(w, r)
}

// handleWebSocket manages client connections and control messaging.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[Error] WS Upgrade: %v", err)
		return
	}
	client := newWSClient(conn)
	defer client.close()
	go client.writeLoop()

	s.clientsMu.Lock()
	s.clients[client] = struct{}{}
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, client)
		s.clientsMu.Unlock()
	}()

	// Send initial state snapshot
	ttsModel, isNeural := s.ttsMgr.ActiveModel()
	asrInfo := s.asrModelInfo()
	stateMsg := map[string]interface{}{
		"event": "voice.state",
		"data": map[string]interface{}{
			"to_state":        s.stateMachine.Current(),
			"timestamp":       time.Now().Format(time.RFC3339),
			"active_source":   s.currentSource(),
			"engine_mode":     s.currentEngine().Name(),
			"asr_engine":      s.ASREngineMode(),
			"asr_engine_name": s.engineDisplayName(),
			"tts_model":       ttsModel,
			"tts_is_neural":   isNeural,
			"enhancer_on":     s.enhancer != nil && s.enhancer.IsEnabled(),
			"enhancer_type":   s.enhancerModelType(),
			"enhancer_name":   s.enhancerName(),
			"asr_models":      asrInfo,
		},
	}
	if data, err := json.Marshal(stateMsg); err == nil {
		client.enqueue(wsFrame{websocket.TextMessage, data})
	}

	evalHarness := false // set by the bench harness; only this read loop touches it

	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}

		if messageType == websocket.BinaryMessage {
			// Binary PCM audio chunk from Web UI mic (16-bit PCM little endian).
			// During a bench run only the harness feeds audio, so a live mic cannot mix in.
			if s.currentSource() == "web_ui_mic" && (evalHarness || !s.EvalStatus().Running) {
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
					s.dictMu.Lock()
					s.dictationTarget = target
					s.dictationDraft = ""
					s.dictMu.Unlock()
					s.newTurn()
					s.stateMachine.StartDictation(target)
				} else {
					s.newTurn()
					s.stateMachine.StartCommand()
				}

			case "cancel":
				s.SetPlaybackActive(false)
				s.turn.Add(1) // before the state change: nothing from this turn may land after it
				s.stateMachine.Cancel()
				s.newTurn()

			case "stop":
				// End dictation but keep speech not yet endpointed. Queued behind pending
				// audio so the recognizer sees every chunk before finalizing, which is
				// why this keeps the turn rather than starting a new one.
				if st := s.stateMachine.Current(); st == statemachine.StateAnnotationListen {
					s.asrChan <- asrWorkItem{turn: s.turn.Load(), state: st, isDictation: true, flush: true}
				} else {
					s.stateMachine.Cancel()
					s.newTurn()
				}

			case "eval_session":
				// Marks this connection as the bench harness and starts its item from clean
				// DSP state, so results do not depend on the order items run in.
				evalHarness = true
				s.dsp.Reset()
				if s.enhancer != nil {
					s.enhancer.Reset()
				}

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

			case "set_asr_model":
				name, _ := req["model"].(string)
				if r, ok := s.sherpaEngine.(*engine.SherpaRunner); ok {
					if err := r.SetStreamingModel(name); err != nil {
						log.Printf("[Error] Failed switching streaming ASR model: %v", err)
					} else {
						log.Printf("[ASR] Streaming model switched to %s", name)
					}
					s.broadcastASRModelInfo()
				}

			case "set_second_pass":
				mode, _ := req["mode"].(string)
				enabled, _ := req["enabled"].(bool)
				if r, ok := s.sherpaEngine.(*engine.SherpaRunner); ok {
					switch mode {
					case "command":
						r.SetSecondPassCommand(enabled)
						log.Printf("[ASR] Second pass for commands: %v", enabled)
					case "dictation":
						r.SetSecondPassDictation(enabled)
						log.Printf("[ASR] Second pass for dictation: %v", enabled)
					default:
						r.SetSecondPass(enabled)
						log.Printf("[ASR] Second pass global: %v", enabled)
					}
					s.broadcastASRModelInfo()
				}

			case "set_noise_gate":
				if th, ok := req["threshold_dbfs"].(float64); ok {
					s.dsp.SetGateThreshold(th)
					if s.dsp.IsGateEnabled() {
						log.Printf("[DSP] Noise gate threshold set to %.1f dBFS", th)
					} else {
						log.Printf("[DSP] Noise gate set to completely OFF (bypassed)")
					}
				}

			case "set_gate_hold":
				if ms, ok := req["hold_ms"].(float64); ok {
					s.dsp.SetGateHold(ms)
					log.Printf("[DSP] Noise gate hold set to %.0f ms", ms)
				}

			case "set_gate_floor":
				if db, ok := req["floor_db"].(float64); ok {
					s.dsp.SetGateFloor(db)
					log.Printf("[DSP] Noise gate floor set to %.0f dB", db)
				}

			case "set_kws_threshold":
				if th, ok := req["threshold"].(float64); ok {
					s.cfg.KWS.Threshold = th
					log.Printf("[KWS] Wake-word threshold set to %.2f", th)
				}

			case "set_language":
				lang, _ := req["language"].(string)
				if lang != "" {
					s.SetInputLanguage(lang)
					log.Printf("[ASR] Input language set to: %s", lang)
					s.BroadcastJSON(map[string]interface{}{
						"event": "audio.language_changed",
						"data": map[string]interface{}{
							"language": lang,
						},
					})
				}

			case "eval.start":
				tier, _ := req["tier"].(string)
				go func() {
					if err := s.StartEvaluation(tier); err != nil {
						log.Printf("[Evaluation] Could not start evaluation: %v", err)
					}
				}()

			case "eval.cancel":
				s.CancelEvaluation()

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
					s.dictMu.Lock()
					if text == "" {
						s.dictationDraft = ""
					} else {
						s.dictationDraft = AppendDictationPhrase(s.dictationDraft, text)
					}
					draft := s.dictationDraft
					s.dictMu.Unlock()
					s.BroadcastJSON(map[string]interface{}{
						"event": "annotation.draft",
						"data": map[string]interface{}{
							"target_id":  "injected",
							"draft_text": draft,
							"word_count": len(intent.Tokenize(draft)),
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

			case "set_speech_enhancer":
				enabled, _ := req["enabled"].(bool)
				modelType, _ := req["model"].(string)
				if s.enhancer != nil {
					if modelType != "" {
						_ = s.enhancer.SwitchModel(modelType)
						s.cfg.Audio.Enhancer.ModelType = s.enhancer.ModelType()
					}
					s.enhancer.SetEnabled(enabled)
					s.cfg.Audio.Enhancer.Enabled = s.enhancer.IsEnabled()
					log.Printf("[SpeechEnhancer] Setting updated: enabled=%v, model=%s (%s)",
						s.enhancer.IsEnabled(), s.enhancer.ModelType(), s.enhancer.Name())
					s.BroadcastJSON(map[string]interface{}{
						"event": "enhancer.changed",
						"data": map[string]interface{}{
							"enabled": s.enhancer.IsEnabled(),
							"model":   s.enhancer.ModelType(),
							"name":    s.enhancer.Name(),
						},
					})
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

// InputLanguage returns the active voice input language code.
func (s *Server) InputLanguage() string {
	s.inputLangMu.RLock()
	defer s.inputLangMu.RUnlock()
	if s.inputLanguage == "" {
		return "en-US"
	}
	return s.inputLanguage
}

// SetInputLanguage updates the active voice input language code.
func (s *Server) SetInputLanguage(lang string) {
	s.inputLangMu.Lock()
	defer s.inputLangMu.Unlock()
	s.inputLanguage = lang
	s.cfg.Audio.InputLanguage = lang
}

// SherpaEngine returns the Sherpa-ONNX speech engine instance.
func (s *Server) SherpaEngine() engine.SpeechEngine {
	s.engineMu.RLock()
	defer s.engineMu.RUnlock()
	return s.sherpaEngine
}

// ActiveEngine returns the currently active speech engine instance.
func (s *Server) ActiveEngine() engine.SpeechEngine {
	s.engineMu.RLock()
	defer s.engineMu.RUnlock()
	return s.engine
}

const (
	wsSendQueue    = 256 // about 4 s of meter and monitor traffic
	wsWriteTimeout = 5 * time.Second
)

type wsFrame struct {
	kind int
	data []byte
}

// wsClient owns all writes to one connection. Broadcasters only enqueue, so a client
// that stops reading can never stall the audio, ASR, or TTS goroutines.
type wsClient struct {
	conn      *websocket.Conn
	send      chan wsFrame
	done      chan struct{}
	closeOnce sync.Once
}

func newWSClient(conn *websocket.Conn) *wsClient {
	return &wsClient{conn: conn, send: make(chan wsFrame, wsSendQueue), done: make(chan struct{})}
}

// close also unblocks the connection's read loop, which then deregisters the client.
func (c *wsClient) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

// enqueue disconnects a client whose queue is full. The UI reconnects by itself,
// whereas silently dropping frames would lose state events.
func (c *wsClient) enqueue(f wsFrame) {
	select {
	case c.send <- f:
	default:
		log.Printf("[Warning] Disconnecting slow WebSocket client %s", c.conn.RemoteAddr())
		c.close()
	}
}

func (c *wsClient) writeLoop() {
	for {
		select {
		case <-c.done:
			return
		case f := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			if err := c.conn.WriteMessage(f.kind, f.data); err != nil {
				c.close()
				return
			}
		}
	}
}

func (s *Server) broadcast(f wsFrame) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()
	for c := range s.clients {
		c.enqueue(f)
	}
}

// BroadcastJSON sends a JSON event to all connected WebSocket clients.
func (s *Server) BroadcastJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("[Error] Failed encoding broadcast: %v", err)
		return
	}
	s.broadcast(wsFrame{websocket.TextMessage, data})
}

// BroadcastBinary sends raw binary message (e.g. PCM audio) to all connected WebSocket clients.
func (s *Server) BroadcastBinary(data []byte) {
	s.broadcast(wsFrame{websocket.BinaryMessage, data})
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
