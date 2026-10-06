# Walkthrough: VoxLab Voice Pipeline Test Bench

VoxLab has been implemented as a complete, cross-platform voice pipeline test bench for web runtimes (WPE WebKit, embedded Chromium, Electron, desktop browsers) on **Windows** and **Linux** in **Go v1.27** and **vanilla HTML5/JS**.

---

## 1. Summary of Changes

### A. Go Daemon Backend (`cmd/`, `pkg/`)

- [`pkg/config/config.go`](file:///C:/Alex/voxlab/pkg/config/config.go): Unified configuration for audio sample rates (16kHz), DSP noise gate ($-42\text{ dBFS}$), $80\text{ Hz}$ high-pass filter, wake-word detection, intent thresholds, dictation timeouts, and Kokoro TTS models.
- [`pkg/audio/ringbuffer.go`](file:///C:/Alex/voxlab/pkg/audio/ringbuffer.go): Thread-safe circular audio ring buffer maintaining a $1.0\text{s}$ pre-roll buffer so words spoken immediately following the wake phrase are never clipped.
- [`pkg/audio/dsp.go`](file:///C:/Alex/voxlab/pkg/audio/dsp.go): Audio pre-processing including RMS energy calculation in dBFS, noise floor gating, first-order high-pass filtering ($80\text{ Hz}$ de-rumble), software AGC, and half-duplex echo suppression latching.
- [`pkg/audio/wav.go`](file:///C:/Alex/voxlab/pkg/audio/wav.go): 16-bit PCM RIFF WAV encoder/decoder and synthetic chord harmonic generator.
- [`pkg/audio/capture.go`](file:///C:/Alex/voxlab/pkg/audio/capture.go): Unified audio source abstraction supporting **Web UI Mic** (`web_ui_mic`), **Host Native/OS Mic** (`host_native_mic`), and **WAV Test Sample Injection** (`wav_file_injection`).
- [`pkg/intent/catalog.go`](file:///C:/Alex/voxlab/pkg/intent/catalog.go) & [`pkg/intent/matcher.go`](file:///C:/Alex/voxlab/pkg/intent/matcher.go): Multi-stage intent router supporting exact matches, verb aliases, distractor/out-of-domain conversational chatter suppression, negation detection, and entity slot extraction.
- [`pkg/statemachine/state.go`](file:///C:/Alex/voxlab/pkg/statemachine/state.go): Finite state machine managing voice turn lifecycles (`IDLE`, `COMMAND_LISTEN`, `PROCESS`, `CONFIRM`, `ANNOTATION_LISTEN`, `REVIEW`, `SPEAKING`, `FAULT`).
- [`pkg/engine/engine.go`](file:///C:/Alex/voxlab/pkg/engine/engine.go), [`pkg/engine/simulator.go`](file:///C:/Alex/voxlab/pkg/engine/simulator.go), & [`pkg/engine/sherpa_runner.go`](file:///C:/Alex/voxlab/pkg/engine/sherpa_runner.go): Pluggable speech engine supporting both live Sherpa-ONNX binaries and a built-in zero-dependency high-fidelity simulator.
- [`pkg/tts/kokoro.go`](file:///C:/Alex/voxlab/pkg/tts/kokoro.go): Multi-speaker Kokoro TTS manager with 8 speaker profiles (`af_heart`, `af_alloy`, `am_adam`, `am_fenrir`, etc.) and automated half-duplex mic muting during playback.
- [`pkg/server/server.go`](file:///C:/Alex/voxlab/pkg/server/server.go): High-throughput HTTP static server and WebSocket IPC bridge (`/ws`) dispatching binary PCM audio and typed JSON events.
- [`cmd/voxlab/main.go`](file:///C:/Alex/voxlab/cmd/voxlab/main.go): CLI entry point supporting `-port`, `-host`, `-engine`, `-models`, `-ui`, and `-config` flags with graceful signal handling.

### B. Frontend Test Bench Web UI (`web/`)

- [`web/index.html`](file:///C:/Alex/voxlab/web/index.html): Three-column responsive engineering console:
  1. **Audio In & DSP**: Source selector, Web mic toggle, live canvas oscilloscope and spectrum analyzer, VU meter with noise-gate open/closed LED, filter toggles, wake-word detector settings.
  2. **Speech Recognition Lab**: Tabbed workspace comparing **Voice Commands** (with live ASR streaming, matched intent cards, slot tags, distractor rejection box, and quick injections) with **Voice Dictations** (isolated spoken draft editor, word counter, and save/discard actions).
  3. **Kokoro TTS Studio & Telemetry**: Voice profile selector, speed slider, prompt editor with quick presets, embedded audio player, pipeline latency waterfall, and real-time IPC event log.
- [`web/css/style.css`](file:///C:/Alex/voxlab/web/css/style.css): Professional dark-mode design system with animated VU meters, glowing state indicators, and typography.
- [`web/js/audio-capture.js`](file:///C:/Alex/voxlab/web/js/audio-capture.js): Web Audio API manager capturing 16kHz PCM audio and rendering time-domain waves and FFT bars to canvas.
- [`web/js/voice-client.js`](file:///C:/Alex/voxlab/web/js/voice-client.js): WebSocket IPC protocol client with automatic reconnect.
- [`web/js/bench-ui.js`](file:///C:/Alex/voxlab/web/js/bench-ui.js): Interactive controller linking UI widgets to server events.

### C. Tooling & Assets (`data/`, `scripts/`, `cmd/gen_samples/`)

- [`data/commands.json`](file:///C:/Alex/voxlab/data/commands.json): Canonical command catalog with 8 actions, aliases, slots, and conversational distractor phrases.
- [`data/test_samples/`](file:///C:/Alex/voxlab/data/test_samples/): Sample WAV files (`chime_alert.wav`, `voice_test.wav`) for deterministic audio injection.
- [`scripts/download_models.ps1`](file:///C:/Alex/voxlab/scripts/download_models.ps1) & [`scripts/download_models.sh`](file:///C:/Alex/voxlab/scripts/download_models.sh): Automation scripts to download pre-trained Sherpa-ONNX Kokoro TTS and Zipformer ASR models.
- [`README.md`](file:///C:/Alex/voxlab/README.md): Comprehensive system documentation, architecture diagrams, CLI guides, and IPC protocol reference.

---

## 2. Test & Verification Results

### Automated Unit & Integration Tests

Executed `go test -v ./...` across all packages:

| Package            | Test                                    | Result                                                                                            |
|:------------------ |:--------------------------------------- |:------------------------------------------------------------------------------------------------- |
| `pkg/audio`        | `TestCalculateRMS`                      | **PASS** (Zero audio vs full-scale DC verification)                                               |
| `pkg/audio`        | `TestRingBuffer`                        | **PASS** (1.0s pre-roll buffer overflow & chronologic ordering)                                   |
| `pkg/audio`        | `TestDSPNoiseGateAndEcho`               | **PASS** ($-40\text{ dBFS}$ noise floor gate & half-duplex mute latch)                            |
| `pkg/audio`        | `TestWAVEncodeDecode`                   | **PASS** (16-bit PCM WAV roundtrip & chime generation)                                            |
| `pkg/engine`       | `TestSimulatorEngine`                   | **PASS** (Quiet vs loud VAD detection & Kokoro synthetic synthesis)                               |
| `pkg/intent`       | `TestIntentMatcher`                     | **PASS** (Exact match, alias matching, slot extraction, distractor rejection, negation detection) |
| `pkg/statemachine` | `TestStateMachine`                      | **PASS** (Wake trigger, endpointing, dictation mode isolation, cancel)                            |
| `pkg/server`       | `TestServerRESTEndpoints`               | **PASS** (`/api/config`, `/api/catalog`, `/api/voices`, `/api/tts` WAV output)                    |
| `pkg/server`       | `TestServerWebSocketHandshakeAndAction` | **PASS** (WS handshake, initial state event, action dispatch, `intent_matched` verification)      |

**Overall Test Result:** `100% PASS` across all test suites in 1.4s.

### Live Server Verification

1. Binary compiled successfully with Go v1.27 (`bin/voxlab.exe`).
2. Server launched on port 8085.
3. Tested `GET /api/config`: returned valid JSON configuration.
4. Tested `GET /api/voices`: returned 8 Kokoro speaker profiles (`af_heart`, `af_alloy`, `am_adam`...).
5. Tested `POST /api/tts`: synthesized 143,564 bytes of valid `audio/wav` payload for *"Welcome to VoxLab. Voice pipeline operational."*.
6. Tested `GET /`: served HTML5 test bench UI with HTTP status 200.

---

## 3. How to Run VoxLab

```bash
# In C:\Alex\voxlab
go run ./cmd/voxlab
```

Open **http://localhost:8080** in any browser.

To run with real Sherpa-ONNX neural models:

```powershell
# Windows
.\scripts\download_models.ps1
go run ./cmd/voxlab -engine=sherpa
```

```bash
# Linux
./scripts/download_models.sh
go run ./cmd/voxlab -engine=sherpa
```
