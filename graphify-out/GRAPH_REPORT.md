# Graph Report - voxlab  (2026-11-08)

## Corpus Check
- 90 files · ~90,504 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 2 file(s) not represented in the graph (top: (none) 1, .css 1)

## Summary
- 844 nodes · 2426 edges · 46 communities (34 shown, 12 thin omitted)
- Extraction: 82% EXTRACTED · 18% INFERRED · 0% AMBIGUOUS · INFERRED: 447 edges (avg confidence: 0.84)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `53b6862a`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Speech Denoiser & Audio Generation
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
- testing.T
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
- Thread-Safe Ring Buffer
- WebSocket Client Connection Lifecycle
- Dictation Phrase Formatting & Normalization
- Benchmark Metrics Reporting
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
- `NewServer()` --calls--> `run()`  [EXTRACTED]
  pkg/server/server.go → cmd/eval_bench/main.go
- `NewServer()` --calls--> `main()`  [EXTRACTED]
  pkg/server/server.go → cmd/voxlab/main.go

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

## Communities (46 total, 12 thin omitted)

### Community 0 - "Speech Denoiser & Audio Generation"
Cohesion: 0.14
Nodes (11): cGeneratedAudio, cKittenModelConfig, cKokoroModelConfig, cMatchaModelConfig, cOfflineTtsConfig, cOfflineTtsModelConfig, cPocketModelConfig, cSupertonicModelConfig (+3 more)

### Community 1 - "Evaluation Runner & Manifest"
Cohesion: 0.10
Nodes (36): EvalItem, EvalManifest, DimensionGroup, ItemResult, LatencyPercentiles, LatencyStatistics, RunConfig, SNRCurvePoint (+28 more)

### Community 2 - "TTSResult"
Cohesion: 0.15
Nodes (13): main(), stubWarmTTS, Float32ToBytesPCM(), Float32ToInt16(), TestWAVEncodeDecode(), EncodeWAV(), GenerateChime(), TTSChunk (+5 more)

### Community 3 - "Embedded Voice Architecture Reference"
Cohesion: 0.07
Nodes (31): Piper TTS Pipeline, PipeWire Audio Server Subsystem, Semantic Intent Router & Distractor Suppression, Silero VAD & Energy Gating, Streaming Zipformer ASR, Host Voice Daemon Reference, Wake-Word Detector, Annotation Dictation Mode (+23 more)

### Community 5 - "VoxLab Architecture & System Specifications"
Cohesion: 0.07
Nodes (25): Ref Architectural Ideas, VoxLab App Specification, Dynamic Model Switching Architecture, Sherpa-ONNX Metadata Requirements, Acoustic Benchmark System, Dual Audio Input Layer, Kokoro TTS Engine, ONNX Metadata Helper Script (+17 more)

### Community 6 - "Audio Capture Sources & Devices"
Cohesion: 0.08
Nodes (16): AudioSource, tailWriter, detectWindowsDirectShowMic(), HostNativeSource, WavFileSource, WebSocketSource, NewWavFileSource(), NewWebSocketSource() (+8 more)

### Community 8 - "Persistent Warm TTS Synthesis"
Cohesion: 0.17
Nodes (5): WarmTTS, windowsWarmTTS, warmTTSKey(), newPlatformWarmTTS(), newPlatformWarmTTS()

### Community 9 - "HTTP Web Server & Networking"
Cohesion: 0.14
Nodes (3): isLoopbackName(), EvalState, Server

### Community 10 - "DSP Audio Preprocessor Controls"
Cohesion: 0.12
Nodes (4): DSPProcessor, NewServer(), NewStateMachine(), TestStateMachine()

### Community 11 - "testing.T"
Cohesion: 0.05
Nodes (70): main(), main(), NewHostNativeSource(), TestHostNativeCapture(), CalculateRMS(), NewDSPProcessor(), TestCalculateRMS(), TestDSPHighPassAndAGC() (+62 more)

### Community 12 - "Warm ASR Daemon Management"
Cohesion: 0.13
Nodes (15): daemon, WarmASR, WarmOffline, killWithParent(), startDaemon(), killWithParent(), StartWarmOffline(), findSherpaBin() (+7 more)

### Community 13 - "ASR Audio Streaming Pipeline"
Cohesion: 0.18
Nodes (4): ASRStream, ASRResult, containsCJK(), fakeEngine

### Community 14 - "Command Catalog & Slot Extraction"
Cohesion: 0.17
Nodes (18): CommandCatalog, CommandEntry, IntentDecision, MatchCandidate, DefaultCatalog(), LoadCatalog(), Tokenize(), calculateSimilarity() (+10 more)

### Community 15 - "Voice Interaction State Machine"
Cohesion: 0.20
Nodes (4): State, StateMachine, asrWorkItem, StateEvent

