# Graph Report - voxlab  (2026-11-16)

## Corpus Check
- 91 files · ~112,250 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 2 file(s) not represented in the graph (top: (none) 1, .css 1)

## Summary
- 885 nodes · 2690 edges · 49 communities (31 shown, 18 thin omitted)
- Extraction: 80% EXTRACTED · 20% INFERRED · 0% AMBIGUOUS · INFERRED: 526 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `442ef87b`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- server.go
- Evaluation Runner & Manifest
- sherpa_runner.go
- Embedded Voice Architecture Reference
- Web Client Voice Controller
- VoxLab Architecture & System Specifications
- Audio Capture Sources & Devices
- TTSResult
- Persistent Warm TTS Synthesis
- .routes
- DSP Audio Preprocessor Controls
- testing.T
- SherpaRunner
- Server
- dictation.go
- Voice Interaction State Machine
- bench-ui.js
- windowsSpeechEnhancer
- SimulatorEngine
- .processIncomingAudio
- SpeechEngine
- Acoustic Perturbations & Corpus Generator
- sync.Mutex
- fakeEngine
- TTSManager
- DecodeWAV
- WebSocket Client Connection Lifecycle
- SpeechEnhancer
- AudioCapture
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
- patch_kokoro_fp16_metadata.py
- review-261115.md

## God Nodes (most connected - your core abstractions)
1. `Server` - 80 edges
2. `SherpaRunner` - 39 edges
3. `VoiceClient` - 35 edges
4. `setupTestServer()` - 31 edges
5. `NewServer()` - 29 edges
6. `SpeechEngine` - 25 edges
7. `StateMachine` - 21 edges
8. `DSPProcessor` - 20 edges
9. `NewSherpaRunner()` - 18 edges
10. `windowsSpeechEnhancer` - 17 edges

## Surprising Connections (you probably didn't know these)
- `Quick Wins` --references--> `escapeHtml()`  [INFERRED]
  review-261115.md → web/js/bench-ui.js
- `Top Issues` --references--> `LoadConfig()`  [INFERRED]
  review-261115.md → pkg/config/config.go
- `Top Issues` --references--> `getFreeTCPPort()`  [INFERRED]
  review-261115.md → pkg/engine/warm_asr.go
- `Cross-Cutting Concerns` --references--> `EncodeWAV()`  [INFERRED]
  review-261115.md → pkg/audio/wav.go
- `Cross-Cutting Concerns` --references--> `truncateString()`  [INFERRED]
  review-261115.md → cmd/eval_bench/main.go

## Import Cycles
- 2-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/server.go -> pkg/evaluation/smoke_test.go`
- 2-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/smoke_test.go`
- 2-file cycle: `pkg/evaluation/smoke_test.go -> pkg/evaluation/tier2_test.go -> pkg/evaluation/smoke_test.go`
- 2-file cycle: `pkg/evaluation/tier2_test.go -> pkg/server/server.go -> pkg/evaluation/tier2_test.go`
- 2-file cycle: `pkg/evaluation/tier2_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/tier2_test.go`
- 3-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/server.go -> pkg/evaluation/tier2_test.go -> pkg/evaluation/smoke_test.go`
- 3-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/tier2_test.go -> pkg/evaluation/smoke_test.go`
- 3-file cycle: `pkg/evaluation/smoke_test.go -> pkg/evaluation/tier2_test.go -> pkg/server/server.go -> pkg/evaluation/smoke_test.go`
- 3-file cycle: `pkg/evaluation/smoke_test.go -> pkg/evaluation/tier2_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/smoke_test.go`
- 4-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/server.go -> pkg/evaluation/tier2_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/smoke_test.go`
- 4-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/tier2_test.go -> pkg/server/server.go -> pkg/evaluation/smoke_test.go`

