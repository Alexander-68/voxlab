# Graph Report - voxlab  (2026-11-16)

## Corpus Check
- 91 files · ~109,155 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 2 file(s) not represented in the graph (top: (none) 1, .css 1)

## Summary
- 853 nodes · 2545 edges · 54 communities (39 shown, 15 thin omitted)
- Extraction: 80% EXTRACTED · 20% INFERRED · 0% AMBIGUOUS · INFERRED: 515 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `68dff075`
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
- SpeechEngine
- Persistent Warm TTS Synthesis
- HTTP Web Server & Networking
- DSP Audio Preprocessor Controls
- testing.T
- NewSherpaRunner
- ASRStream
- NewServer
- Voice Interaction State Machine
- bench-ui.js
- denoiser_windows.go
- SherpaRunner
- RingBuffer
- matcher.go
- Acoustic Perturbations & Corpus Generator
- NewSimulatorEngine
- TestSetTTSModelOnlyAcceptsInstalledModels
- TTSManager
- config.go
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
- TestCalculateCER
- patch_kokoro_fp16_metadata.py
- lifecycle_test.go
- formatDictationPhraseInternal
- optimizeStreamingClauses
- TestKokoroLang
- normalizeTTSText
- NewStateMachine

## God Nodes (most connected - your core abstractions)
1. `Server` - 79 edges
2. `SherpaRunner` - 37 edges
3. `VoiceClient` - 35 edges
4. `setupTestServer()` - 31 edges
5. `NewServer()` - 28 edges
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
- `Top Issues` --references--> `getFreeTCPPort()`  [INFERRED]
  review-261115.md → pkg/engine/warm_asr.go
- `Cross-Cutting Concerns` --references--> `truncateString()`  [INFERRED]
  review-261115.md → cmd/eval_bench/main.go
- `Cross-Cutting Concerns` --references--> `EncodeWAV()`  [INFERRED]
  review-261115.md → pkg/audio/wav.go

