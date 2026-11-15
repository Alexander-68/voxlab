# Graph Report - voxlab  (2026-11-16)

## Corpus Check
- 92 files · ~112,892 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 2 file(s) not represented in the graph (top: (none) 1, .css 1)

## Summary
- 887 nodes · 2686 edges · 51 communities (35 shown, 16 thin omitted)
- Extraction: 80% EXTRACTED · 20% INFERRED · 0% AMBIGUOUS · INFERRED: 530 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `4bf9e8d8`
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
- config.go
- .SynthesizeStream
- .routes
- DSP Audio Preprocessor Controls
- testing.T
- SherpaRunner
- Server
- dictation.go
- Voice Interaction State Machine
- review-261115.md
- ASRStream
- SimulatorEngine
- .processIncomingAudio
- EvalManifest
- Acoustic Perturbations & Corpus Generator
- NewServer
- findSherpaBin
- TTSManager
- formatDictationPhraseInternal
- WebSocket Client Connection Lifecycle
- SpeechEnhancer
- .BroadcastJSON
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
- dialTestClient
- TestSetTTSModelOnlyAcceptsInstalledModels
- TestCalculateCER
- optimizeStreamingClauses

## God Nodes (most connected - your core abstractions)
1. `Server` - 79 edges
2. `SherpaRunner` - 38 edges
3. `VoiceClient` - 35 edges
4. `setupTestServer()` - 31 edges
5. `NewServer()` - 28 edges
6. `SpeechEngine` - 25 edges
7. `StateMachine` - 21 edges
8. `DSPProcessor` - 20 edges
9. `windowsSpeechEnhancer` - 17 edges
10. `NewSherpaRunner()` - 17 edges

## Surprising Connections (you probably didn't know these)
- `Top Issues` --references--> `LoadConfig()`  [INFERRED]
  review-261115.md → pkg/config/config.go
- `Quick Wins` --references--> `escapeHtml()`  [INFERRED]
  review-261115.md → web/js/bench-ui.js
- `Cross-Cutting Concerns` --references--> `EncodeWAV()`  [INFERRED]
  review-261115.md → pkg/audio/wav.go
- `Go build / test artifacts` --references--> `escapeHtml()`  [INFERRED]
  review-diff-261115.md → web/js/bench-ui.js
- `Top Issues` --references--> `getFreeTCPPort()`  [INFERRED]
  review-261115.md → pkg/engine/warm_asr.go

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

## Communities (51 total, 16 thin omitted)

### Community 0 - "server.go"
Cohesion: 0.15
Nodes (11): cGeneratedAudio, cKittenModelConfig, cKokoroModelConfig, cMatchaModelConfig, cOfflineTtsConfig, cOfflineTtsModelConfig, cPocketModelConfig, cSupertonicModelConfig (+3 more)

### Community 1 - "Evaluation Runner & Manifest"
Cohesion: 0.07
Nodes (46): BaseUtterance, EvalItem, MatrixVariation, DimensionGroup, ItemResult, LatencyPercentiles, LatencyStatistics, RunConfig (+38 more)

### Community 2 - "sherpa_runner.go"
Cohesion: 0.23
Nodes (7): TestMissingModelIsNotReportedAsNeural(), FindInstalledKokoroModels(), findKokoroModelDir(), isRegularFile(), isValidKokoroDir(), TestFindKokoroModelDir(), TestKokoroINT8Model()

### Community 3 - "Embedded Voice Architecture Reference"
Cohesion: 0.07
Nodes (31): Piper TTS Pipeline, PipeWire Audio Server Subsystem, Semantic Intent Router & Distractor Suppression, Silero VAD & Energy Gating, Streaming Zipformer ASR, Host Voice Daemon Reference, Wake-Word Detector, Annotation Dictation Mode (+23 more)

### Community 5 - "VoxLab Architecture & System Specifications"
Cohesion: 0.07
Nodes (25): Ref Architectural Ideas, VoxLab App Specification, Dynamic Model Switching Architecture, Sherpa-ONNX Metadata Requirements, Acoustic Benchmark System, Dual Audio Input Layer, Kokoro TTS Engine, ONNX Metadata Helper Script (+17 more)

