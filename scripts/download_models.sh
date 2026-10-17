#!/usr/bin/env bash
set -e

KOKORO_VER="${1:-v1_1}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$SCRIPT_DIR/.."
MODELS_DIR="$ROOT_DIR/models"
BIN_DIR="$ROOT_DIR/bin"

mkdir -p "$MODELS_DIR"
mkdir -p "$BIN_DIR"

echo "=================================================="
echo " VoxLab Neural Model & Engine Downloader (Linux)"
echo " Target directory: $MODELS_DIR"
echo "=================================================="

# 1. Precompiled Sherpa-ONNX Executable Tools (Linux x64)
SHERPA_TTS_BIN="$BIN_DIR/sherpa-onnx-offline-tts"
if [ ! -f "$SHERPA_TTS_BIN" ]; then
    echo "[1/3] Downloading Sherpa-ONNX pre-compiled binaries (v1.13.8 Linux x64)..."
    SHERPA_URL="https://github.com/k2-fsa/sherpa-onnx/releases/download/v1.13.8/sherpa-onnx-v1.13.8-linux-x64-static.tar.bz2"
    wget -c "$SHERPA_URL" -O "$MODELS_DIR/sherpa-bin.tar.bz2"
    tar -xjf "$MODELS_DIR/sherpa-bin.tar.bz2" -C "$BIN_DIR" --strip-components=2 "*/bin/*" || true
    rm -f "$MODELS_DIR/sherpa-bin.tar.bz2"
    echo "Sherpa-ONNX binaries installed into $BIN_DIR!"
else
    echo "[1/3] Sherpa-ONNX binaries already installed in bin/."
fi

# 2. Kokoro TTS Model Selection
KOKORO_NAME="kokoro-multi-lang-v1_1"
if [ "$KOKORO_VER" == "int8" ]; then
    KOKORO_NAME="kokoro-int8-multi-lang-v1_1"
elif [ "$KOKORO_VER" == "v1_0" ]; then
    KOKORO_NAME="kokoro-multi-lang-v1_0"
elif [ "$KOKORO_VER" == "v0_19" ]; then
    KOKORO_NAME="kokoro-en-v0_19"
fi

KOKORO_FOLDER="$MODELS_DIR/$KOKORO_NAME"
ALREADY_INSTALLED=false
if [ -d "$KOKORO_FOLDER" ]; then
    ALREADY_INSTALLED=true
elif [ "$KOKORO_VER" == "int8" ] && [ -f "$MODELS_DIR/kokoro-multi-lang-v1_1/model.int8.onnx" ]; then
    ALREADY_INSTALLED=true
fi

if [ "$ALREADY_INSTALLED" = false ]; then
    echo "[2/3] Downloading latest Kokoro TTS Model ($KOKORO_NAME)..."
    KOKORO_URL="https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/$KOKORO_NAME.tar.bz2"
    wget -c "$KOKORO_URL" -O "$MODELS_DIR/$KOKORO_NAME.tar.bz2"
    tar -xjf "$MODELS_DIR/$KOKORO_NAME.tar.bz2" -C "$MODELS_DIR"
    rm -f "$MODELS_DIR/$KOKORO_NAME.tar.bz2"
    if [ "$KOKORO_VER" == "int8" ] && [ -d "$MODELS_DIR/kokoro-multi-lang-v1_1" ]; then
        cp "$KOKORO_FOLDER/model.int8.onnx" "$MODELS_DIR/kokoro-multi-lang-v1_1/model.int8.onnx"
        rm -rf "$KOKORO_FOLDER"
    fi
    echo "Kokoro TTS ($KOKORO_NAME) installed!"
else
    echo "[2/3] Kokoro TTS Model already present."
fi

# 3. ASR Models: Kroko streaming (first pass, default), Parakeet TDT (second pass),
#    LibriSpeech Zipformer (fallback, used by tests). Nemotron 560ms is optional (~460 MB, heavier on CPU).
for ASR_NAME in sherpa-onnx-streaming-zipformer-en-kroko-2025-08-06 sherpa-onnx-nemo-parakeet-tdt-0.6b-v2-int8 sherpa-onnx-streaming-zipformer-en-2023-06-26; do
    if [ ! -d "$MODELS_DIR/$ASR_NAME" ]; then
        echo "[3/3] Downloading ASR model $ASR_NAME..."
        wget -c "https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/$ASR_NAME.tar.bz2" -O "$MODELS_DIR/$ASR_NAME.tar.bz2"
        tar -xjf "$MODELS_DIR/$ASR_NAME.tar.bz2" -C "$MODELS_DIR"
        rm -f "$MODELS_DIR/$ASR_NAME.tar.bz2"
        echo "$ASR_NAME installed!"
    else
        echo "[3/3] $ASR_NAME already present."
    fi
done

echo "=================================================="
echo " Neural models & Sherpa engine ready for VoxLab!"
echo " Run with: go run ./cmd/voxlab -engine=sherpa"
echo "=================================================="