## Import Cycles
- 2-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/smoke_test.go`
- 2-file cycle: `pkg/evaluation/tier2_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/tier2_test.go`
- 2-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/server.go -> pkg/evaluation/smoke_test.go`
- 2-file cycle: `pkg/evaluation/tier2_test.go -> pkg/server/server.go -> pkg/evaluation/tier2_test.go`
- 2-file cycle: `pkg/evaluation/smoke_test.go -> pkg/evaluation/tier2_test.go -> pkg/evaluation/smoke_test.go`
- 3-file cycle: `pkg/evaluation/smoke_test.go -> pkg/evaluation/tier2_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/smoke_test.go`
- 3-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/tier2_test.go -> pkg/evaluation/smoke_test.go`
- 3-file cycle: `pkg/evaluation/smoke_test.go -> pkg/evaluation/tier2_test.go -> pkg/server/server.go -> pkg/evaluation/smoke_test.go`
- 3-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/server.go -> pkg/evaluation/tier2_test.go -> pkg/evaluation/smoke_test.go`
- 4-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/server.go -> pkg/evaluation/tier2_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/smoke_test.go`
- 4-file cycle: `pkg/evaluation/smoke_test.go -> pkg/server/eval_handler.go -> pkg/evaluation/tier2_test.go -> pkg/server/server.go -> pkg/evaluation/smoke_test.go`

## Hyperedges (group relationships)
- **Voice Command Processing Architecture** — readme_audio_input_layer, readme_wake_word_detector, readme_voice_commands_pipeline, readme_semantic_intent_router, web_index_commands_tab [EXTRACTED 1.00]
- **Acoustic Benchmark & Evaluation Framework** — readme_acoustic_eval_bench, todo_acoustic_matrix_findings, todo_audio_resampler, web_index_acoustic_eval_tab [INFERRED 0.95]
- **Two-Pass Speech Recognition Pipeline** — todo_nemotron_35_upgrade, todo_parakeet_ultra_upgrade, readme_voice_dictations_pipeline, web_index_dictation_tab [INFERRED 0.95]
- **** — ref_synthetic_evaluation_bench_synthetic_benchmark_harness, ref_synthetic_evaluation_bench_two_pass_asr, ref_synthetic_evaluation_bench_two_tier_test_suite, ref_synthetic_evaluation_bench_tier1_golden_cases, ref_synthetic_evaluation_bench_quantitative_metrics [INFERRED]
- **** — ref_linux_wpe_voice_recipe_interaction_state_machine, ref_linux_wpe_voice_recipe_command_catalog_slots, ref_linux_wpe_voice_recipe_annotation_dictation_mode, ref_linux_wpe_voice_recipe_echo_control_policy [INFERRED]
- **** — ref_embedded_voice_architecture_recipe_silero_vad, ref_embedded_voice_architecture_recipe_wake_word_detector, ref_embedded_voice_architecture_recipe_streaming_asr_zipformer, ref_embedded_voice_architecture_recipe_semantic_intent_router, ref_embedded_voice_architecture_recipe_piper_tts [INFERRED]

## Communities (54 total, 15 thin omitted)

### Community 0 - "server.go"
Cohesion: 0.14
Nodes (11): cGeneratedAudio, cKittenModelConfig, cKokoroModelConfig, cMatchaModelConfig, cOfflineTtsConfig, cOfflineTtsModelConfig, cPocketModelConfig, cSupertonicModelConfig (+3 more)

### Community 1 - "Evaluation Runner & Manifest"
Cohesion: 0.09
Nodes (37): EvalItem, EvalManifest, DimensionGroup, ItemResult, LatencyPercentiles, LatencyStatistics, RunConfig, SNRCurvePoint (+29 more)

### Community 2 - "sherpa_runner.go"
Cohesion: 0.19
Nodes (10): detectKokoroVersion(), FindInstalledKokoroModels(), findKokoroModelDir(), isRegularFile(), isValidKokoroDir(), mapVoiceToSID(), TestFindKokoroModelDir(), TestKokoroFP16Model() (+2 more)

### Community 3 - "Embedded Voice Architecture Reference"
Cohesion: 0.07
Nodes (31): Piper TTS Pipeline, PipeWire Audio Server Subsystem, Semantic Intent Router & Distractor Suppression, Silero VAD & Energy Gating, Streaming Zipformer ASR, Host Voice Daemon Reference, Wake-Word Detector, Annotation Dictation Mode (+23 more)

### Community 5 - "VoxLab Architecture & System Specifications"
Cohesion: 0.07
Nodes (25): Ref Architectural Ideas, VoxLab App Specification, Dynamic Model Switching Architecture, Sherpa-ONNX Metadata Requirements, Acoustic Benchmark System, Dual Audio Input Layer, Kokoro TTS Engine, ONNX Metadata Helper Script (+17 more)

### Community 6 - "Audio Capture Sources & Devices"
Cohesion: 0.11
Nodes (9): AudioSource, tailWriter, detectWindowsDirectShowMic(), HostNativeSource, WavFileSource, WebSocketSource, NewWavFileSource(), NewWebSocketSource() (+1 more)

### Community 7 - "SpeechEngine"
Cohesion: 0.13
Nodes (9): TTSChunk, TTSResult, TestResample(), Resample(), SpeechEngine, TTSRequest, GenerateTier1Corpus(), LoadOrSynthesizeClip() (+1 more)

### Community 8 - "Persistent Warm TTS Synthesis"
Cohesion: 0.09
Nodes (15): truncateString(), stubWarmTTS, WarmTTS, windowsWarmTTS, EncodeWAV(), kokoroLexicons(), warmTTSKey(), float32SliceToTTSResult() (+7 more)

### Community 9 - "HTTP Web Server & Networking"
Cohesion: 0.13
Nodes (3): isLoopbackName(), EvalState, Server

### Community 10 - "DSP Audio Preprocessor Controls"
Cohesion: 0.10
Nodes (3): DSPProcessor, downsampleWave(), NewServer()

### Community 11 - "testing.T"
Cohesion: 0.12
Nodes (31): TestSpeechEnhancerFactoryAndProcessing(), TestCalculateWER(), requireDeepRun(), TestMultilingualDictations(), TestTier2DeepAcousticMatrix(), TestAppendDictationPhrase(), TestEvalReportRejectsUnknownTiers(), TestFailedSaveKeepsDraft() (+23 more)

### Community 12 - "NewSherpaRunner"
Cohesion: 0.12
Nodes (14): daemon, WarmASR, WarmOffline, killWithParent(), startDaemon(), killWithParent(), StartWarmOffline(), containsCJK() (+6 more)

### Community 13 - "ASRStream"
Cohesion: 0.24
Nodes (3): ASRResult, ASRStream, TestCutSegmentKeepsEndPad()

### Community 14 - "NewServer"
Cohesion: 0.38
Nodes (10): CommandCatalog, CommandEntry, DefaultCatalog(), LoadCatalog(), NewIntentMatcher(), TestCatalogExamplesAvoidRejectionWords(), testCatalogs(), TestIntentMatcher() (+2 more)

### Community 15 - "Voice Interaction State Machine"
Cohesion: 0.20
Nodes (4): State, StateMachine, asrWorkItem, StateEvent

### Community 16 - "bench-ui.js"
Cohesion: 0.13
Nodes (12): escapeHtml(), getLanguageGroup(), handleCustomDictInject(), handleKwsToggle(), logEvent(), populateVoicesDropdown(), renderCasesTable(), renderEvalReport() (+4 more)

### Community 17 - "denoiser_windows.go"
Cohesion: 0.13
Nodes (8): cDenoisedAudio, cDenoiserModelConfig, cDpdfNetModelConfig, cGtcrnModelConfig, cOnlineSpeechDenoiserConfig, windowsSpeechEnhancer, newPlatformSpeechEnhancer(), toCStringEnhancer()

### Community 19 - "RingBuffer"
Cohesion: 0.25
Nodes (3): TestRingBuffer(), RingBuffer, NewRingBuffer()

### Community 20 - "matcher.go"
Cohesion: 0.29
Nodes (8): IntentDecision, MatchCandidate, Tokenize(), calculateSimilarity(), IntentMatcher, HasNegation(), IsNarration(), TestTokenizeNonASCII()

### Community 21 - "Acoustic Perturbations & Corpus Generator"
Cohesion: 0.13
Nodes (23): BaseUtterance, MatrixVariation, NewHostNativeSource(), TestHostNativeCapture(), CalculateRMS(), TestCalculateRMS(), AddNoiseAtSNR(), ApplyReverb() (+15 more)

### Community 22 - "NewSimulatorEngine"
Cohesion: 0.29
Nodes (10): NewDSPProcessor(), TestDSPHighPassAndAGC(), TestDSPNoiseGateAndEcho(), NewSimulatorEngine(), TestSimulatorEngine(), TestLoadOrSynthesizeClipRejectsSimulatorTones(), NewTTSManager(), TestTTSManagerEchoSuppression() (+2 more)

### Community 23 - "TestSetTTSModelOnlyAcceptsInstalledModels"
Cohesion: 0.67
Nodes (3): installKokoro(), TestSetTTSModelOnlyAcceptsInstalledModels(), TestUnavailableConfiguredModelFallsBackToDiscovery()

### Community 24 - "TTSManager"
Cohesion: 0.20
Nodes (4): VoiceProfile, AppendDictationPhrase(), appendJSONLine(), TTSManager

### Community 25 - "config.go"
Cohesion: 0.07
Nodes (38): main(), pct(), printLatencyLine(), printReportSummary(), run(), main(), main(), main() (+30 more)

### Community 26 - "WebSocket Client Connection Lifecycle"
Cohesion: 0.39
Nodes (3): newWSClient(), wsClient, wsFrame

### Community 27 - "SpeechEnhancer"
Cohesion: 0.13
Nodes (6): BypassEnhancer, SpeechEnhancer, NewBypassEnhancer(), NewSpeechEnhancer(), newPlatformSpeechEnhancer(), TestBypassEnhancer()

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
Cohesion: 0.33
Nodes (3): inspect_metadata(), main(), patch_metadata()

### Community 48 - "lifecycle_test.go"
Cohesion: 0.67
Nodes (5): dialTestClient(), TestCancelledRunLeavesNoReportAndAllowsANewRun(), TestDictationIsIsolatedFromOtherActions(), TestDisconnectDuringPlaybackClearsEchoMute(), waitUntil()

### Community 49 - "formatDictationPhraseInternal"
Cohesion: 0.40
Nodes (4): FormatDictationPhrase(), formatDictationPhraseInternal(), replaceSpokenWord(), TestFormatDictationPhrase()

### Community 50 - "optimizeStreamingClauses"
Cohesion: 0.67
Nodes (3): optimizeStreamingClauses(), TestOptimizeStreamingClauses(), TestWarmTTSSentenceSplitting()

## Knowledge Gaps
- **24 isolated node(s):** `voxlab`, `Host Voice Daemon Reference`, `Web Audio Canvas Oscilloscope & Spectrum`, `Piper TTS Pipeline`, `PipeWire Audio Server Subsystem` (+19 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 125 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **15 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Server` connect `HTTP Web Server & Networking` to `server.go`, `Evaluation Runner & Manifest`, `Audio Capture Sources & Devices`, `SpeechEngine`, `DSP Audio Preprocessor Controls`, `testing.T`, `Voice Interaction State Machine`, `lifecycle_test.go`, `RingBuffer`, `matcher.go`, `TTSManager`, `WebSocket Client Connection Lifecycle`, `SpeechEnhancer`?**
  _High betweenness centrality (0.132) - this node is a cross-community bridge._