## Hyperedges (group relationships)
- **Voice Command Processing Architecture** — readme_audio_input_layer, readme_wake_word_detector, readme_voice_commands_pipeline, readme_semantic_intent_router, web_index_commands_tab [EXTRACTED 1.00]
- **Acoustic Benchmark & Evaluation Framework** — readme_acoustic_eval_bench, todo_acoustic_matrix_findings, todo_audio_resampler, web_index_acoustic_eval_tab [INFERRED 0.95]
- **Two-Pass Speech Recognition Pipeline** — todo_nemotron_35_upgrade, todo_parakeet_ultra_upgrade, readme_voice_dictations_pipeline, web_index_dictation_tab [INFERRED 0.95]
- **** — ref_synthetic_evaluation_bench_synthetic_benchmark_harness, ref_synthetic_evaluation_bench_two_pass_asr, ref_synthetic_evaluation_bench_two_tier_test_suite, ref_synthetic_evaluation_bench_tier1_golden_cases, ref_synthetic_evaluation_bench_quantitative_metrics [INFERRED]
- **** — ref_linux_wpe_voice_recipe_interaction_state_machine, ref_linux_wpe_voice_recipe_command_catalog_slots, ref_linux_wpe_voice_recipe_annotation_dictation_mode, ref_linux_wpe_voice_recipe_echo_control_policy [INFERRED]
- **** — ref_embedded_voice_architecture_recipe_silero_vad, ref_embedded_voice_architecture_recipe_wake_word_detector, ref_embedded_voice_architecture_recipe_streaming_asr_zipformer, ref_embedded_voice_architecture_recipe_semantic_intent_router, ref_embedded_voice_architecture_recipe_piper_tts [INFERRED]

## Communities (49 total, 18 thin omitted)

### Community 0 - "server.go"
Cohesion: 0.14
Nodes (16): cDenoisedAudio, cDenoiserModelConfig, cDpdfNetModelConfig, cGtcrnModelConfig, cOnlineSpeechDenoiserConfig, cGeneratedAudio, cKittenModelConfig, cKokoroModelConfig (+8 more)

### Community 1 - "Evaluation Runner & Manifest"
Cohesion: 0.06
Nodes (50): BaseUtterance, EvalItem, MatrixVariation, DimensionGroup, ItemResult, LatencyPercentiles, LatencyStatistics, RunConfig (+42 more)

### Community 2 - "sherpa_runner.go"
Cohesion: 0.14
Nodes (15): TestKokoroLang(), detectKokoroVersion(), FindInstalledKokoroModels(), findKokoroModelDir(), isRegularFile(), isValidKokoroDir(), kokoroLang(), mapVoiceToSID() (+7 more)

### Community 3 - "Embedded Voice Architecture Reference"
Cohesion: 0.07
Nodes (31): Piper TTS Pipeline, PipeWire Audio Server Subsystem, Semantic Intent Router & Distractor Suppression, Silero VAD & Energy Gating, Streaming Zipformer ASR, Host Voice Daemon Reference, Wake-Word Detector, Annotation Dictation Mode (+23 more)

### Community 5 - "VoxLab Architecture & System Specifications"
Cohesion: 0.07
Nodes (25): Ref Architectural Ideas, VoxLab App Specification, Dynamic Model Switching Architecture, Sherpa-ONNX Metadata Requirements, Acoustic Benchmark System, Dual Audio Input Layer, Kokoro TTS Engine, ONNX Metadata Helper Script (+17 more)

### Community 6 - "Audio Capture Sources & Devices"
Cohesion: 0.17
Nodes (5): AudioSource, detectWindowsDirectShowMic(), HostNativeSource, WavFileSource, NewWavFileSource()

### Community 7 - "TTSResult"
Cohesion: 0.31
Nodes (5): TTSChunk, TTSResult, TTSRequest, clampSpeed(), float32SliceToTTSResult()

### Community 8 - "Persistent Warm TTS Synthesis"
Cohesion: 0.13
Nodes (7): stubWarmTTS, WarmTTS, windowsWarmTTS, kokoroLexicons(), warmTTSKey(), newPlatformWarmTTS(), newPlatformWarmTTS()

### Community 11 - "testing.T"
Cohesion: 0.05
Nodes (70): main(), pct(), printLatencyLine(), printReportSummary(), run(), main(), NewHostNativeSource(), TestHostNativeCapture() (+62 more)

### Community 12 - "SherpaRunner"
Cohesion: 0.06
Nodes (33): DictationConfig, EngineConfig, IntentConfig, KWSConfig, TTSConfig, ASRResult, ASRStream, daemon (+25 more)

### Community 14 - "dictation.go"
Cohesion: 0.09
Nodes (30): CommandCatalog, CommandEntry, IntentDecision, MatchCandidate, DefaultCatalog(), LoadCatalog(), Tokenize(), calculateSimilarity() (+22 more)

### Community 15 - "Voice Interaction State Machine"
Cohesion: 0.20
Nodes (4): State, StateMachine, asrWorkItem, StateEvent

### Community 16 - "bench-ui.js"
Cohesion: 0.13
Nodes (18): base64ToArrayBuffer(), escapeHtml(), getLanguageGroup(), getStreamAudioCtx(), handleCustomDictInject(), handleKwsToggle(), handleStreamChunk(), logEvent() (+10 more)