### Community 6 - "Audio Capture Sources & Devices"
Cohesion: 0.10
Nodes (12): AudioSource, tailWriter, chunkInterval(), detectWindowsDirectShowMic(), HostNativeSource, WavFileSource, WebSocketSource, NewWavFileSource() (+4 more)

### Community 7 - "config.go"
Cohesion: 0.17
Nodes (17): main(), pct(), printLatencyLine(), printReportSummary(), run(), main(), AppConfig, AudioConfig (+9 more)

### Community 8 - ".SynthesizeStream"
Cohesion: 0.06
Nodes (34): main(), stubWarmTTS, TTSChunk, TTSResult, WarmTTS, windowsWarmTTS, BytesToFloat32PCM(), BytesToInt16PCM() (+26 more)

### Community 11 - "testing.T"
Cohesion: 0.15
Nodes (26): TestCalculateWER(), TestEvalReportRejectsUnknownTiers(), TestFailedSaveKeepsDraft(), TestSaveDraftStoresTextAndClearsDraft(), TestDefaultBindIsLoopback(), TestListenReportsATakenPort(), TestOversizedRequestsAreRefused(), TestRESTRejectsCrossSiteRequests() (+18 more)

### Community 12 - "SherpaRunner"
Cohesion: 0.17
Nodes (10): SherpaRunner, ASRInstalled(), InstalledStreamingModels(), NewSherpaRunner(), resolveModelDir(), TestKokoroFP16Model(), TestSecondPassPerMode(), TestSynthesizeChinese() (+2 more)

### Community 14 - "dictation.go"
Cohesion: 0.17
Nodes (20): CommandCatalog, CommandEntry, IntentDecision, MatchCandidate, DefaultCatalog(), LoadCatalog(), Tokenize(), calculateSimilarity() (+12 more)

### Community 15 - "Voice Interaction State Machine"
Cohesion: 0.16
Nodes (7): State, StateMachine, NewStateMachine(), TestFinishProcessOnlyFromProcess(), TestStateMachine(), asrWorkItem, StateEvent

### Community 16 - "review-261115.md"
Cohesion: 0.05
Nodes (33): truncateString(), getFreeTCPPort(), TestGetFreeTCPPort(), toCString(), Cross-Cutting Concerns, Module Hotspots, Project-Level Review Summary, Quick Wins (+25 more)

### Community 17 - "ASRStream"
Cohesion: 0.20
Nodes (5): ASRResult, ASRStream, containsCJK(), TestCutSegmentKeepsEndPad(), normalizeCase()

### Community 18 - "SimulatorEngine"
Cohesion: 0.12
Nodes (3): SimulatorEngine, SpeechEngine, fakeEngine

### Community 20 - "EvalManifest"
Cohesion: 0.43
Nodes (7): main(), EvalManifest, LoadManifest(), GenerateTier1Corpus(), GenerateTier1Corpus(), GenerateTier2Corpus(), LoadManifest()

### Community 21 - "Acoustic Perturbations & Corpus Generator"
Cohesion: 0.20
Nodes (16): CalculateRMS(), TestCalculateRMS(), AddNoiseAtSNR(), ApplyReverb(), GenerateBabbleNoise(), GeneratePinkNoise(), GenerateTypingNoise(), GenerateWhiteNoise() (+8 more)

### Community 22 - "NewServer"
Cohesion: 0.17
Nodes (17): NewHostNativeSource(), TestHostNativeCapture(), NewDSPProcessor(), TestDSPHighPassAndAGC(), TestDSPNoiseGateAndEcho(), TestRingBuffer(), NewRingBuffer(), NewSimulatorEngine() (+9 more)

### Community 23 - "findSherpaBin"
Cohesion: 0.18
Nodes (12): daemon, WarmASR, WarmOffline, killWithParent(), startDaemon(), killWithParent(), StartWarmOffline(), findSherpaBin() (+4 more)

