# Graph Report - voxlab  (2026-11-15)

## Corpus Check
- 91 files · ~110,547 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 2 file(s) not represented in the graph (top: (none) 1, .css 1)

## Summary
- 850 nodes · 2499 edges · 44 communities (33 shown, 11 thin omitted)
- Extraction: 79% EXTRACTED · 21% INFERRED · 0% AMBIGUOUS · INFERRED: 515 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `978a0e5c`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- testing.T
- Evaluation Runner & Manifest
- TTSResult
- Embedded Voice Architecture Reference
- Web Client Voice Controller
- VoxLab Architecture & System Specifications
- Audio Capture Sources & Devices
- Simulator Engine & Test Generation
- Persistent Warm TTS Synthesis
- HTTP Web Server & Networking
- DSP Audio Preprocessor Controls
- warm_tts_windows.go
- Warm ASR Daemon Management
- ASR Audio Streaming Pipeline
- Command Catalog & Slot Extraction
- Voice Interaction State Machine
- Benchmark UI & Audio Helpers
- Neural Speech Denoisers & Enhancer
- TTS Model Selection & Warmup
- .processIncomingAudio
- Sherpa-ONNX Runner Integration
- Acoustic Perturbations & Corpus Generator
- Audio Visualizer & PCM Player
- ONNX Metadata Patching Utilities
- TTSManager
- EncodeWAV
- WebSocket Client Connection Lifecycle
- Evaluation Manifest Tier 1
- Evaluation Manifest Tier 2
- Q: Are the 16 inferred relationships involving setupTestServer() correct?
- Q: What connects voxlab, Ref Architectural Ideas, and Semantic Intent Router?
- Q: Why does Server connect HTTP Web Server & Networking to 14 other subsystems?
- Kokoro FP16 Model Profiles
- Kokoro INT8 Model Profiles
- Q: Why does SherpaRunner connect Sherpa-ONNX Runner Integration to 8 subsystems?
- Q: Should Speech Denoiser & Audio Generation be split?
- WPE WebKit IPC Integration
- Web Test Bench Architecture
- Bash Model Download Script
- VoxLab Root Module
- Q: Should Benchmark Evaluation & Metrics be split?
- Q: Why does SpeechEngine connect Simulator Engine & Test Generation to 8 subsystems?

## God Nodes (most connected - your core abstractions)
1. `Server` - 80 edges
2. `SherpaRunner` - 38 edges
3. `VoiceClient` - 35 edges
4. `setupTestServer()` - 31 edges
5. `NewServer()` - 29 edges
6. `SpeechEngine` - 23 edges
7. `StateMachine` - 21 edges
8. `DSPProcessor` - 20 edges
9. `windowsSpeechEnhancer` - 17 edges
10. `NewSherpaRunner()` - 17 edges

## Surprising Connections (you probably didn't know these)
- `Quick Wins` --references--> `escapeHtml()`  [INFERRED]
  review-261115.md → web/js/bench-ui.js
- `Top Issues` --references--> `LoadConfig()`  [INFERRED]
  review-261115.md → pkg/config/config.go
- `Cross-Cutting Concerns` --references--> `EncodeWAV()`  [INFERRED]
  review-261115.md → pkg/audio/wav.go
- `Top Issues` --references--> `getFreeTCPPort()`  [INFERRED]
  review-261115.md → pkg/engine/warm_asr.go
- `Top Issues` --references--> `toCString()`  [INFERRED]
  review-261115.md → pkg/engine/warm_tts_windows.go

## Import Cycles
- 2-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/server.go -> pkg/evaluation/smoke_test.go`
- 2-file cycle: `pkg/evaluation/tier2_test.go -> pkg/server/server.go -> pkg/evaluation/tier2_test.go`
- 2-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/smoke_test.go`
- 2-file cycle: `pkg/evaluation/tier2_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/tier2_test.go`
- 2-file cycle: `pkg/evaluation/smoke_test.go -> pkg/evaluation/tier2_test.go -> pkg/evaluation/smoke_test.go`
- 3-file cycle: `pkg/evaluation/smoke_test.go -> pkg/evaluation/tier2_test.go -> pkg/server/server.go -> pkg/evaluation/smoke_test.go`
- 3-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/server.go -> pkg/evaluation/tier2_test.go -> pkg/evaluation/smoke_test.go`
- 3-file cycle: `pkg/evaluation/smoke_test.go -> pkg/evaluation/tier2_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/smoke_test.go`
- 3-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/tier2_test.go -> pkg/evaluation/smoke_test.go`
- 4-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/tier2_test.go -> pkg/server/server.go -> pkg/evaluation/smoke_test.go`
- 4-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/server.go -> pkg/evaluation/tier2_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/smoke_test.go`

