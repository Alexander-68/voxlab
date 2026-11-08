# Graph Report - voxlab  (2026-11-08)

## Corpus Check
- 190 files · ~88,491 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 2 file(s) not represented in the graph (top: (none) 1, .css 1)

## Summary
- 794 nodes · 2427 edges · 43 communities (30 shown, 13 thin omitted)
- Extraction: 82% EXTRACTED · 18% INFERRED · 0% AMBIGUOUS · INFERRED: 442 edges (avg confidence: 0.84)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Speech Denoiser & Audio Generation
- Benchmark Evaluation & Metrics
- Speech Enhancer DSP Pipeline
- Embedded Voice Architecture Reference
- Web Client Voice Controller
- VoxLab Architecture & System Specifications
- Audio Capture Sources & Devices
- Simulator Engine & Test Generation
- Persistent Warm TTS Synthesis
- HTTP Web Server & Networking
- DSP Audio Preprocessor Controls
- Core Test Suite & Security Verification
- Warm ASR Daemon Management
- ASR Audio Streaming Pipeline
- Command Catalog & Slot Extraction
- Voice Interaction State Machine
- Benchmark UI & Audio Helpers
- Configuration & Application Entrypoint
- TTS Model Selection & Warmup
- TTS Manager & Dictation Streaming
- Sherpa-ONNX Runner Integration
- Acoustic Perturbations & RMS Calculations
- Audio Visualizer & PCM Player
- ONNX Metadata Patching Utilities
- DSP & TTS Integration Tests
- Thread-Safe Ring Buffer
- WebSocket Client Connection Lifecycle
- Dictation Phrase Formatting & Normalization
- Integration End-to-End Tests
- Evaluation Manifest Tier 1
- Evaluation Manifest Tier 2
- Kokoro Installation & Model Fallback
- Streaming Clause Optimization & Splitting
- Tail Log Buffer Utilities
- Kokoro FP16 Model Profiles
- Kokoro INT8 Model Profiles
- Kokoro Language Mapping & Tests
- TTS Text Normalization & Pauses
- WPE WebKit IPC Integration
- Web Test Bench Architecture
- Bash Model Download Script
- VoxLab Root Module

## God Nodes (most connected - your core abstractions)
1. `Server` - 80 edges
2. `SherpaRunner` - 37 edges
3. `VoiceClient` - 35 edges
4. `setupTestServer()` - 31 edges
5. `NewServer()` - 29 edges
6. `SpeechEngine` - 23 edges
7. `StateMachine` - 21 edges
8. `DSPProcessor` - 20 edges
9. `windowsSpeechEnhancer` - 17 edges
10. `NewSherpaRunner()` - 17 edges

## Surprising Connections (you probably didn't know these)
- `Audio In & DSP Panel` --conceptually_related_to--> `DSP Preprocessing Pipeline`  [INFERRED]
  web/index.html → README.md
- `Voice Commands Tab` --conceptually_related_to--> `Voice Commands Pipeline`  [INFERRED]
  web/index.html → README.md
- `Voice Dictations Tab` --conceptually_related_to--> `Voice Dictations Pipeline`  [INFERRED]
  web/index.html → README.md
- `run()` --calls--> `NewServer()`  [EXTRACTED]
  cmd/eval_bench/main.go → pkg/server/server.go
- `main()` --calls--> `DefaultConfig()`  [EXTRACTED]
  cmd/gen_eval_samples/main.go → pkg/config/config.go

## Import Cycles
- 2-file cycle: `pkg/evaluation/smoke_test.go -> pkg/evaluation/tier2_test.go -> pkg/evaluation/smoke_test.go`
- 2-file cycle: `pkg/evaluation/tier2_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/tier2_test.go`
- 2-file cycle: `pkg/evaluation/tier2_test.go -> pkg/server/server.go -> pkg/evaluation/tier2_test.go`
- 2-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/smoke_test.go`
- 2-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/server.go -> pkg/evaluation/smoke_test.go`
- 3-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/tier2_test.go -> pkg/evaluation/smoke_test.go`
- 3-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/server.go -> pkg/evaluation/tier2_test.go -> pkg/evaluation/smoke_test.go`
- 3-file cycle: `pkg/evaluation/smoke_test.go -> pkg/evaluation/tier2_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/smoke_test.go`
- 3-file cycle: `pkg/evaluation/smoke_test.go -> pkg/evaluation/tier2_test.go -> pkg/server/server.go -> pkg/evaluation/smoke_test.go`
- 4-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/server.go -> pkg/evaluation/tier2_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/smoke_test.go`
- 4-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/tier2_test.go -> pkg/server/server.go -> pkg/evaluation/smoke_test.go`