- **Are the 16 inferred relationships involving `setupTestServer()` (e.g. with `TestCancelledRunLeavesNoReportAndAllowsANewRun()` and `TestDictationIsIsolatedFromOtherActions()`) actually correct?**
  _`setupTestServer()` has 16 INFERRED edges - model-reasoned connections that need verification._
- **What connects `voxlab`, `Host Voice Daemon Reference`, `Web Audio Canvas Oscilloscope & Spectrum` to the rest of the system?**
  _24 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `server.go` be split into smaller, more focused modules?**
  _Cohesion score 0.14306753629286767 - nodes in this community are weakly interconnected._
- **Why does `SherpaRunner` connect `SherpaRunner` to `server.go`, `sherpa_runner.go`, `Audio Capture Sources & Devices`, `SpeechEngine`, `Persistent Warm TTS Synthesis`, `NewSherpaRunner`, `ASRStream`, `config.go`?**
  _High betweenness centrality (0.053) - this node is a cross-community bridge._
- **Should `Evaluation Runner & Manifest` be split into smaller, more focused modules?**
  _Cohesion score 0.09292929292929293 - nodes in this community are weakly interconnected._
- **Should `Embedded Voice Architecture Reference` be split into smaller, more focused modules?**
  _Cohesion score 0.06507936507936508 - nodes in this community are weakly interconnected._