## Hyperedges (group relationships)
- **Voice Command Processing Architecture** — readme_audio_input_layer, readme_wake_word_detector, readme_voice_commands_pipeline, readme_semantic_intent_router, web_index_commands_tab [EXTRACTED 1.00]
- **Acoustic Benchmark & Evaluation Framework** — readme_acoustic_eval_bench, todo_acoustic_matrix_findings, todo_audio_resampler, web_index_acoustic_eval_tab [INFERRED 0.95]
- **Two-Pass Speech Recognition Pipeline** — todo_nemotron_35_upgrade, todo_parakeet_ultra_upgrade, readme_voice_dictations_pipeline, web_index_dictation_tab [INFERRED 0.95]
- **** — ref_synthetic_evaluation_bench_synthetic_benchmark_harness, ref_synthetic_evaluation_bench_two_pass_asr, ref_synthetic_evaluation_bench_two_tier_test_suite, ref_synthetic_evaluation_bench_tier1_golden_cases, ref_synthetic_evaluation_bench_quantitative_metrics [INFERRED]
- **** — ref_linux_wpe_voice_recipe_interaction_state_machine, ref_linux_wpe_voice_recipe_command_catalog_slots, ref_linux_wpe_voice_recipe_annotation_dictation_mode, ref_linux_wpe_voice_recipe_echo_control_policy [INFERRED]
- **** — ref_embedded_voice_architecture_recipe_silero_vad, ref_embedded_voice_architecture_recipe_wake_word_detector, ref_embedded_voice_architecture_recipe_streaming_asr_zipformer, ref_embedded_voice_architecture_recipe_semantic_intent_router, ref_embedded_voice_architecture_recipe_piper_tts [INFERRED]

## Communities (44 total, 11 thin omitted)

### Community 0 - "testing.T"
Cohesion: 0.07
Nodes (70): main(), main(), jobObjectExtendedLimit, NewDSPProcessor(), TestDSPHighPassAndAGC(), TestDSPNoiseGateAndEcho(), DefaultConfig(), TestTailWriterKeepsEnd() (+62 more)

### Community 1 - "Evaluation Runner & Manifest"
Cohesion: 0.10
Nodes (35): EvalItem, EvalManifest, DimensionGroup, ItemResult, LatencyPercentiles, LatencyStatistics, RunConfig, SNRCurvePoint (+27 more)

### Community 2 - "TTSResult"
Cohesion: 0.28
Nodes (4): TTSChunk, TTSRequest, TTSResult, float32SliceToTTSResult()

### Community 3 - "Embedded Voice Architecture Reference"
Cohesion: 0.07
Nodes (31): Piper TTS Pipeline, PipeWire Audio Server Subsystem, Semantic Intent Router & Distractor Suppression, Silero VAD & Energy Gating, Streaming Zipformer ASR, Host Voice Daemon Reference, Wake-Word Detector, Annotation Dictation Mode (+23 more)

### Community 5 - "VoxLab Architecture & System Specifications"
Cohesion: 0.07
Nodes (25): Ref Architectural Ideas, VoxLab App Specification, Dynamic Model Switching Architecture, Sherpa-ONNX Metadata Requirements, Acoustic Benchmark System, Dual Audio Input Layer, Kokoro TTS Engine, ONNX Metadata Helper Script (+17 more)

### Community 6 - "Audio Capture Sources & Devices"
Cohesion: 0.08
Nodes (15): AudioSource, tailWriter, detectWindowsDirectShowMic(), HostNativeSource, WavFileSource, WebSocketSource, NewWavFileSource(), NewWebSocketSource() (+7 more)

