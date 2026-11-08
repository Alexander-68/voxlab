<#
.SYNOPSIS
    Executes the VoxLab Tier-2 Deep Acoustic Matrix test suite manually with optional audio sample regeneration.

.DESCRIPTION
    Runs the 104-clip synthetic round-trip acoustic evaluation suite spanning clean speech, SNR ladders,
    acoustic perturbations (typing clacks, room babble, reverb), and multilingual dictations.
    Can run either the official Go test suite (`go test -run TestTier2DeepAcousticMatrix`) or the CLI evaluation
    bench harness (`go run ./cmd/eval_bench -tier 2`).

.PARAMETER ForceRegenerate
    Force fresh synthesis and perturbation of all Tier-2 audio clips. Existing WAVs are purged and
    regenerated via `go run ./cmd/gen_eval_samples -tier 2`.
    Aliases: -RegenerateAudio, -Regenerate, -Force.

.PARAMETER Mode
    Execution mode: 'test' runs `go test` with VOXLAB_TIER2=1 (default); 'bench' runs `cmd/eval_bench -tier 2`
    and generates the web dashboard JSON report at data/eval_report_tier-2-deep-matrix.json.

.PARAMETER Short
    Runs in short/quick evaluation mode (10-12 representative items instead of all 104) for quick turnaround.
    Aliases: -Quick.

.PARAMETER TimeoutMinutes
    Maximum execution timeout in minutes for go test (default: 40).

.PARAMETER IncludeMultilingual
    When running under 'test' mode, also runs the dedicated TestMultilingualDictations suite.

.PARAMETER RegenerateOnly
    Only regenerates the audio corpus and exits without running the test suite.

.EXAMPLE
    # Run standard Tier-2 test suite
    pwsh scripts/run_tier2_test.ps1

.EXAMPLE
    # Force fresh regeneration of all audio clips before running
    pwsh scripts/run_tier2_test.ps1 -ForceRegenerate

.EXAMPLE
    # Quick short-run with audio regeneration
    pwsh scripts/run_tier2_test.ps1 -RegenerateAudio -Short

.EXAMPLE
    # Run the CLI bench harness to update the JSON report for the web UI
    pwsh scripts/run_tier2_test.ps1 -Mode bench
#>

[CmdletBinding(DefaultParameterSetName = "Default")]
param(
    [Parameter(HelpMessage = "Force regeneration of all Tier-2 audio samples")]
    [Alias("RegenerateAudio", "Regenerate", "Force")]
    [switch]$ForceRegenerate,

    [Parameter(HelpMessage = "Execution harness: 'test' (go test suite) or 'bench' (CLI eval_bench)")]
    [ValidateSet("test", "bench")]
    [string]$Mode = "test",

    [Parameter(HelpMessage = "Run short representative slice for quick turnaround")]
    [Alias("Quick")]
    [switch]$Short,

    [Parameter(HelpMessage = "Maximum test timeout in minutes (default: 40)")]
    [int]$TimeoutMinutes = 40,

    [Parameter(HelpMessage = "Include dedicated multilingual dictation suite in go test")]
    [switch]$IncludeMultilingual,

    [Parameter(HelpMessage = "Only regenerate the audio samples, do not run the test")]
    [switch]$RegenerateOnly
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

# Resolve repository root
$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location -Path $RootDir

# Verify Go toolchain
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "Go executable ('go') not found in PATH. Please install Go v1.24+."
    exit 1
}

$Tier2Dir = Join-Path $RootDir "data\test_samples\eval_tier2"
$ManifestPath = Join-Path $Tier2Dir "manifest.json"

# Check if regeneration is needed or requested
$NeedsGeneration = $ForceRegenerate -or (-not (Test-Path $ManifestPath))

