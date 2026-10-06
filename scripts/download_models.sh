#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODELS_DIR="$SCRIPT_DIR/../models"
mkdir -p "$MODELS_DIR"

echo "=================================================="
echo " VoxLab Model Downloader (Linux)"
echo " Target directory: $MODELS_DIR"
echo "=================================================="

# 1. Kokoro TTS Model
KOKORO_FOLDER="$MODELS_DIR/kokoro-en-v0_19"
if [ ! -d "$KOKORO_FOLDER" ]; then
    echo "[1/2] Downloading Kokoro TTS Model..."
    wget -c "https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kokoro-en-v0_19.tar.bz2" -O "$MODELS_DIR/kokoro-en-v0_19.tar.bz2"
    tar -xjf "$MODELS_DIR/kokoro-en-v0_19.tar.bz2" -C "$MODELS_DIR"
    rm -f "$MODELS_DIR/kokoro-en-v0_19.tar.bz2"
    echo "Kokoro TTS model installed!"
else
    echo "[1/2] Kokoro TTS Model already present."
fi

# 2. Streaming Zipformer ASR Model
ASR_FOLDER="$MODELS_DIR/sherpa-onnx-streaming-zipformer-en-2023-06-26"
if [ ! -d "$ASR_FOLDER" ]; then
    echo "[2/2] Downloading Streaming Zipformer ASR Model..."
    wget -c "https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/sherpa-onnx-streaming-zipformer-en-2023-06-26.tar.bz2" -O "$MODELS_DIR/sherpa-onnx-streaming-zipformer-en-2023-06-26.tar.bz2"
    tar -xjf "$MODELS_DIR/sherpa-onnx-streaming-zipformer-en-2023-06-26.tar.bz2" -C "$MODELS_DIR"
    rm -f "$MODELS_DIR/sherpa-onnx-streaming-zipformer-en-2023-06-26.tar.bz2"
    echo "Streaming Zipformer model installed!"
else
    echo "[2/2] Streaming Zipformer ASR Model already present."
fi

echo "=================================================="
echo " Models ready for VoxLab Sherpa-ONNX engine!"
echo "=================================================="