## Hyperedges (group relationships)
- **Voice Command Processing Architecture** — readme_audio_input_layer, readme_wake_word_detector, readme_voice_commands_pipeline, readme_semantic_intent_router, web_index_commands_tab [EXTRACTED 1.00]
- **Two-Pass Speech Recognition Pipeline** — todo_nemotron_35_upgrade, todo_parakeet_ultra_upgrade, readme_voice_dictations_pipeline, web_index_dictation_tab [INFERRED 0.95]
- **Acoustic Benchmark & Evaluation Framework** — readme_acoustic_eval_bench, todo_acoustic_matrix_findings, todo_audio_resampler, web_index_acoustic_eval_tab [INFERRED 0.95]
- **** — ref_embedded_voice_architecture_recipe_silero_vad, ref_embedded_voice_architecture_recipe_wake_word_detector, ref_embedded_voice_architecture_recipe_streaming_asr_zipformer, ref_embedded_voice_architecture_recipe_semantic_intent_router, ref_embedded_voice_architecture_recipe_piper_tts [INFERRED]
- **** — ref_linux_wpe_voice_recipe_interaction_state_machine, ref_linux_wpe_voice_recipe_command_catalog_slots, ref_linux_wpe_voice_recipe_annotation_dictation_mode, ref_linux_wpe_voice_recipe_echo_control_policy [INFERRED]
- **** — ref_synthetic_evaluation_bench_synthetic_benchmark_harness, ref_synthetic_evaluation_bench_two_pass_asr, ref_synthetic_evaluation_bench_two_tier_test_suite, ref_synthetic_evaluation_bench_tier1_golden_cases, ref_synthetic_evaluation_bench_quantitative_metrics [INFERRED]

## Communities (43 total, 13 thin omitted)

### Community 0 - "Speech Denoiser & Audio Generation"
Cohesion: 0.13
Nodes (16): cDenoisedAudio, cDenoiserModelConfig, cDpdfNetModelConfig, cGtcrnModelConfig, cOnlineSpeechDenoiserConfig, cGeneratedAudio, cKittenModelConfig, cKokoroModelConfig (+8 more)

### Community 1 - "Benchmark Evaluation & Metrics"
Cohesion: 0.07
Nodes (45): pct(), printLatencyLine(), printReportSummary(), BaseUtterance, DimensionGroup, EvalItem, EvalManifest, LatencyStatistics (+37 more)

### Community 2 - "Speech Enhancer DSP Pipeline"
Cohesion: 0.09
Nodes (9): BypassEnhancer, windowsSpeechEnhancer, SpeechEnhancer, NewBypassEnhancer(), NewSpeechEnhancer(), newPlatformSpeechEnhancer(), TestBypassEnhancer(), newPlatformSpeechEnhancer() (+1 more)

### Community 3 - "Embedded Voice Architecture Reference"
Cohesion: 0.07
Nodes (31): Piper TTS Pipeline, PipeWire Audio Server Subsystem, Semantic Intent Router & Distractor Suppression, Silero VAD & Energy Gating, Streaming Zipformer ASR, Host Voice Daemon Reference, Wake-Word Detector, Annotation Dictation Mode (+23 more)

### Community 5 - "VoxLab Architecture & System Specifications"
Cohesion: 0.07
Nodes (25): Ref Architectural Ideas, VoxLab App Specification, Dynamic Model Switching Architecture, Sherpa-ONNX Metadata Requirements, Acoustic Benchmark System, Dual Audio Input Layer, Kokoro TTS Engine, ONNX Metadata Helper Script (+17 more)

### Community 6 - "Audio Capture Sources & Devices"
Cohesion: 0.11
Nodes (8): AudioSource, tailWriter, detectWindowsDirectShowMic(), HostNativeSource, WavFileSource, WebSocketSource, NewWavFileSource(), NewWebSocketSource()

### Community 7 - "Simulator Engine & Test Generation"
Cohesion: 0.11
Nodes (10): generateTier1(), main(), main(), SimulatorEngine, TestWAVEncodeDecode(), EncodeWAV(), GenerateChime(), SpeechEngine (+2 more)

