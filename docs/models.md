# VoxLab Speech Models & Utilities Reference

This guide details the text-to-speech (TTS) and automatic speech recognition (ASR) neural models supported by **VoxLab**, model precision variants (FP32, INT8, FP16), dynamic model discovery, and the ONNX metadata patching helper script.

---

## 1. Supported Neural Models

VoxLab leverages [Sherpa-ONNX](https://github.com/k2-fsa/sherpa-onnx) for local, low-latency, offline neural speech processing.

### Kokoro TTS (Text-to-Speech)
[Kokoro](https://github.com/hexgrad/Kokoro-82M) is a lightweight multi-lingual, multi-speaker neural speech synthesizer ($82\text{M}$ parameters) producing natural speech at 24kHz.

VoxLab supports multiple generations and quantization variants:

| Model Folder | Model Identifier | Precision | Weights File | Speaker Voices | Disk Footprint |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `models/kokoro-multi-lang-v1_1` | `kokoro-multi-lang-v1_1` | **FP32** | `model.onnx` | 103 voices | ~325 MB |
| `models/kokoro-multi-lang-v1_1` | `kokoro-multi-lang-v1_1 (INT8)` | **INT8** | `model.int8.onnx` | 103 voices | **~114 MB** |
| `models/kokoro-multi-lang-v1_0` | `kokoro-multi-lang-v1_0` | **FP32** | `model.onnx` | 54 voices | ~325 MB |
| `models/kokoro-multi-lang-v1_0` | `kokoro-multi-lang-v1_0 (FP16)` | **FP16** | `model.fp16.onnx` | 54 voices | **~163 MB** |
| `models/kokoro-en-v0_19` | `kokoro-en-v0_19` | **FP32** | `model.onnx` | 11 voices | ~345 MB |

*(Note: Standalone legacy directory `models/kokoro-int8-multi-lang-v1_1` is also supported for backwards compatibility).*

### Zipformer ASR (Automatic Speech Recognition)
- **Model Path**: `models/sherpa-onnx-streaming-zipformer-en-2023-06-26`
- **Architecture**: Streaming transducer with chunk size $16$ (30ms per chunk), left context $128$.
- **Components**:
  - Acoustic Encoder: `encoder-epoch-99-avg-1-chunk-16-left-128.int8.onnx` (or `.onnx`)
  - Decoder Predictor: `decoder-epoch-99-avg-1-chunk-16-left-128.int8.onnx` (or `.onnx`)
  - Joiner Transducer: `joiner-epoch-99-avg-1-chunk-16-left-128.int8.onnx` (or `.onnx`)
  - Subword Token Vocabulary: `tokens.txt` and `bpe.model`

---

## 2. Quantization & Precision Variants (INT8 & FP16)

VoxLab supports co-locating multiple weight variants inside a single Kokoro directory, sharing lexicons, dictionaries, and `voices.bin`:

### INT8 Quantized Model (`model.int8.onnx`)
- **~3x Smaller Weights**: ~114 MB compared to ~325 MB for FP32.
- **x86 vs ARM Note**: While INT8 provides speedups on mobile ARM hardware with integer vector extensions, ONNX Runtime CPU execution provider (MLAS) on x86-64 CPUs experiences dynamic quantization overhead on Kokoro's AdaIN layers. On modern x86 CPUs, **FP16 and FP32 run ~4–5x faster than INT8**.
- **Co-located Assets**: Lives inside `models/kokoro-multi-lang-v1_1/` sharing all 103 voice profiles and multilingual lexicons without duplicating ~60 MB of files.
- **Automatic Discovery**: Discovered and surfaced with the `(INT8)` suffix (e.g. `kokoro-multi-lang-v1_1 (INT8)`).

### FP16 Half-Precision Model (`model.fp16.onnx`)
- **50% Smaller Footprint**: ~163 MB compared to ~325 MB for FP32.
- **Sub-Second Latency**: Generates speech at ~0.9s on warm in-memory engines (RTF ~0.37).
- **Lower Memory Usage**: Halves RAM usage for model tensor loading.
- **Preserved Speech Quality**: Retains full vocal timbre, prosody, and speaker characteristics across all 54 voices.
- **Automatic Discovery**: Discovered and surfaced with the `(FP16)` suffix (e.g. `kokoro-multi-lang-v1_0 (FP16)`).

---

## 3. Persistent In-Memory Engine & Hardware Optimization

To achieve real-time speech generation, VoxLab uses a two-tier execution pipeline:

### 1. In-Process Warm TTS Engine (`WarmTTS`)
- **Zero Process Cold-Start**: Instead of launching a separate `.exe` process per utterance (which incurs ~1.8–2.0s reading weights and parsing protobuf graphs off disk), VoxLab binds directly to `sherpa-onnx-c-api.dll` and `onnxruntime.dll` via dynamic linking.
- **Resident Model in RAM**: Model weights, style vectors (`voices.bin`), and dictionaries remain cached in RAM.
- **Sub-Second Response**: Speech synthesis latency drops from ~4.3s to **~0.9–1.2s** (Time-to-Audio < 1.0s).
- **Zero Disk I/O**: Synthesized floating-point samples are encoded to WAV directly in memory.

### 2. Multi-Threading & Hardware Execution Providers
- **Multi-Threading (`num_threads`)**: Configurable via `config.json` or CLI (`-threads=4`). Defaults to 4 threads (matching physical CPU cores), delivering a >2x speedup over single-threaded defaults.
- **Execution Providers (`provider`)**: Configurable via `config.json` or CLI (`-provider=cpu`, `-provider=directml`, `-provider=cuda`).

---

## 4. Dynamic Model Switching Architecture

VoxLab supports hot-swapping active models at runtime without service restarts:

```
┌────────────────────────────────────────────────────────┐
│                        Web UI                          │
│     Dropdown Option: "kokoro-multi-lang-v1_0 (FP16)"   │
└───────────────────────────┬────────────────────────────┘
                            │ WS: action="set_tts_model"
                            │ REST: POST /api/models
                            ▼
┌────────────────────────────────────────────────────────┐
│                      Server API                        │
│             s.ttsMgr.SetModel(modelName)               │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│                  SherpaRunner Engine                   │
│  - Sets KokoroModelDir = "models/kokoro-multi-lang-v1_0"│
│  - Sets KokoroModelFile = "model.fp16.onnx"             │
│  - Detects version ("v1_0") -> maps 54 speaker profiles│
└──────────────┬──────────────────────────┬──────────────┘
               │ (Primary)                │ (Fallback)
               ▼                          ▼
┌──────────────────────────────┐ ┌──────────────────────┐
│ In-Process Warm Engine (DLL) │ │ CLI Process Runner   │
│ - Zero disk I/O              │ │ - --num-threads=4    │
│ - Sub-second latency (<1.0s) │ │ - --provider=cpu     │
└──────────────────────────────┘ └──────────────────────┘
```

### API Endpoints
1. **Query Available Voices & Models**:
   ```http
   GET /api/voices?model=kokoro-multi-lang-v1_0 (FP16)
   ```
   Returns:
   ```json
   {
     "model": "kokoro-multi-lang-v1_1 (INT8)",
     "is_neural": true,
     "installed_models": [
       "kokoro-multi-lang-v1_1",
       "kokoro-multi-lang-v1_1 (INT8)",
       "kokoro-multi-lang-v1_0",
       "kokoro-multi-lang-v1_0 (FP16)",
       "kokoro-en-v0_19"
     ],
     "voices": [ ... 103 profiles ... ]
   }
   ```

2. **Switch Active Model**:
   ```http
   POST /api/models
   Content-Type: application/json

   {"model": "kokoro-multi-lang-v1_1 (INT8)"}
   ```

3. **WebSocket Action**:
   ```json
   {"action": "set_tts_model", "model": "kokoro-multi-lang-v1_0 (FP16)"}
   ```

---

## 4. ONNX Metadata Helper Script (`patch_kokoro_fp16_metadata.py`)

### Background
Sherpa-ONNX reads embedded model metadata properties (`metadata_props`) directly from the ONNX graph protobuf. Raw ONNX exports downloaded from external repositories (e.g. HuggingFace `hexgrad/Kokoro-82M`) generally lack these keys, resulting in:

```text
offline-tts-kokoro-model.cc:Init:149 'sample_rate' does not exist in the metadata
```

### Script Overview
[`scripts/patch_kokoro_fp16_metadata.py`](file:///C:/Alex/voxlab/scripts/patch_kokoro_fp16_metadata.py) inspects and copies all missing Sherpa-ONNX metadata keys from a working reference model (`model.onnx`) into the target model (`model.fp16.onnx`).

### Script Syntax
```text
usage: patch_kokoro_fp16_metadata.py [-h] [--check] [target] [ref]

Inject or verify Sherpa-ONNX metadata properties on Kokoro ONNX models.

positional arguments:
  target      Path to target model (default: models/kokoro-multi-lang-v1_0/model.fp16.onnx)
  ref         Path to reference model with valid metadata (default: models/kokoro-multi-lang-v1_0/model.onnx)

options:
  -h, --help  show this help message and exit
  --check     Inspect and print metadata of the target model without modifying it
```

### Examples

1. **Inspect Target Model Metadata**:
   ```bash
   python scripts/patch_kokoro_fp16_metadata.py --check models/kokoro-multi-lang-v1_0/model.fp16.onnx
   ```

2. **Patch Metadata using Default Paths**:
   ```bash
   python scripts/patch_kokoro_fp16_metadata.py
   ```

3. **Patch a Custom ONNX File**:
   ```bash
   python scripts/patch_kokoro_fp16_metadata.py models/custom-kokoro/model.fp16.onnx models/kokoro-multi-lang-v1_1/model.onnx
   ```
