# VoxLab Neural Speech Models Reference (TTS & ASR)

This document details the neural speech synthesis (TTS) and speech recognition (ASR) models supported by **VoxLab**, model precision variants (FP32, INT8, FP16), in-memory warm runtime architecture, dynamic model discovery, and ONNX metadata utilities.

---

## 1. Directory Structure & Installed Models

Speech models are stored in the local `models/` directory (excluded from git tracking due to file size):

```
models/
├── kokoro-multi-lang-v1_1/                    # Kokoro v1.1 Multi-Language (103 voices, FP32 & INT8)
│   ├── model.onnx                             # 32-bit floating point weights (~325 MB)
│   ├── model.int8.onnx                        # 8-bit quantized weights (~114 MB, shares lexicons)
│   ├── voices.bin                             # Speaker style embeddings (103 profiles)
│   ├── tokens.txt, espeak-ng-data/, dict/     # G2P & phonemizer dictionaries
│   └── lexicon-*.txt, *.fst                   # Multi-language lexical & number rules
│
├── kokoro-multi-lang-v1_0/                    # Kokoro v1.0 Multi-Language (54 voices, FP32 & FP16)
│   ├── model.onnx                             # Standard FP32 model (~325 MB)
│   ├── model.fp16.onnx                        # Half-precision FP16 model (~163 MB)
│   └── voices.bin, tokens.txt, ...
│
├── kokoro-en-v0_19/                           # Legacy Kokoro v0.19 English (11 voices)
│   ├── model.onnx                             # Standard weights (~345 MB)
│   └── voices.bin, tokens.txt, espeak-ng-data/
│
└── sherpa-onnx-streaming-zipformer-en-2023-06-26/  # Real-Time Streaming ASR
    ├── encoder-*.onnx / .int8.onnx            # Streaming acoustic encoder
    ├── decoder-*.onnx / .int8.onnx            # Autoregressive predictor
    ├── joiner-*.onnx / .int8.onnx             # Transducer joint network
    └── tokens.txt, bpe.model                  # BPE subword vocabulary
```

*(Note: If a legacy standalone `models/kokoro-int8-multi-lang-v1_1/` directory is present, VoxLab also discovers it for full backward compatibility).*

---

## 2. Supported Neural Models

