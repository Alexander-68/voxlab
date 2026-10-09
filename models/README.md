# VoxLab Neural Models & Variants

This directory contains neural speech models loaded and executed by the **VoxLab** host runtime via [Sherpa-ONNX](https://github.com/k2-fsa/sherpa-onnx).

---

## 1. Directory Structure & Installed Models

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

*(Note: If a legacy standalone `kokoro-int8-multi-lang-v1_1/` directory is present, VoxLab also discovers it for full backward compatibility).*

---

## 2. Model Variants & Precision

| Model Identifier | Folder | Precision | Weights File | Speaker Voices | Size |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `kokoro-multi-lang-v1_1` | `kokoro-multi-lang-v1_1` | FP32 | `model.onnx` | 103 voices | ~325 MB |
| `kokoro-multi-lang-v1_1 (INT8)` | `kokoro-multi-lang-v1_1` | **INT8** | `model.int8.onnx` | 103 voices | ~114 MB |
| `kokoro-multi-lang-v1_0` | `kokoro-multi-lang-v1_0` | FP32 | `model.onnx` | 54 voices | ~325 MB |
| `kokoro-multi-lang-v1_0 (FP16)` | `kokoro-multi-lang-v1_0` | **FP16** | `model.fp16.onnx` | 54 voices | ~163 MB |
| `kokoro-en-v0_19` | `kokoro-en-v0_19` | FP32 | `model.onnx` | 11 voices | ~345 MB |

### Precision Variants & Co-location
- **INT8 Quantized Model (`model.int8.onnx`)**: The 8-bit quantized weights cut model footprint nearly 3x (~114 MB vs ~325 MB) and increase CPU inference throughput. Placing `model.int8.onnx` directly inside `kokoro-multi-lang-v1_1/` allows it to share identical lexicons, phonemizer data, and `voices.bin` without redundant disk usage. VoxLab surfaces it as `kokoro-multi-lang-v1_1 (INT8)`.
- **FP16 Half-Precision Model (`model.fp16.onnx`)**: Cuts memory and disk footprint in half (~163 MB vs ~325 MB) while preserving vocal quality and speaker timbre. When present, VoxLab surfaces it as an extra selectable model with the `(FP16)` suffix (e.g. `kokoro-multi-lang-v1_0 (FP16)`).

---

## 3. Dynamic Model Discovery & Switching

VoxLab automatically discovers all installed Kokoro models on disk:
- **Web UI**: The **Model** selector dropdown dynamically enumerates all installed models with speaker counts and precision tags (`(103 voices, Multi-lang)`, `(INT8)`, `(54 voices, Multi-lang)`, `(FP16)`). Selecting an option switches the active engine and refreshes the voice list without restarting the server.
- **REST API**:
  - `GET /api/voices?model=kokoro-multi-lang-v1_1 (INT8)` returns the 103 speaker profiles corresponding to the model.
  - `POST /api/models` with body `{"model": "kokoro-multi-lang-v1_1 (INT8)"}` activates the model.
- **WebSocket**: Send `{"action": "set_tts_model", "model": "kokoro-multi-lang-v1_1 (INT8)"}` to switch models dynamically during a live session.

---

## 4. Sherpa-ONNX Metadata Requirements

The native `sherpa-onnx-offline-tts` binary parses ONNX `metadata_props` at initialization. Checkpoints downloaded directly from general HuggingFace repositories (such as raw exports from `hexgrad/Kokoro-82M`) often lack these properties, causing Sherpa-ONNX to fail with errors such as:

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

## 5. Metadata Helper Script (`patch_kokoro_fp16_metadata.py`)

VoxLab includes a dedicated utility script [`scripts/patch_kokoro_fp16_metadata.py`](file:///C:/Alex/voxlab/scripts/patch_kokoro_fp16_metadata.py) to inspect and inject missing Sherpa-ONNX metadata properties from an existing reference model into a newly downloaded ONNX file.

### Prerequisites
```bash
pip install onnx
```

### Inspecting Model Metadata
Check whether an ONNX file has the necessary Sherpa-ONNX metadata properties:
```bash
python scripts/patch_kokoro_fp16_metadata.py --check models/kokoro-multi-lang-v1_0/model.fp16.onnx
```

### Patching Missing Metadata
To copy metadata from `model.onnx` into `model.fp16.onnx`:
```bash
python scripts/patch_kokoro_fp16_metadata.py models/kokoro-multi-lang-v1_0/model.fp16.onnx models/kokoro-multi-lang-v1_0/model.onnx
```
*(Positional arguments default to `models/kokoro-multi-lang-v1_0/model.fp16.onnx` and `models/kokoro-multi-lang-v1_0/model.onnx` if omitted).*

---

## 6. Downloading Official Models

To download official pre-packaged Sherpa-ONNX Kokoro and Zipformer models:

### On Windows
```powershell
# Automatically download default Kokoro and Zipformer models
.\scripts\download_models.ps1

# Or specify a particular Kokoro version:
.\scripts\download_models.ps1 -KokoroVersion v1_1   # 103 voices, full FP32
.\scripts\download_models.ps1 -KokoroVersion int8   # 103 voices, compact INT8 (86 MB)
.\scripts\download_models.ps1 -KokoroVersion v1_0   # 54 voices
```

### On Linux
```bash
chmod +x scripts/download_models.sh
./scripts/download_models.sh v1_1     # Options: v1_1, int8, v1_0, v0_19
```
