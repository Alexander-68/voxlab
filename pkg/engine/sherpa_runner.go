package engine

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"voxlab/pkg/audio"
	"voxlab/pkg/config"
)

// SherpaRunner executes official Sherpa-ONNX binaries or falls back to simulation.
type SherpaRunner struct {
	cfg       *config.AppConfig
	simulator *SimulatorEngine
}

// NewSherpaRunner creates a new SherpaRunner instance.
func NewSherpaRunner(cfg *config.AppConfig) *SherpaRunner {
	return &SherpaRunner{
		cfg:       cfg,
		simulator: NewSimulatorEngine(),
	}
}

func (r *SherpaRunner) Name() string { return "sherpa_onnx_native" }

func (r *SherpaRunner) DetectVAD(chunk []float32) (bool, float32) {
	return r.simulator.DetectVAD(chunk)
}

func (r *SherpaRunner) DetectWakeWord(chunk []float32) (bool, string, float64) {
	return r.simulator.DetectWakeWord(chunk)
}

func (r *SherpaRunner) ProcessASRChunk(chunk []float32, isDictation bool) (*ASRResult, error) {
	return r.simulator.ProcessASRChunk(chunk, isDictation)
}

func (r *SherpaRunner) ResetASR() {
	r.simulator.ResetASR()
}

// findKokoroModelDir automatically discovers the latest available Kokoro model directory.
func findKokoroModelDir(configured string) (string, bool) {
	candidates := []string{
		configured,
		filepath.Join("models", "kokoro-multi-lang-v1_1"),
		filepath.Join("models", "kokoro-multi-lang-v1_0"),
		filepath.Join("models", "kokoro-en-v0_19"),
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			if _, err := os.Stat(filepath.Join(c, "model.onnx")); err == nil {
				return c, true
			}
		}
	}
	return configured, false
}

// findSherpaBin searches for a sherpa-onnx binary on PATH or local folders.
func findSherpaBin(name string) (string, bool) {
	// 1. Direct LookPath
	if path, err := exec.LookPath(name); err == nil {
		return path, true
	}

	// 2. Local bin/ or models/bin/ directories
	candidates := []string{
		filepath.Join("bin", name),
		filepath.Join("bin", name+".exe"),
		filepath.Join("models", "bin", name),
		filepath.Join("models", "bin", name+".exe"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, true
		}
	}

	// 3. Search within models subfolders for unpacked binaries
	matches, _ := filepath.Glob(filepath.Join("models", "*", "bin", name+"*"))
	if len(matches) > 0 {
		return matches[0], true
	}
	matchesBin, _ := filepath.Glob(filepath.Join("bin", "*", "bin", name+"*"))
	if len(matchesBin) > 0 {
		return matchesBin[0], true
	}

	return name, false
}

// Synthesize runs Kokoro TTS via sherpa-onnx-offline-tts binary if available.
func (r *SherpaRunner) Synthesize(req TTSRequest) (*TTSResult, error) {
	modelDir, modelExists := findKokoroModelDir(r.cfg.Engine.KokoroModelDir)
	ttsBin, binExists := findSherpaBin(r.cfg.Engine.SherpaTtsBin)

	if !binExists || !modelExists {
		// Gracefully fall back to simulator if real model/binary is not yet downloaded
		return r.simulator.Synthesize(req)
	}

	startTime := time.Now()
	tempWav := filepath.Join(os.TempDir(), fmt.Sprintf("voxlab_tts_%d.wav", time.Now().UnixNano()))
	defer os.Remove(tempWav)

	voicesPath := filepath.Join(modelDir, "voices.bin")
	tokensPath := filepath.Join(modelDir, "tokens.txt")
	dataDirPath := filepath.Join(modelDir, "espeak-ng-data")

	args := []string{
		fmt.Sprintf("--kokoro-model-dir=%s", modelDir),
		fmt.Sprintf("--kokoro-voices=%s", voicesPath),
		fmt.Sprintf("--kokoro-tokens=%s", tokensPath),
		fmt.Sprintf("--kokoro-data-dir=%s", dataDirPath),
		fmt.Sprintf("--tts-wav-output=%s", tempWav),
	}

	// Map speaker voice if sid parameter is supported
	sid := 0
	if strings.Contains(req.Voice, "alloy") {
		sid = 1
	} else if strings.Contains(req.Voice, "aoede") {
		sid = 2
	} else if strings.Contains(req.Voice, "bella") {
		sid = 3
	} else if strings.Contains(req.Voice, "adam") {
		sid = 4
	}
	args = append(args, fmt.Sprintf("--sid=%d", sid))
	args = append(args, req.Text)

	cmd := exec.Command(ttsBin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("[SherpaRunner] Execution error (%v), falling back to simulator: %s", err, stderr.String())
		return r.simulator.Synthesize(req)
	}

	wavData, err := os.ReadFile(tempWav)
	if err != nil {
		return nil, fmt.Errorf("failed reading output wav: %w", err)
	}

	samples, sRate, err := audio.DecodeWAV(bytes.NewReader(wavData))
	if err != nil {
		return nil, fmt.Errorf("failed decoding output wav: %w", err)
	}

	duration := float64(len(samples)) / float64(sRate)
	latency := time.Since(startTime).Milliseconds()

	return &TTSResult{
		AudioSamples: samples,
		WAVBytes:     wavData,
		SampleRate:   sRate,
		DurationSec:  duration,
		LatencyMs:    latency,
	}, nil
}