### Community 7 - "Simulator Engine & Test Generation"
Cohesion: 0.11
Nodes (4): SimulatorEngine, SpeechEngine, GenerateTier1Corpus(), LoadOrSynthesizeClip()

### Community 8 - "Persistent Warm TTS Synthesis"
Cohesion: 0.12
Nodes (11): stubWarmTTS, WarmTTS, windowsWarmTTS, optimizeStreamingClauses(), warmTTSKey(), getFreeTCPPort(), TestGetFreeTCPPort(), newPlatformWarmTTS() (+3 more)

### Community 9 - "HTTP Web Server & Networking"
Cohesion: 0.14
Nodes (3): isLoopbackName(), EvalState, Server

### Community 11 - "warm_tts_windows.go"
Cohesion: 0.14
Nodes (15): cDenoisedAudio, cDenoiserModelConfig, cDpdfNetModelConfig, cGtcrnModelConfig, cOnlineSpeechDenoiserConfig, cGeneratedAudio, cKittenModelConfig, cKokoroModelConfig (+7 more)

### Community 12 - "Warm ASR Daemon Management"
Cohesion: 0.19
Nodes (9): daemon, WarmASR, WarmOffline, killWithParent(), startDaemon(), killWithParent(), NewWarmASR(), StartWarmASR() (+1 more)

### Community 13 - "ASR Audio Streaming Pipeline"
Cohesion: 0.16
Nodes (5): ASRStream, ASRResult, containsCJK(), normalizeCase(), fakeEngine

### Community 14 - "Command Catalog & Slot Extraction"
Cohesion: 0.10
Nodes (25): CommandCatalog, CommandEntry, IntentDecision, MatchCandidate, TestRingBuffer(), NewRingBuffer(), DefaultCatalog(), LoadCatalog() (+17 more)

### Community 15 - "Voice Interaction State Machine"
Cohesion: 0.20
Nodes (4): State, StateMachine, asrWorkItem, StateEvent

### Community 16 - "Benchmark UI & Audio Helpers"
Cohesion: 0.13
Nodes (12): escapeHtml(), getLanguageGroup(), handleCustomDictInject(), handleKwsToggle(), logEvent(), populateVoicesDropdown(), renderCasesTable(), renderEvalReport() (+4 more)

### Community 17 - "Neural Speech Denoisers & Enhancer"
Cohesion: 0.08
Nodes (19): AudioConfig, DictationConfig, EngineConfig, IntentConfig, KWSConfig, TTSConfig, BypassEnhancer, windowsSpeechEnhancer (+11 more)

### Community 18 - "TTS Model Selection & Warmup"
Cohesion: 0.16
Nodes (10): detectKokoroVersion(), FindInstalledKokoroModels(), findKokoroModelDir(), isRegularFile(), isValidKokoroDir(), kokoroLang(), kokoroLexicons(), mapVoiceToSID() (+2 more)

### Community 20 - "Sherpa-ONNX Runner Integration"
Cohesion: 0.23
Nodes (8): SherpaRunner, StartWarmOffline(), findSherpaBin(), InstalledStreamingModels(), NewSherpaRunner(), resolveModelDir(), findTransducerModel(), TestWarmASRStreaming()

### Community 21 - "Acoustic Perturbations & Corpus Generator"
Cohesion: 0.13
Nodes (23): BaseUtterance, MatrixVariation, NewHostNativeSource(), TestHostNativeCapture(), CalculateRMS(), TestCalculateRMS(), AddNoiseAtSNR(), ApplyReverb() (+15 more)

### Community 22 - "Audio Visualizer & PCM Player"
Cohesion: 0.11
Nodes (11): main(), pct(), printLatencyLine(), printReportSummary(), run(), truncateString(), Cross-Cutting Concerns, Module Hotspots (+3 more)

### Community 23 - "ONNX Metadata Patching Utilities"
Cohesion: 0.33
Nodes (3): inspect_metadata(), main(), patch_metadata()

### Community 24 - "TTSManager"
Cohesion: 0.21
Nodes (3): VoiceProfile, appendJSONLine(), TTSManager

