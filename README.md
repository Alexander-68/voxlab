# VoxLab

> **Tagline:** The voice pipeline test bench for web runtimes.

**VoxLab** is an engineering test bench to develop, benchmark, inspect, and test voice input (ASR + Wake-Word + Semantic Intent Routing) and voice output (TTS via Kokoro) architectures for web runtimes (WPE WebKit, embedded Chromium, Electron, and desktop browsers) on **Windows** and **Linux**.

---

## Architecture Overview

```
 ┌────────────────────────────────────────────────────────────────────────┐
 │                      AUDIO INPUT LAYER                                 │
 ├──────────────────────────┬─────────────────────────┬───────────────────┤
 │ Web UI Mic (getUserMedia)│ Host Native/OS Mic (ALSA│ Test WAV Injection│
 └─────────────┬────────────┴────────────┬────────────┴─────────┬─────────┘
               │                         │                      │
               ▼                         ▼                      ▼
 ┌────────────────────────────────────────────────────────────────────────┐
 │                      VOXLAB HOST GO DAEMON (Go v1.27)                  │
 │                                                                        │
 │  [DSP Preprocessing]  RMS Energy Floor (-42 dBFS) + 80Hz High-Pass +   │
 │                       Software AGC + Half-Duplex Echo Latch            │
 │         │                                                              │
 │         ▼                                                              │
 │  [Pre-roll Ring Buffer] (1.0s circular buffer preserving early words)  │
 │         │                                                              │
 │         ▼                                                              │
 │  [Stage 1: Wake-Word Detector] (Sherpa-ONNX KWS / "Hey VoxLab")        │
 │         │ (Triggered / Manual Activation)                              │
 │         ▼                                                              │
 │  [Stage 2: Mode Gate]                                                  │
 │         ├──────────────────────────────┬───────────────────────────────┤
 │         ▼                              ▼                               │
 │  [Purpose A: Voice Commands]    [Purpose B: Voice Dictations]          │
 │  - Kroko+Parakeet ASR           - Streaming continuous transcription   │
 │  - Semantic Intent Router       - Silence endpointing (800ms)          │
 │  - Distractor / Chatter Filter  - Isolated Annotation Draft Editor     │
 │  - State Machine Policy checks  - No command execution allowed        │
 │                                                                        │
 │  [Voice Output (TTS)] Kokoro Multi-Speaker Offline Synthesis           │
 │  - Speaker Voices: af_heart, af_alloy, am_adam, am_fenrir, etc.       │
 │  - Half-Duplex Echo Gate: Mutes mic input during speech playback       │
 └───────────────────────────────────┬────────────────────────────────────┘
                                     │
           WebSocket IPC (ws://localhost:8080/ws) + HTTP Static UI
                                     ▼
 ┌────────────────────────────────────────────────────────────────────────┐
 │                      WEB RUNTIME / TEST BENCH UI                       │
 │  - High-density Plain HTML5 + CSS + JavaScript (no npm, zero bloat)    │
 │  - Real-time oscilloscope & frequency spectrum analyzer                │
 │  - VU meter with noise-gate open/closed LED                            │
 │  - Ranked intent confidence & separation margin meters                 │
 │  - Annotation draft review card (word count, save/copy/discard)        │
 │  - Kokoro TTS Studio & Audio Player with latency waterfall             │
 │  - Real-time JSON IPC event log inspector                              │
 └────────────────────────────────────────────────────────────────────────┘
```

---

## Key Features

1. **Dual Audio Input**:
   - **Web UI Mic**: Captured by the browser and streamed as 16kHz PCM chunks over WebSocket.
   - **Host Go App Mic**: Direct microphone capture on the host operating system (crucial for headless or appliance deployments like WPE WebKit where WPE has no mic access).
   - **Test WAV Injection**: Stream pre-recorded test audio clips to evaluate noise rejection and ASR deterministically.

2. **Noise Cancelling & DSP**:
   - **Noise Gate**: Drops chunks below configured threshold (default `-42 dBFS`) to reject room chatter.
   - **High-Pass Filter**: 80Hz cut-off filter removing sub-audible environmental rumble.
   - **Software AGC**: Smooth gain normalization for quiet speakers.
   - **Half-Duplex Echo Latch**: Suppresses microphone input while TTS is speaking to prevent self-triggering loops.

3. **Wake-Word Spotting (KWS)**:
   - Supports phrases such as *"Hey VoxLab"* or *"Hey Console"*.
   - **1.0-second circular pre-roll buffer** ensures speech spoken immediately following the wake word is not lost or clipped.