### Community 16 - "Benchmark UI & Audio Helpers"
Cohesion: 0.13
Nodes (11): escapeHtml(), getLanguageGroup(), handleCustomDictInject(), handleKwsToggle(), logEvent(), populateVoicesDropdown(), renderCasesTable(), renderEvalReport() (+3 more)

### Community 17 - "Neural Speech Denoisers & Enhancer"
Cohesion: 0.06
Nodes (23): AudioConfig, DictationConfig, EngineConfig, IntentConfig, KWSConfig, TTSConfig, BypassEnhancer, cDenoisedAudio (+15 more)

### Community 18 - "TTS Model Selection & Warmup"
Cohesion: 0.19
Nodes (13): detectKokoroVersion(), FindInstalledKokoroModels(), findKokoroModelDir(), isRegularFile(), isValidKokoroDir(), kokoroLang(), kokoroLexicons(), mapVoiceToSID() (+5 more)

### Community 19 - ".processIncomingAudio"
Cohesion: 0.14
Nodes (3): AppendDictationPhrase(), appendJSONLine(), downsampleWave()

### Community 20 - "Sherpa-ONNX Runner Integration"
Cohesion: 0.21
Nodes (7): SherpaRunner, InstalledStreamingModels(), NewSherpaRunner(), resolveModelDir(), TestKokoroFP16Model(), TestKokoroINT8Model(), findTransducerModel()

### Community 21 - "Acoustic Perturbations & Corpus Generator"
Cohesion: 0.21
Nodes (15): BaseUtterance, MatrixVariation, AddNoiseAtSNR(), ApplyReverb(), GenerateBabbleNoise(), GeneratePinkNoise(), GenerateTypingNoise(), GenerateWhiteNoise() (+7 more)

### Community 23 - "ONNX Metadata Patching Utilities"
Cohesion: 0.33
Nodes (3): inspect_metadata(), main(), patch_metadata()

### Community 25 - "Thread-Safe Ring Buffer"
Cohesion: 0.25
Nodes (3): TestRingBuffer(), RingBuffer, NewRingBuffer()

### Community 26 - "WebSocket Client Connection Lifecycle"
Cohesion: 0.39
Nodes (3): newWSClient(), wsClient, wsFrame

### Community 27 - "Dictation Phrase Formatting & Normalization"
Cohesion: 0.40
Nodes (4): FormatDictationPhrase(), formatDictationPhraseInternal(), replaceSpokenWord(), TestFormatDictationPhrase()

### Community 28 - "Benchmark Metrics Reporting"
Cohesion: 0.29
Nodes (6): main(), pct(), printLatencyLine(), printReportSummary(), run(), truncateString()

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
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 127 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **12 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Server` connect `HTTP Web Server & Networking` to `Speech Denoiser & Audio Generation`, `Evaluation Runner & Manifest`, `Audio Capture Sources & Devices`, `Simulator Engine & Test Generation`, `DSP Audio Preprocessor Controls`, `testing.T`, `Command Catalog & Slot Extraction`, `Voice Interaction State Machine`, `Neural Speech Denoisers & Enhancer`, `.processIncomingAudio`, `TTSManager`, `Thread-Safe Ring Buffer`, `WebSocket Client Connection Lifecycle`?**
  _High betweenness centrality (0.134) - this node is a cross-community bridge._
- **Are the 16 inferred relationships involving `setupTestServer()` (e.g. with `TestCancelledRunLeavesNoReportAndAllowsANewRun()` and `TestDictationIsIsolatedFromOtherActions()`) actually correct?**
  _`setupTestServer()` has 16 INFERRED edges - model-reasoned connections that need verification._
- **What connects `voxlab`, `Host Voice Daemon Reference`, `Web Audio Canvas Oscilloscope & Spectrum` to the rest of the system?**
  _24 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Speech Denoiser & Audio Generation` be split into smaller, more focused modules?**
  _Cohesion score 0.13927376970855232 - nodes in this community are weakly interconnected._
- **Why does `SherpaRunner` connect `Sherpa-ONNX Runner Integration` to `TTSResult`, `Audio Capture Sources & Devices`, `Simulator Engine & Test Generation`, `Persistent Warm TTS Synthesis`, `Warm ASR Daemon Management`, `ASR Audio Streaming Pipeline`, `Neural Speech Denoisers & Enhancer`, `TTS Model Selection & Warmup`?**
  _High betweenness centrality (0.047) - this node is a cross-community bridge._
- **Should `Evaluation Runner & Manifest` be split into smaller, more focused modules?**
  _Cohesion score 0.09619450317124736 - nodes in this community are weakly interconnected._
- **Why does `SpeechEngine` connect `Simulator Engine & Test Generation` to `Speech Denoiser & Audio Generation`, `Evaluation Runner & Manifest`, `TTSResult`, `HTTP Web Server & Networking`, `DSP Audio Preprocessor Controls`, `testing.T`, `ASR Audio Streaming Pipeline`, `Acoustic Perturbations & Corpus Generator`, `TTSManager`?**
  _High betweenness centrality (0.025) - this node is a cross-community bridge._