### Community 8 - "Persistent Warm TTS Synthesis"
Cohesion: 0.12
Nodes (11): stubWarmTTS, WarmTTS, windowsWarmTTS, TTSChunk, TTSResult, kokoroLexicons(), warmTTSKey(), float32SliceToTTSResult() (+3 more)

### Community 9 - "HTTP Web Server & Networking"
Cohesion: 0.16
Nodes (3): isLoopbackName(), NewServer(), Server

### Community 11 - "Core Test Suite & Security Verification"
Cohesion: 0.14
Nodes (27): TestSpeechEnhancerFactoryAndProcessing(), TestAppendDictationPhrase(), TestEvalReportRejectsUnknownTiers(), TestFailedSaveKeepsDraft(), TestSaveDraftStoresTextAndClearsDraft(), TestDefaultBindIsLoopback(), TestListenReportsATakenPort(), TestOversizedRequestsAreRefused() (+19 more)

### Community 12 - "Warm ASR Daemon Management"
Cohesion: 0.10
Nodes (20): daemon, WarmASR, WarmOffline, BytesToFloat32PCM(), BytesToInt16PCM(), Int16ToFloat32(), TestDecodeWAVEdgeCases(), DecodeWAV() (+12 more)

### Community 13 - "ASR Audio Streaming Pipeline"
Cohesion: 0.13
Nodes (4): ASRStream, ASRResult, containsCJK(), fakeEngine

### Community 14 - "Command Catalog & Slot Extraction"
Cohesion: 0.18
Nodes (18): CommandCatalog, CommandEntry, IntentDecision, MatchCandidate, DefaultCatalog(), LoadCatalog(), Tokenize(), calculateSimilarity() (+10 more)

### Community 15 - "Voice Interaction State Machine"
Cohesion: 0.18
Nodes (6): State, StateMachine, NewStateMachine(), TestStateMachine(), asrWorkItem, StateEvent

### Community 16 - "Benchmark UI & Audio Helpers"
Cohesion: 0.13
Nodes (11): escapeHtml(), getLanguageGroup(), handleCustomDictInject(), handleKwsToggle(), logEvent(), populateVoicesDropdown(), renderCasesTable(), renderEvalReport() (+3 more)

### Community 17 - "Configuration & Application Entrypoint"
Cohesion: 0.17
Nodes (20): main(), run(), truncateString(), main(), AudioConfig, DictationConfig, EngineConfig, IntentConfig (+12 more)

### Community 18 - "TTS Model Selection & Warmup"
Cohesion: 0.17
Nodes (11): TestMissingModelIsNotReportedAsNeural(), detectKokoroVersion(), FindInstalledKokoroModels(), findKokoroModelDir(), isRegularFile(), isValidKokoroDir(), mapVoiceToSID(), TestFindKokoroModelDir() (+3 more)

### Community 19 - "TTS Manager & Dictation Streaming"
Cohesion: 0.17
Nodes (4): VoiceProfile, AppendDictationPhrase(), appendJSONLine(), TTSManager

### Community 20 - "Sherpa-ONNX Runner Integration"
Cohesion: 0.19
Nodes (8): SherpaRunner, InstalledStreamingModels(), NewSherpaRunner(), resolveModelDir(), TestSecondPassPerMode(), TestSynthesizeChinese(), TestSynthesizeKokoroV10(), findTransducerModel()

### Community 21 - "Acoustic Perturbations & RMS Calculations"
Cohesion: 0.18
Nodes (17): NewHostNativeSource(), TestHostNativeCapture(), CalculateRMS(), TestCalculateRMS(), AddNoiseAtSNR(), ApplyReverb(), GenerateBabbleNoise(), GeneratePinkNoise() (+9 more)

### Community 23 - "ONNX Metadata Patching Utilities"
Cohesion: 0.33
Nodes (3): inspect_metadata(), main(), patch_metadata()

### Community 24 - "DSP & TTS Integration Tests"
Cohesion: 0.33
Nodes (9): NewDSPProcessor(), TestDSPHighPassAndAGC(), TestDSPNoiseGateAndEcho(), NewSimulatorEngine(), TestSimulatorEngine(), NewTTSManager(), TestTTSManagerEchoSuppression(), TestTTSManagerModelAndVoices() (+1 more)