if ($ForceRegenerate) {
    Write-Host "==================================================" -ForegroundColor Cyan
    Write-Host " [VoxLab] Purging Tier-2 Audio Samples for Clean Regen" -ForegroundColor Cyan
    Write-Host " Target directory: $Tier2Dir" -ForegroundColor Gray
    Write-Host "==================================================" -ForegroundColor Cyan

    if (Test-Path $Tier2Dir) {
        $wavFiles = Get-ChildItem -Path $Tier2Dir -Filter "*.wav" -File
        if ($wavFiles) {
            Write-Host "Removing $($wavFiles.Count) existing WAV files..." -ForegroundColor Yellow
            $wavFiles | Remove-Item -Force
        }
        if (Test-Path $ManifestPath) {
            Remove-Item -Path $ManifestPath -Force
        }
    } else {
        New-Item -ItemType Directory -Path $Tier2Dir -Force | Out-Null
    }
} elseif (-not (Test-Path $ManifestPath)) {
    Write-Host "Notice: Tier-2 manifest missing at $ManifestPath. Generating audio samples..." -ForegroundColor Yellow
}

if ($NeedsGeneration) {
    Write-Host "`n>> Generating Tier-2 acoustic evaluation samples..." -ForegroundColor Green
    Write-Host "   Command: go run ./cmd/gen_eval_samples -tier 2" -ForegroundColor Gray
    
    & go run ./cmd/gen_eval_samples -tier 2
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed generating Tier-2 audio samples (exit code $LASTEXITCODE)."
        exit $LASTEXITCODE
    }
    Write-Host ">> Tier-2 audio samples successfully generated.`n" -ForegroundColor Green
}

if ($RegenerateOnly) {
    Write-Host "[Done] RegenerateOnly requested. Exiting without running tests." -ForegroundColor Green
    exit 0
}

# Verify required model assets
$SherpaModelDir = Join-Path $RootDir "models\sherpa-onnx-nemotron-3.5-asr-streaming-0.6b-560ms-int8-2026-06-11"
$KokoroDir = Join-Path $RootDir "models\kokoro-multi-lang-v1_0"
if ((-not (Test-Path $SherpaModelDir)) -or (-not (Test-Path $KokoroDir))) {
    Write-Warning "Required neural models were not found in 'models/'."
    Write-Warning "Run 'pwsh scripts/download_models.ps1' to fetch model assets."
}

# Execute Tier-2 Test / Bench
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host " Executing VoxLab Tier-2 Deep Acoustic Matrix" -ForegroundColor Cyan
Write-Host " Mode:    $Mode" -ForegroundColor White
Write-Host " Short:   $($Short.IsPresent)" -ForegroundColor White
Write-Host " Timeout: ${TimeoutMinutes}m" -ForegroundColor White
Write-Host "==================================================" -ForegroundColor Cyan

$exitCode = 0

if ($Mode -eq "bench") {
    $benchArgs = @("run", "./cmd/eval_bench", "-tier", "2")
    if ($Short) {
        $benchArgs += "-quick"
    }
    Write-Host "Running: go $($benchArgs -join ' ')`n" -ForegroundColor Gray
    & go @benchArgs
    $exitCode = $LASTEXITCODE
} else {
    $env:VOXLAB_TIER2 = "1"
    
    $runPattern = "TestTier2DeepAcousticMatrix"
    if ($IncludeMultilingual) {
        $runPattern = "TestTier2DeepAcousticMatrix|TestMultilingualDictations"
    }

    $testArgs = @("test", "-v", "-timeout", "${TimeoutMinutes}m", "./pkg/evaluation/", "-run", $runPattern)
    if ($Short) {
        $testArgs += "-short"
    }
    
    Write-Host "Running: go $($testArgs -join ' ')`n" -ForegroundColor Gray
    & go @testArgs
    $exitCode = $LASTEXITCODE
}

if ($exitCode -eq 0) {
    Write-Host "`n[PASS] Tier-2 evaluation suite passed successfully!" -ForegroundColor Green
} else {
    Write-Host "`n[FAIL] Tier-2 evaluation suite failed with exit code $exitCode." -ForegroundColor Red
}

exit $exitCode