### Community 25 - "EncodeWAV"
Cohesion: 0.40
Nodes (6): main(), Float32ToBytesPCM(), Float32ToInt16(), TestWAVEncodeDecode(), EncodeWAV(), GenerateChime()

### Community 26 - "WebSocket Client Connection Lifecycle"
Cohesion: 0.39
Nodes (3): newWSClient(), wsClient, wsFrame

### Community 29 - "Evaluation Manifest Tier 1"
Cohesion: 0.50
Nodes (3): items, tier, version

### Community 30 - "Evaluation Manifest Tier 2"
Cohesion: 0.50
Nodes (3): items, tier, version

### Community 31 - "Q: Are the 16 inferred relationships involving setupTestServer() correct?"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: Are the 16 inferred relationships involving setupTestServer() correct?, Source Nodes

### Community 32 - "Q: What connects voxlab, Ref Architectural Ideas, and Semantic Intent Router?"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: What connects voxlab, Ref Architectural Ideas, and Semantic Intent Router?, Source Nodes

### Community 33 - "Q: Why does Server connect HTTP Web Server & Networking to 14 other subsystems?"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: Why does Server connect HTTP Web Server & Networking to 14 other subsystems?, Source Nodes

### Community 36 - "Q: Why does SherpaRunner connect Sherpa-ONNX Runner Integration to 8 subsystems?"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: Why does SherpaRunner connect Sherpa-ONNX Runner Integration to 8 subsystems?, Source Nodes

### Community 37 - "Q: Should Speech Denoiser & Audio Generation be split?"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: Should Speech Denoiser & Audio Generation be split?, Source Nodes

### Community 43 - "Q: Should Benchmark Evaluation & Metrics be split?"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: Should Benchmark Evaluation & Metrics be split?, Source Nodes

### Community 44 - "Q: Why does SpeechEngine connect Simulator Engine & Test Generation to 8 subsystems?"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: Why does SpeechEngine connect Simulator Engine & Test Generation to 8 subsystems?, Source Nodes

## Knowledge Gaps
- **24 isolated node(s):** `voxlab`, `Host Voice Daemon Reference`, `Web Audio Canvas Oscilloscope & Spectrum`, `Piper TTS Pipeline`, `PipeWire Audio Server Subsystem` (+19 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 124 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **11 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Server` connect `HTTP Web Server & Networking` to `testing.T`, `Evaluation Runner & Manifest`, `Audio Capture Sources & Devices`, `Simulator Engine & Test Generation`, `DSP Audio Preprocessor Controls`, `Command Catalog & Slot Extraction`, `Voice Interaction State Machine`, `Neural Speech Denoisers & Enhancer`, `.processIncomingAudio`, `TTSManager`, `WebSocket Client Connection Lifecycle`?**
  _High betweenness centrality (0.135) - this node is a cross-community bridge._
- **Are the 16 inferred relationships involving `setupTestServer()` (e.g. with `TestCancelledRunLeavesNoReportAndAllowsANewRun()` and `TestDictationIsIsolatedFromOtherActions()`) actually correct?**
  _`setupTestServer()` has 16 INFERRED edges - model-reasoned connections that need verification._
- **What connects `voxlab`, `Host Voice Daemon Reference`, `Web Audio Canvas Oscilloscope & Spectrum` to the rest of the system?**
  _24 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.06877271456527817 - nodes in this community are weakly interconnected._
- **Why does `SherpaRunner` connect `Sherpa-ONNX Runner Integration` to `TTSResult`, `Audio Capture Sources & Devices`, `Simulator Engine & Test Generation`, `Persistent Warm TTS Synthesis`, `Warm ASR Daemon Management`, `ASR Audio Streaming Pipeline`, `Neural Speech Denoisers & Enhancer`, `TTS Model Selection & Warmup`?**
  _High betweenness centrality (0.054) - this node is a cross-community bridge._
- **Should `Evaluation Runner & Manifest` be split into smaller, more focused modules?**
  _Cohesion score 0.09966777408637874 - nodes in this community are weakly interconnected._
- **Should `Embedded Voice Architecture Reference` be split into smaller, more focused modules?**
  _Cohesion score 0.06507936507936508 - nodes in this community are weakly interconnected._