### Community 20 - "SpeechEngine"
Cohesion: 0.23
Nodes (14): main(), main(), EvalManifest, TestWAVEncodeDecode(), EncodeWAV(), GenerateChime(), SpeechEngine, LoadManifest() (+6 more)

### Community 21 - "Acoustic Perturbations & Corpus Generator"
Cohesion: 0.19
Nodes (15): CalculateRMS(), TestCalculateRMS(), AddNoiseAtSNR(), ApplyReverb(), GenerateBabbleNoise(), GeneratePinkNoise(), GenerateTypingNoise(), GenerateWhiteNoise() (+7 more)

### Community 22 - "sync.Mutex"
Cohesion: 0.22
Nodes (4): tailWriter, WebSocketSource, NewWebSocketSource(), TestTailWriterKeepsEnd()

### Community 24 - "TTSManager"
Cohesion: 0.19
Nodes (3): VoiceProfile, appendJSONLine(), TTSManager

### Community 25 - "DecodeWAV"
Cohesion: 0.47
Nodes (5): BytesToFloat32PCM(), BytesToInt16PCM(), Int16ToFloat32(), TestDecodeWAVEdgeCases(), DecodeWAV()

### Community 26 - "WebSocket Client Connection Lifecycle"
Cohesion: 0.29
Nodes (3): newWSClient(), wsClient, wsFrame

### Community 27 - "SpeechEnhancer"
Cohesion: 0.11
Nodes (10): AudioConfig, BypassEnhancer, SpeechEnhancer, NewBypassEnhancer(), NewSpeechEnhancer(), newPlatformSpeechEnhancer(), TestBypassEnhancer(), TestSpeechEnhancerFactoryAndProcessing() (+2 more)

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

### Community 47 - "patch_kokoro_fp16_metadata.py"
Cohesion: 0.36
Nodes (4): inspect_metadata(), main(), missing_keys(), patch_metadata()

### Community 50 - "review-261115.md"
Cohesion: 0.18
Nodes (10): truncateString(), optimizeStreamingClauses(), TestOptimizeStreamingClauses(), TestWarmTTSSentenceSplitting(), toCString(), Cross-Cutting Concerns, Module Hotspots, Project-Level Review Summary (+2 more)

## Knowledge Gaps
- **24 isolated node(s):** `voxlab`, `Host Voice Daemon Reference`, `Web Audio Canvas Oscilloscope & Spectrum`, `Piper TTS Pipeline`, `PipeWire Audio Server Subsystem` (+19 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 116 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **18 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Server` connect `Server` to `server.go`, `Evaluation Runner & Manifest`, `Audio Capture Sources & Devices`, `.routes`, `DSP Audio Preprocessor Controls`, `testing.T`, `SherpaRunner`, `dictation.go`, `Voice Interaction State Machine`, `.ModelType`, `.processIncomingAudio`, `SpeechEngine`, `sync.Mutex`, `TTSManager`, `WebSocket Client Connection Lifecycle`, `SpeechEnhancer`?**
  _High betweenness centrality (0.130) - this node is a cross-community bridge._
- **Are the 16 inferred relationships involving `setupTestServer()` (e.g. with `TestCancelledRunLeavesNoReportAndAllowsANewRun()` and `TestDictationIsIsolatedFromOtherActions()`) actually correct?**
  _`setupTestServer()` has 16 INFERRED edges - model-reasoned connections that need verification._
- **What connects `voxlab`, `Host Voice Daemon Reference`, `Web Audio Canvas Oscilloscope & Spectrum` to the rest of the system?**
  _24 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `server.go` be split into smaller, more focused modules?**
  _Cohesion score 0.13791208791208792 - nodes in this community are weakly interconnected._
- **Why does `VoiceClient` connect `Web Client Voice Controller` to `review-261115.md`?**
  _High betweenness centrality (0.058) - this node is a cross-community bridge._
- **Should `Evaluation Runner & Manifest` be split into smaller, more focused modules?**
  _Cohesion score 0.06352087114337568 - nodes in this community are weakly interconnected._
- **Why does `SherpaRunner` connect `SherpaRunner` to `server.go`, `sherpa_runner.go`, `TTSResult`, `Persistent Warm TTS Synthesis`, `testing.T`, `review-261115.md`, `SimulatorEngine`, `sync.Mutex`?**
  _High betweenness centrality (0.054) - this node is a cross-community bridge._