### Community 25 - "Thread-Safe Ring Buffer"
Cohesion: 0.25
Nodes (3): TestRingBuffer(), RingBuffer, NewRingBuffer()

### Community 26 - "WebSocket Client Connection Lifecycle"
Cohesion: 0.39
Nodes (3): newWSClient(), wsClient, wsFrame

### Community 27 - "Dictation Phrase Formatting & Normalization"
Cohesion: 0.40
Nodes (4): FormatDictationPhrase(), formatDictationPhraseInternal(), replaceSpokenWord(), TestFormatDictationPhrase()

### Community 28 - "Integration End-to-End Tests"
Cohesion: 0.50
Nodes (5): dialTestClient(), TestCancelledRunLeavesNoReportAndAllowsANewRun(), TestDictationIsIsolatedFromOtherActions(), TestDisconnectDuringPlaybackClearsEchoMute(), waitUntil()

### Community 29 - "Evaluation Manifest Tier 1"
Cohesion: 0.50
Nodes (3): items, tier, version

### Community 30 - "Evaluation Manifest Tier 2"
Cohesion: 0.50
Nodes (3): items, tier, version

### Community 31 - "Kokoro Installation & Model Fallback"
Cohesion: 0.67
Nodes (3): installKokoro(), TestSetTTSModelOnlyAcceptsInstalledModels(), TestUnavailableConfiguredModelFallsBackToDiscovery()

### Community 32 - "Streaming Clause Optimization & Splitting"
Cohesion: 0.67
Nodes (3): optimizeStreamingClauses(), TestOptimizeStreamingClauses(), TestWarmTTSSentenceSplitting()

## Knowledge Gaps
- **24 isolated node(s):** `voxlab`, `Ref Architectural Ideas`, `Semantic Intent Router`, `WebSocket IPC Protocol`, `Kokoro v1.1 Multi-Language` (+19 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 96 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **13 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Server` connect `HTTP Web Server & Networking` to `Speech Denoiser & Audio Generation`, `Benchmark Evaluation & Metrics`, `Speech Enhancer DSP Pipeline`, `Audio Capture Sources & Devices`, `Simulator Engine & Test Generation`, `DSP Audio Preprocessor Controls`, `Core Test Suite & Security Verification`, `Command Catalog & Slot Extraction`, `Voice Interaction State Machine`, `Configuration & Application Entrypoint`, `TTS Manager & Dictation Streaming`, `Thread-Safe Ring Buffer`, `WebSocket Client Connection Lifecycle`, `Integration End-to-End Tests`?**
  _High betweenness centrality (0.144) - this node is a cross-community bridge._
- **Are the 16 inferred relationships involving `setupTestServer()` (e.g. with `TestCancelledRunLeavesNoReportAndAllowsANewRun()` and `TestDictationIsIsolatedFromOtherActions()`) actually correct?**
  _`setupTestServer()` has 16 INFERRED edges - model-reasoned connections that need verification._
- **What connects `voxlab`, `Ref Architectural Ideas`, `Semantic Intent Router` to the rest of the system?**
  _24 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Speech Denoiser & Audio Generation` be split into smaller, more focused modules?**
  _Cohesion score 0.12692307692307692 - nodes in this community are weakly interconnected._
- **Why does `SherpaRunner` connect `Sherpa-ONNX Runner Integration` to `Speech Denoiser & Audio Generation`, `Audio Capture Sources & Devices`, `Simulator Engine & Test Generation`, `Persistent Warm TTS Synthesis`, `Warm ASR Daemon Management`, `ASR Audio Streaming Pipeline`, `Configuration & Application Entrypoint`, `TTS Model Selection & Warmup`?**
  _High betweenness centrality (0.051) - this node is a cross-community bridge._
- **Should `Benchmark Evaluation & Metrics` be split into smaller, more focused modules?**
  _Cohesion score 0.06894049346879536 - nodes in this community are weakly interconnected._
- **Why does `SpeechEngine` connect `Simulator Engine & Test Generation` to `Speech Denoiser & Audio Generation`, `HTTP Web Server & Networking`, `Core Test Suite & Security Verification`, `ASR Audio Streaming Pipeline`, `TTS Model Selection & Warmup`, `TTS Manager & Dictation Streaming`, `Acoustic Perturbations & RMS Calculations`, `DSP & TTS Integration Tests`?**
  _High betweenness centrality (0.026) - this node is a cross-community bridge._