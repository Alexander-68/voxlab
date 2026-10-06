# PowerShell script to download official pre-trained Sherpa-ONNX models for VoxLab
$ErrorActionPreference = "Stop"

$ModelsDir = Join-Path $PSScriptRoot "..\models"
if (-not (Test-Path $ModelsDir)) {
    New-Item -ItemType Directory -Path $ModelsDir | Out-Null
}

Write-Host "=================================================="
Write-Host " VoxLab Model Downloader (Windows)"
Write-Host " Target directory: $ModelsDir"
Write-Host "=================================================="

# 1. Kokoro TTS Model
$KokoroUrl = "https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kokoro-en-v0_19.tar.bz2"
$KokoroArchive = Join-Path $ModelsDir "kokoro-en-v0_19.tar.bz2"
$KokoroFolder = Join-Path $ModelsDir "kokoro-en-v0_19"

if (-not (Test-Path $KokoroFolder)) {
    Write-Host "[1/2] Downloading Kokoro TTS Model (kokoro-en-v0_19)..."
    Invoke-WebRequest -Uri $KokoroUrl -OutFile $KokoroArchive
    Write-Host "Extracting Kokoro archive..."
    tar -xjf $KokoroArchive -C $ModelsDir
    Remove-Item $KokoroArchive -Force
    Write-Host "Kokoro TTS model installed successfully!" -ForegroundColor Green
} else {
    Write-Host "[1/2] Kokoro TTS Model already present." -ForegroundColor Yellow
}

# 2. Streaming Zipformer ASR Model
$AsrUrl = "https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/sherpa-onnx-streaming-zipformer-en-2023-06-26.tar.bz2"
$AsrArchive = Join-Path $ModelsDir "sherpa-onnx-streaming-zipformer-en-2023-06-26.tar.bz2"
$AsrFolder = Join-Path $ModelsDir "sherpa-onnx-streaming-zipformer-en-2023-06-26"

if (-not (Test-Path $AsrFolder)) {
    Write-Host "[2/2] Downloading Streaming Zipformer ASR Model..."
    Invoke-WebRequest -Uri $AsrUrl -OutFile $AsrArchive
    Write-Host "Extracting ASR archive..."
    tar -xjf $AsrArchive -C $ModelsDir
    Remove-Item $AsrArchive -Force
    Write-Host "Streaming Zipformer model installed successfully!" -ForegroundColor Green
} else {
    Write-Host "[2/2] Streaming Zipformer ASR Model already present." -ForegroundColor Yellow
}

Write-Host "=================================================="
Write-Host " Models ready for VoxLab Sherpa-ONNX engine!" -ForegroundColor Green
Write-Host "=================================================="