4. **Dual-Purpose Speech Processing**:
   - **Purpose 1: Voice Commands**:
     - Two-pass speech recognition: streaming Nemotron-3.5 560ms / Kroko Zipformer (live partials, responsive endpointing) + optional Parakeet Ultra 0.6B second pass (punctuated, higher accuracy).
     - **Second Pass UI Control (Command Mode)**: Disabled by default for minimum latency (<150ms command execution directly from streaming tokens), but toggleable via UI checkbox.
     - Streaming model switchable at runtime in the UI (Nemotron-3.5 560ms default / Kroko).
     - Semantic Intent Router with cosine / token scoring against a configurable JSON catalog (`data/commands.json`).
     - Distractor suppression (rejects conversational chatter quietly).
     - Negation detection (e.g. *"do not stop recording"* is rejected).
     - Multi-turn confirmation state machine (`IDLE` $\to$ `COMMAND_LISTEN` $\to$ `PROCESS` $\to$ `CONFIRM` $\to$ `IDLE`).
     - 600ms responsive silence endpointing.
   - **Purpose 2: Voice Dictations**:
     - **Live Partial Speech Streaming**: Each listening session holds one live stream to the resident recognizer; interim hypotheses arrive every ~330ms (Nemotron-3.5 560ms default or Kroko).
     - **Natural Pause Endpointing & Two-Pass Refinement**: Recognizer endpoints phrases on natural speech pauses (~720ms `dictation.silence_endpoint_ms`).
     - **Second Pass UI Control (Dictation Mode)**: Enabled by default. Resident **Parakeet Ultra (INT8)** re-transcribes completed phrases in the background, adding sentence punctuation (`.`), proper casing, and enhanced vocabulary. Can be toggled in the UI.
     - **Noise & Keyboard Clack Drop**: When typing noise or background sound triggers VAD without intelligible speech, the test bench cleanly resets back to listening state without hanging.
     - **Leading Syllable Pre-roll**: 360ms circular buffer preserves initial consonants and word onsets.
     - Strictly isolated from command execution: words spoken during dictation can never accidentally trigger system commands.
     - Draft review workflow: edit, approve, copy, or discard.

5. **Voice Synthesis (Kokoro TTS)**:
   - High-quality offline multi-speaker neural speech synthesis.
   - **Persistent In-Memory Warm Engine**: Eliminates process startup cold-start overhead (~1.8s) by keeping models cached in RAM via dynamic C-API linking, delivering **sub-second latency (<1.0s)**.
   - **Multi-Threading & Hardware Providers**: Runs inference on 4 threads by default (configurable via `-threads`), with selectable execution providers (`-provider=cpu`, `-provider=directml`, `-provider=cuda`).
   - **Multi-Model Dynamic Selection**: Switch seamlessly between Kokoro v1.1 (103 voices), v1.0 (54 voices), and v0.19 (11 voices) at runtime without service restarts.
   - **Flexible Quantization & Precision**: Supports **FP32** (full precision), **INT8** (quantized compact), and **FP16** (half-precision `model.fp16.onnx`, ~163MB, sub-second latency).
   - Speed adjustment ($0.7\times$ to $1.5\times$).
   - Output to browser speaker AND host speaker with automated half-duplex echo gate suppression.

6. **Web UI Test Bench Controls & CPU Optimization**:
   - **Optional Live Oscilloscope & Audio Meter**: A dedicated toggle disables canvas `requestAnimationFrame` loops and DOM meter updates completely, freeing 100% of visualizer CPU cycles.
   - **Decoupled ASR Pipeline Worker**: The audio processing loop and 30Hz WebSocket metering run on an independent thread, ensuring audio visualizers and level meters never stall during speech recognition.

7. **Dual Engine Mode**:
   - **Simulator Mode (Default)**: Zero-dependency Go engine with realistic latencies, simulated ASR partials, and synthetic audio for instant development and testing.
   - **Sherpa-ONNX Mode**: Executes persistent, resident in-memory daemon (`sherpa-onnx-online-websocket-server`) with a 4-thread streaming model for live partials, plus a resident `sherpa-onnx-offline-websocket-server` (Parakeet Ultra) that re-transcribes each endpointed phrase (~0.1 s per second of speech on CPU).

---

## Installed Models & Precision Variants

VoxLab auto-discovers all installed Kokoro models located in `models/`:

| Model Identifier | Precision | Weights File | Description / Notes | Footprint |
| :--- | :--- | :--- | :--- | :--- |
| `kokoro-multi-lang-v1_1` | FP32 | `model.onnx` | 103 voices | ~325 MB |
| `kokoro-multi-lang-v1_1 (INT8)` | **INT8** | `model.int8.onnx` | 103 voices | ~114 MB |
| `kokoro-multi-lang-v1_0` | FP32 | `model.onnx` | 54 voices | ~325 MB |
| `kokoro-multi-lang-v1_0 (FP16)` | **FP16** | `model.fp16.onnx` | 54 voices | ~163 MB |
| `kokoro-en-v0_19` | FP32 | `model.onnx` | 11 voices | ~345 MB |
| `nemotron-3.5-asr-streaming-0.6b` | **INT8** | `encoder.int8.onnx` | 1st Pass Streaming (Default, 40+ langs, 560ms chunk) | ~475 MB |
| `streaming-zipformer-en-kroko` | FP32 | `encoder.onnx` | 1st Pass Streaming (Lightweight, ~65M params) | ~120 MB |
| `nemo-parakeet-tdt-0.6b-ultra` | **INT8** | `encoder.int8.onnx` | 2nd Pass Offline Refinement (Moondream Ultra) | ~487 MB |

Model weight variants (`model.int8.onnx`, `model.fp16.onnx`) can be co-located within the same model folder, sharing lexicons and voice embeddings. VoxLab automatically exposes them with their respective suffix (`(INT8)`, `(FP16)`) in the Web UI, REST API (`/api/voices`, `/api/models`), and WebSocket actions.