VoxLab leverages [Sherpa-ONNX](https://github.com/k2-fsa/sherpa-onnx) for local, low-latency, offline neural speech processing.

### Kokoro TTS (Text-to-Speech)
[Kokoro](https://github.com/hexgrad/Kokoro-82M) is a lightweight multi-lingual, multi-speaker neural speech synthesizer ($82\text{M}$ parameters) producing natural speech at 24 kHz.

| Model Folder | Model Identifier | Precision | Weights File | Speaker Voices | Disk Footprint |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `models/kokoro-multi-lang-v1_1` | `kokoro-multi-lang-v1_1` | **FP32** | `model.onnx` | 103 voices | ~325 MB |
| `models/kokoro-multi-lang-v1_1` | `kokoro-multi-lang-v1_1 (INT8)` | **INT8** | `model.int8.onnx` | 103 voices | **~114 MB** |
| `models/kokoro-multi-lang-v1_0` | `kokoro-multi-lang-v1_0` | **FP32** | `model.onnx` | 54 voices | ~325 MB |
| `models/kokoro-multi-lang-v1_0` | `kokoro-multi-lang-v1_0 (FP16)` | **FP16** | `model.fp16.onnx` | 54 voices | **~163 MB** |
| `models/kokoro-en-v0_19` | `kokoro-en-v0_19` | **FP32** | `model.onnx` | 11 voices | ~345 MB |

### Zipformer ASR (Automatic Speech Recognition)
- **Model Path**: `models/sherpa-onnx-streaming-zipformer-en-2023-06-26`
- **Architecture**: Streaming transducer with chunk size $16$ (30 ms per chunk), left context $128$.
- **Components**:
  - Acoustic Encoder: `encoder-epoch-99-avg-1-chunk-16-left-128.int8.onnx` (or `.onnx`)
  - Decoder Predictor: `decoder-epoch-99-avg-1-chunk-16-left-128.int8.onnx` (or `.onnx`)
  - Joiner Transducer: `joiner-epoch-99-avg-1-chunk-16-left-128.int8.onnx` (or `.onnx`)
  - Subword Token Vocabulary: `tokens.txt` and `bpe.model`

---

## 3. Quantization & Precision Variants (FP16 & INT8)

VoxLab supports co-locating multiple weight variants inside a single Kokoro directory, sharing lexicons, dictionaries, and `voices.bin`:

### FP16 Half-Precision Model (`model.fp16.onnx`)
- **50% Smaller Footprint**: ~163 MB compared to ~325 MB for FP32.
- **Sub-Second Latency**: Generates speech at **~0.9s** on warm in-memory engines (RTF ~0.37).
- **Lower Memory Usage**: Halves RAM usage for model tensor loading.
- **Preserved Speech Quality**: Retains full vocal timbre, prosody, and speaker characteristics across all voices.
- **Automatic Discovery**: Discovered and surfaced with the `(FP16)` suffix (e.g. `kokoro-multi-lang-v1_0 (FP16)`).

### INT8 Quantized Model (`model.int8.onnx`)
- **~3x Smaller Weights**: ~114 MB compared to ~325 MB for FP32.
- **x86 vs ARM Note**: While INT8 provides speedups on mobile ARM hardware with integer vector extensions, the ONNX Runtime CPU execution provider (MLAS) on x86-64 CPUs experiences dynamic quantization overhead on Kokoro's AdaIN layers. On modern x86 CPUs, **FP16 and FP32 run ~4–5x faster than INT8**.
- **Co-located Assets**: Lives inside `models/kokoro-multi-lang-v1_1/` sharing all 103 voice profiles and multilingual lexicons without duplicating ~60 MB of files.
- **Automatic Discovery**: Discovered and surfaced with the `(INT8)` suffix (e.g. `kokoro-multi-lang-v1_1 (INT8)`).

---

## 4. Persistent In-Memory Engine & Hardware Optimization

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

## 5. Dynamic Model Switching Architecture

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
     "model": "kokoro-multi-lang-v1_0 (FP16)",
     "is_neural": true,
     "installed_models": [
       "kokoro-multi-lang-v1_1",
       "kokoro-multi-lang-v1_1 (INT8)",
       "kokoro-multi-lang-v1_0",
       "kokoro-multi-lang-v1_0 (FP16)",
       "kokoro-en-v0_19"
     ],
     "voices": [ ... 54 profiles ... ]
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

## 6. Sherpa-ONNX Metadata Requirements

The native `sherpa-onnx-offline-tts` binary parses ONNX `metadata_props` directly from the model graph protobuf at initialization. Checkpoints downloaded directly from general HuggingFace repositories (such as raw exports from `hexgrad/Kokoro-82M`) often lack these properties, causing Sherpa-ONNX to fail:

```text
offline-tts-kokoro-model.cc:Init:149 'sample_rate' does not exist in the metadata
```

### Required Metadata Properties
For a Kokoro ONNX model to run under Sherpa-ONNX, the following metadata keys must be present:
- `sample_rate`: Sample rate in Hz (e.g. `24000`)
- `model_type`: Must be `kokoro`
- `version`: Version number (e.g. `2` for v1.0 and v1.1)
- `has_espeak`: `1` (indicates eSpeak-ng phoneme preprocessing)
- `style_dim`: Embedding dimension, e.g. `510,1,256`
- `n_speakers`: Total speaker count (e.g. `54` or `103`)
- `id2speaker` / `speaker2id` / `speaker_names`: Comma-separated speaker IDs and aliases

---

## 7. Metadata Helper Script (`patch_kokoro_fp16_metadata.py`)

VoxLab includes a dedicated utility script [`scripts/patch_kokoro_fp16_metadata.py`](file:///C:/Alex/voxlab/scripts/patch_kokoro_fp16_metadata.py) to inspect and inject missing Sherpa-ONNX metadata properties from an existing reference model into a newly downloaded ONNX file.

### Prerequisites
```bash
pip install onnx
```

### Syntax
```text
usage: patch_kokoro_fp16_metadata.py [-h] [--check] [target] [ref]

positional arguments:
  target      Path to target model (default: models/kokoro-multi-lang-v1_0/model.fp16.onnx)
  ref         Path to reference model with valid metadata (default: models/kokoro-multi-lang-v1_0/model.onnx)

options:
  -h, --help  show this help message and exit
  --check     Inspect and print metadata of the target model without modifying it
```

### Examples
1. **Inspect Model Metadata**:
   ```bash
   python scripts/patch_kokoro_fp16_metadata.py --check models/kokoro-multi-lang-v1_0/model.fp16.onnx
   ```
2. **Patch Metadata from Reference**:
   ```bash
   python scripts/patch_kokoro_fp16_metadata.py models/kokoro-multi-lang-v1_0/model.fp16.onnx models/kokoro-multi-lang-v1_0/model.onnx
   ```

---

## 8. Downloading Official Models

VoxLab provides download scripts to retrieve pre-packaged Sherpa-ONNX models:

### On Windows (PowerShell)
```powershell
# Automatically download default Kokoro and Zipformer models
.\scripts\download_models.ps1

# Or specify a particular Kokoro version:
.\scripts\download_models.ps1 -KokoroVersion v1_1   # 103 voices, full FP32
.\scripts\download_models.ps1 -KokoroVersion int8   # 103 voices, compact INT8 (~114 MB)
.\scripts\download_models.ps1 -KokoroVersion v1_0   # 54 voices
```

### On Linux (Bash)
```bash
chmod +x scripts/download_models.sh
./scripts/download_models.sh v1_1     # Options: v1_1, int8, v1_0, v0_19
```
