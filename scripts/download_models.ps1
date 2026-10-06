# PowerShell script to download official Sherpa-ONNX tools and models for VoxLab
# Features:
# - Native curl.exe hardware acceleration with resumable download (-C -)
# - No PowerShell progress GUI freeze ($ProgressPreference = 'SilentlyContinue')
# - Support for Kokoro v1.1 (103 voices), quantized int8 (fast ~147MB), or v0.19
# - Automatic detection of already present models and tools

param (
    [string]$KokoroVersion = "auto", # Options: "auto", "int8" (recommended fast), "v1_1" (full 103 voices), "v0_19"
    [switch]$SkipBinaries = $false,
    [switch]$SkipKokoro = $false,
    [switch]$SkipAsr = $false
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$RootDir = Join-Path $PSScriptRoot ".."
$ModelsDir = Join-Path $RootDir "models"
$BinDir = Join-Path $RootDir "bin"

if (-not (Test-Path $ModelsDir)) { New-Item -ItemType Directory -Path $ModelsDir | Out-Null }
if (-not (Test-Path $BinDir)) { New-Item -ItemType Directory -Path $BinDir | Out-Null }

function Invoke-FastDownload {
    param(
        [string]$Url,
        [string]$OutFile,
        [string]$Description
    )
    Write-Host "-> Downloading $Description..." -ForegroundColor Cyan
    Write-Host "   URL: $Url" -ForegroundColor Gray

    if (Get-Command curl.exe -ErrorAction SilentlyContinue) {
        # curl.exe uses libcurl, unthrottled streaming buffers, and resumable downloads
        & curl.exe -L -C - --progress-bar --retry 3 --retry-delay 2 -o $OutFile $Url
        if ($LASTEXITCODE -ne 0) {
            throw "curl.exe download failed for $Url (exit code $LASTEXITCODE)"
        }
    } else {
        Invoke-WebRequest -Uri $Url -OutFile $OutFile
    }
}

Write-Host "=================================================="
Write-Host " VoxLab Accelerated Neural Model Downloader"
Write-Host " Target directory: $ModelsDir"
Write-Host "=================================================="

# 1. Precompiled Sherpa-ONNX Executable Tools
$SherpaTtsExe = Join-Path $BinDir "sherpa-onnx-offline-tts.exe"
$SherpaBinArchive = Join-Path $ModelsDir "sherpa-onnx-bin.tar.bz2"

if (-not $SkipBinaries) {
    if (-not (Test-Path $SherpaTtsExe)) {
        Write-Host "[1/3] Sherpa-ONNX pre-compiled binaries (v1.13.8 x64)..." -ForegroundColor White
        $SherpaBinUrl = "https://github.com/k2-fsa/sherpa-onnx/releases/download/v1.13.8/sherpa-onnx-v1.13.8-win-x64-static-MT-Release.tar.bz2"
        Invoke-FastDownload -Url $SherpaBinUrl -OutFile $SherpaBinArchive -Description "Sherpa-ONNX Windows Tools"
        Write-Host "Extracting binaries into $BinDir..." -ForegroundColor Yellow
        tar -xjf $SherpaBinArchive -C $BinDir --strip-components=2 "*/bin/*"
        Remove-Item $SherpaBinArchive -Force -ErrorAction SilentlyContinue
        Write-Host "[OK] Sherpa-ONNX binaries installed in bin/" -ForegroundColor Green
    } else {
        Write-Host "[1/3] Sherpa-ONNX binaries already installed in bin/." -ForegroundColor Yellow
    }
}

# 2. Kokoro TTS Model Selection
if (-not $SkipKokoro) {
    $HasV019 = Test-Path (Join-Path $ModelsDir "kokoro-en-v0_19\model.onnx")
    $HasV11  = (Test-Path (Join-Path $ModelsDir "kokoro-multi-lang-v1_1\model.onnx")) -or (Test-Path (Join-Path $ModelsDir "kokoro-int8-multi-lang-v1_1\model.int8.onnx"))

    if ($KokoroVersion -eq "auto") {
        if ($HasV11) {
            Write-Host "[2/3] Kokoro v1.1 model already installed." -ForegroundColor Yellow
        } elseif ($HasV019) {
            Write-Host "[2/3] Existing Kokoro v0.19 model found on disk (ready to use)." -ForegroundColor Yellow
            Write-Host "     (To upgrade to Kokoro v1.1 with 103 voices, run: .\scripts\download_models.ps1 -KokoroVersion v1_1)" -ForegroundColor Gray
            Write-Host "     (Or for compact high-speed 86MB int8: .\scripts\download_models.ps1 -KokoroVersion int8)" -ForegroundColor Gray
        } else {
            # Default to compact int8 v1.1 (103 voices, 4x smaller download, fast inference)
            $KokoroVersion = "int8"
        }
    }

    if ($KokoroVersion -ne "auto" -and -not ($KokoroVersion -eq "v0_19" -and $HasV019)) {
        $KokoroArchiveName = "kokoro-int8-multi-lang-v1_1"
        $KokoroTargetDir = Join-Path $ModelsDir "kokoro-int8-multi-lang-v1_1"
        if ($KokoroVersion -eq "v1_1") {
            $KokoroArchiveName = "kokoro-multi-lang-v1_1"
            $KokoroTargetDir = Join-Path $ModelsDir "kokoro-multi-lang-v1_1"
        } elseif ($KokoroVersion -eq "v1_0") {
            $KokoroArchiveName = "kokoro-multi-lang-v1_0"
            $KokoroTargetDir = Join-Path $ModelsDir "kokoro-multi-lang-v1_0"
        }

        if (-not (Test-Path $KokoroTargetDir)) {
            Write-Host "[2/3] Downloading Kokoro TTS Model ($KokoroArchiveName)..." -ForegroundColor White
            $KokoroUrl = "https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/$KokoroArchiveName.tar.bz2"
            $KokoroArchive = Join-Path $ModelsDir "$KokoroArchiveName.tar.bz2"
            Invoke-FastDownload -Url $KokoroUrl -OutFile $KokoroArchive -Description "Kokoro Model ($KokoroArchiveName)"
            Write-Host "Extracting $KokoroArchiveName..." -ForegroundColor Yellow
            tar -xjf $KokoroArchive -C $ModelsDir
            Remove-Item $KokoroArchive -Force -ErrorAction SilentlyContinue
            Write-Host "[OK] Kokoro TTS ($KokoroArchiveName) installed!" -ForegroundColor Green
        } else {
            Write-Host "[2/3] Kokoro model ($KokoroArchiveName) already installed." -ForegroundColor Yellow
        }
    }
}

# 3. Streaming Zipformer ASR Model
$AsrFolder = Join-Path $ModelsDir "sherpa-onnx-streaming-zipformer-en-2023-06-26"
$AsrArchive = Join-Path $ModelsDir "sherpa-onnx-streaming-zipformer-en-2023-06-26.tar.bz2"

if (-not $SkipAsr) {
    if (-not (Test-Path $AsrFolder)) {
        Write-Host "[3/3] Downloading Streaming Zipformer ASR Model..." -ForegroundColor White
        $AsrUrl = "https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/sherpa-onnx-streaming-zipformer-en-2023-06-26.tar.bz2"
        Invoke-FastDownload -Url $AsrUrl -OutFile $AsrArchive -Description "Streaming Zipformer ASR"
        Write-Host "Extracting ASR archive..." -ForegroundColor Yellow
        tar -xjf $AsrArchive -C $ModelsDir
        Remove-Item $AsrArchive -Force -ErrorAction SilentlyContinue
        Write-Host "[OK] Streaming Zipformer model installed!" -ForegroundColor Green
    } else {
        Write-Host "[3/3] Streaming Zipformer ASR Model already present." -ForegroundColor Yellow
    }
}

Write-Host "=================================================="
Write-Host " Neural models & Sherpa engine ready for VoxLab!" -ForegroundColor Green
Write-Host " Run with: go run ./cmd/voxlab -engine=sherpa"
Write-Host "=================================================="