See [`models-tts.md`](file:///C:/Alex/voxlab/models-tts.md) for full architecture and directory details.

---

## ONNX Metadata Helper Script

Sherpa-ONNX requires specific metadata properties (`sample_rate`, `model_type`, `version`, `has_espeak`, speaker maps) embedded within the ONNX file. Raw checkpoints downloaded from HuggingFace (e.g. `hexgrad/Kokoro-82M`) often lack these properties.

VoxLab provides a helper script [`scripts/patch_kokoro_fp16_metadata.py`](file:///C:/Alex/voxlab/scripts/patch_kokoro_fp16_metadata.py) to inspect and inject the missing properties:

```bash
# Check existing metadata
python scripts/patch_kokoro_fp16_metadata.py --check models/kokoro-multi-lang-v1_0/model.fp16.onnx

# Copy Sherpa-ONNX metadata from model.onnx to model.fp16.onnx
python scripts/patch_kokoro_fp16_metadata.py models/kokoro-multi-lang-v1_0/model.fp16.onnx models/kokoro-multi-lang-v1_0/model.onnx
```

---

## Quick Start

### 1. Run with Go v1.27
```bash
# Clone and enter directory
cd C:\Alex\voxlab

# Run the test bench
go run ./cmd/voxlab
```
Open your browser to: **http://localhost:8080**

### 2. Command-Line Options
```text
  -port int        HTTP and WebSocket server port (default 8080)
  -host string     Host address to bind (default "127.0.0.1"; 0.0.0.0 exposes the mic and controls to the LAN, unauthenticated)
  -engine string   Speech engine: 'simulator' or 'sherpa' (default "simulator")
  -models string   Directory containing Sherpa-ONNX model files (default "models")
  -ui string       Directory containing static web UI assets (default "web")
  -config string   Path to config file (default "config.json")
```

---

## Installing Sherpa-ONNX Models (Optional)

To enable real neural model inference:

### On Windows (PowerShell):
```powershell
.\scripts\download_models.ps1
```

### On Linux (Bash):
```bash
chmod +x scripts/download_models.sh
./scripts/download_models.sh
```

Then start VoxLab in Sherpa mode:
```bash
go run ./cmd/voxlab -engine=sherpa -models=models
```

---

## Running Unit & Integration Tests

```bash
go test -v ./...

# On-demand Tier 2 matrix (about 10 minutes, audio is fed at real time)
VOXLAB_TIER2=1 go test ./pkg/evaluation/ -timeout 40m

# Same matrix from the CLI. Writes data/eval_report_tier-2-deep-matrix.json, the report the web UI loads,
# and exits 1 below the pass bar (Tier 1: 100%, Tier 2: 80%).
go run ./cmd/eval_bench -tier 2

# PowerShell script to run Tier 2 (supports -ForceRegenerate to re-synthesize audio samples, -Short, -Mode bench):
pwsh scripts/run_tier2_test.ps1 [-ForceRegenerate] [-Short] [-Mode bench|test]

# Regenerate missing corpus clips (existing WAVs are kept; delete one to resynthesize it)
go run ./cmd/gen_eval_samples -tier all
```
Dictation items may carry an `alt_text` with the same words in written form ("2 mg" for "two milligrams"); the closer of `text` and `alt_text` is scored.
Tests cover:
- Audio DSP (RMS calculation, high-pass filter, noise floor gating, half-duplex echo latch)
- Circular Ring Buffer (overflow, snapshot ordering)
- WAV Codec (16-bit PCM encode/decode, synthetic chime generation)
- Intent Matcher (exact match, aliases, slot extraction, distractor rejection, negation detection)
- State Machine (all voice lifecycle state transitions)
- Server Integration (REST endpoints `/api/config`, `/api/catalog`, `/api/voices`, `/api/tts`, and full WebSocket handshake & action handling)
- Synthetic Round-Trip Acoustic Evaluation (Tier-1 smoke suite with Kokoro TTS audio, Nemotron-3.5 streaming, Parakeet Ultra 2nd pass, Intent Router, and noise gating)

---

## IPC Protocol Specification

The VoxLab host daemon communicates with web clients over WebSocket (`ws://localhost:8080/ws`).

### Client $\to$ Server Actions (JSON)
- `{"action": "activate", "mode": "command"}`: Enter command listening mode.
- `{"action": "activate", "mode": "dictation", "target_id": "scope_1"}`: Enter dictation mode.
- `{"action": "cancel"}`: Reset state to `IDLE`.
- `{"action": "confirm", "approved": true}`: Confirm pending action.
- `{"action": "speak", "payload": {"text": "Hello", "voice": "af_heart", "speed": 1.0}}`: Request Kokoro TTS. The voice prefix sets the language (`ef_`/`em_` Spanish, `ff_` French, `if_`/`im_` Italian, `pf_`/`pm_` Portuguese, `hf_`/`hm_` Hindi). At most 4 run at once; more get `tts.error`.
- `{"action": "set_source", "source": "web_ui_mic" | "host_native_mic" | "wav_file_injection"}`: Switch audio source.
- `{"action": "set_asr_model", "model": "sherpa-onnx-nemotron-3.5-asr-streaming-0.6b-560ms-int8-2026-06-11" | "sherpa-onnx-streaming-zipformer-en-20M-2023-02-17"}`: Switch streaming ASR model dynamically.
- `{"action": "set_second_pass", "mode": "command" | "dictation", "enabled": true | false}`: Toggle offline second-pass refinement per mode.
- `{"action": "inject_text", "text": "open settings", "mode": "command"}`: Inject text for zero-mic testing.
- Binary Frames: 16kHz Float32 or Int16 mono PCM audio frames from the browser microphone. Messages over 1 MiB close the connection (1009); HTTP bodies have the same limit.

### Server $\to$ Client Events (JSON)
- `voice.state`: Emitted on state transitions (`IDLE`, `COMMAND_LISTEN`, `PROCESS`, `CONFIRM`, `ANNOTATION_LISTEN`, `REVIEW`, `SPEAKING`).
- `audio.meter`: Real-time audio metrics (`rms`, `dbfs`, `passed_gate`, `echo_muted`).
- `wake_word_detected`: Wake-word detected (`phrase`, `confidence`).
- `transcript.partial`: Streaming hypothesis (`transcript`, `tokens`).
- `transcript.final`: Finalized ASR transcript.
- `intent_matched`: Approved intent (`intent_id`, `confidence`, `margin`, `slots`).
- `utterance_rejected`: Quiet rejection (`reason`, `raw_transcript`, `confidence`).
- `annotation.draft`: Finalized dictation text (`draft_text`, `word_count`, `target_id`).
- `tts.started` / `tts.finished`: TTS generation notifications with base64 WAV payload and latency metrics.
- `tts.error`: Synthesis failed or was refused (`text`, `message`).
- `transcript.refined`: Second-pass result (`mode`, `first_pass`, `transcript`, `latency_ms`).
- `eval.started` / `eval.item_started` / `eval.progress` / `eval.completed` / `eval.cancelled`: Benchmark run lifecycle.
- `eval.error`: A run could not start (`message`, `running`: whether another run is in progress).
- `telemetry.benchmark`: Latency breakdown (`intent_latency_ms`, `vad_latency_ms`, `tts_ttfa_ms`).

---

## License
MIT
