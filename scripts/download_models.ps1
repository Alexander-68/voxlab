# PowerShell script to download official Sherpa-ONNX tools and models for VoxLab
param (
    [string]$KokoroVersion = "v1_1" # Options: "v1_1" (103 speakers), "v1_0" (53 speakers), "v0_19" (11 speakers)
)

$ErrorActionPreference = "Stop"

$RootDir = Join-Path $PSScriptRoot ".."
$ModelsDir = Join-Path $RootDir "models"
$BinDir = Join-Path $RootDir "bin"

if (-not (Test-Path $ModelsDir)) { New-Item -ItemType Directory -Path $ModelsDir | Out-Null }
if (-not (Test-Path $BinDir)) { New-Item -ItemType Directory -Path $BinDir | Out-Null }

Write-Host "=================================================="
Write-Host " VoxLab Neural Model & Engine Downloader (Windows)"
Write-Host " Target directory: $ModelsDir"
Write-Host "=================================================="

# 1. Precompiled Sherpa-ONNX Executable Tools
$SherpaBinArchive = Join-Path $ModelsDir "sherpa-onnx-bin.tar.bz2"
$SherpaTtsExe = Join-Path $BinDir "sherpa-onnx-offline-tts.exe"

if (-not (Test-Path $SherpaTtsExe)) {
    Write-Host "[1/3] Downloading Sherpa-ONNX pre-compiled binaries (v1.13.8 x64)..."
    $SherpaBinUrl = "https://github.com/k2-fsa/sherpa-onnx/releases/download/v1.13.8/sherpa-onnx-v1.13.8-win-x64-static-MT-Release.tar.bz2"
    Invoke-WebRequest -Uri $SherpaBinUrl -OutFile $SherpaBinArchive
    Write-Host "Extracting binaries into $BinDir..."
    tar -xjf $SherpaBinArchive -C $BinDir --strip-components=2 "*/bin/*"
    Remove-Item $SherpaBinArchive -Force
    Write-Host "Sherpa-ONNX binaries installed successfully!" -ForegroundColor Green
} else {
    Write-Host "[1/3] Sherpa-ONNX binaries already installed in bin/." -ForegroundColor Yellow
}

# 2. Kokoro TTS Model Selection
$KokoroName = "kokoro-multi-lang-v1_1"
if ($KokoroVersion -eq "v1_0") {
    $KokoroName = "kokoro-multi-lang-v1_0"
} elseif ($KokoroVersion -eq "v0_19") {
    $KokoroName = "kokoro-en-v0_19"
}

$KokoroFolder = Join-Path $ModelsDir $KokoroName
$KokoroLegacyFolder = Join-Path $ModelsDir "kokoro-en-v0_19"

# If user already downloaded v0_19, check if they want to upgrade
if ((Test-Path $KokoroFolder) -or ((Test-Path $KokoroLegacyFolder) -and $KokoroVersion -eq "v0_19")) {
    Write-Host "[2/3] Kokoro TTS Model already present." -ForegroundColor Yellow
} else {
    Write-Host "[2/3] Downloading latest Kokoro TTS Model ($KokoroName)..."
    $KokoroUrl = "https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/$KokoroName.tar.bz2"
    $KokoroArchive = Join-Path $ModelsDir "$KokoroName.tar.bz2"
    Invoke-WebRequest -Uri $KokoroUrl -OutFile $KokoroArchive
    Write-Host "Extracting Kokoro archive..."
    tar -xjf $KokoroArchive -C $ModelsDir
    Remove-Item $KokoroArchive -Force
    Write-Host "Kokoro TTS ($KokoroName) installed successfully!" -ForegroundColor Green
}

# 3. Streaming Zipformer ASR Model
$AsrFolder = Join-Path $ModelsDir "sherpa-onnx-streaming-zipformer-en-2023-06-26"
$AsrArchive = Join-Path $ModelsDir "sherpa-onnx-streaming-zipformer-en-2023-06-26.tar.bz2"

if (-not (Test-Path $AsrFolder)) {
    Write-Host "[3/3] Downloading Streaming Zipformer ASR Model..."
    $AsrUrl = "https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/sherpa-onnx-streaming-zipformer-en-2023-06-26.tar.bz2"
    Invoke-WebRequest -Uri $AsrUrl -OutFile $AsrArchive
    Write-Host "Extracting ASR archive..."
    tar -xjf $AsrArchive -C $ModelsDir
    Remove-Item $AsrArchive -Force
    Write-Host "Streaming Zipformer model installed successfully!" -ForegroundColor Green
} else {
    Write-Host "[3/3] Streaming Zipformer ASR Model already present." -ForegroundColor Yellow
}

Write-Host "=================================================="
Write-Host " Neural models & Sherpa engine ready for VoxLab!" -ForegroundColor Green
Write-Host " Run with: go run ./cmd/voxlab -engine=sherpa"
Write-Host "=================================================="