### Community 25 - "formatDictationPhraseInternal"
Cohesion: 0.25
Nodes (7): capitalizeAfterSentences(), FormatDictationPhrase(), formatDictationPhraseInternal(), replaceSpokenWord(), TestDictationAbbreviationsAndLoneMarks(), TestFormatDictationPhrase(), upperLastLetter()

### Community 26 - "WebSocket Client Connection Lifecycle"
Cohesion: 0.39
Nodes (3): newWSClient(), wsClient, wsFrame

### Community 27 - "SpeechEnhancer"
Cohesion: 0.08
Nodes (15): BypassEnhancer, cDenoisedAudio, cDenoiserModelConfig, cDpdfNetModelConfig, cGtcrnModelConfig, cOnlineSpeechDenoiserConfig, windowsSpeechEnhancer, SpeechEnhancer (+7 more)

### Community 28 - ".BroadcastJSON"
Cohesion: 0.29
Nodes (4): AppendDictationPhrase(), collapsePunctuation(), TestAppendDictationPhrase(), appendJSONLine()

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

### Community 47 - "dialTestClient"
Cohesion: 0.50
Nodes (5): dialTestClient(), TestCancelledRunLeavesNoReportAndAllowsANewRun(), TestDictationIsIsolatedFromOtherActions(), TestDisconnectDuringPlaybackClearsEchoMute(), waitUntil()

### Community 48 - "TestSetTTSModelOnlyAcceptsInstalledModels"
Cohesion: 0.67
Nodes (3): installKokoro(), TestSetTTSModelOnlyAcceptsInstalledModels(), TestUnavailableConfiguredModelFallsBackToDiscovery()

### Community 50 - "optimizeStreamingClauses"
Cohesion: 0.67
Nodes (3): optimizeStreamingClauses(), TestOptimizeStreamingClauses(), TestWarmTTSSentenceSplitting()

## Knowledge Gaps
- **24 isolated node(s):** `voxlab`, `Host Voice Daemon Reference`, `Web Audio Canvas Oscilloscope & Spectrum`, `Piper TTS Pipeline`, `PipeWire Audio Server Subsystem` (+19 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 116 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **16 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Server` connect `Server` to `server.go`, `Evaluation Runner & Manifest`, `Audio Capture Sources & Devices`, `.routes`, `DSP Audio Preprocessor Controls`, `testing.T`, `dictation.go`, `Voice Interaction State Machine`, `dialTestClient`, `SimulatorEngine`, `.processIncomingAudio`, `NewServer`, `TTSManager`, `WebSocket Client Connection Lifecycle`, `SpeechEnhancer`, `.BroadcastJSON`?**
  _High betweenness centrality (0.127) - this node is a cross-community bridge._
- **Are the 16 inferred relationships involving `setupTestServer()` (e.g. with `TestCancelledRunLeavesNoReportAndAllowsANewRun()` and `TestDictationIsIsolatedFromOtherActions()`) actually correct?**
  _`setupTestServer()` has 16 INFERRED edges - model-reasoned connections that need verification._
- **What connects `voxlab`, `Host Voice Daemon Reference`, `Web Audio Canvas Oscilloscope & Spectrum` to the rest of the system?**
  _24 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Evaluation Runner & Manifest` be split into smaller, more focused modules?**
  _Cohesion score 0.06688311688311688 - nodes in this community are weakly interconnected._
- **Why does `VoiceClient` connect `Web Client Voice Controller` to `review-261115.md`?**
  _High betweenness centrality (0.058) - this node is a cross-community bridge._
- **Should `Embedded Voice Architecture Reference` be split into smaller, more focused modules?**
  _Cohesion score 0.06507936507936508 - nodes in this community are weakly interconnected._
- **Why does `SherpaRunner` connect `SherpaRunner` to `server.go`, `sherpa_runner.go`, `Audio Capture Sources & Devices`, `.SynthesizeStream`, `.FlushASR`, `review-261115.md`, `ASRStream`, `SimulatorEngine`, `findSherpaBin`?**
  _High betweenness centrality (0.051) - this node is a cross-community bridge._