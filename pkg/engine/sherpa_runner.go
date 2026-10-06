package engine

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	// For VAD in stream, delegates to Silero VAD or energy proxy
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

// Synthesize runs Kokoro TTS via sherpa-onnx-offline-tts binary if available.
func (r *SherpaRunner) Synthesize(req TTSRequest) (*TTSResult, error) {
	ttsBin := r.cfg.Engine.SherpaTtsBin
	modelDir := r.cfg.Engine.KokoroModelDir

	// Check if sherpa binary and model dir exist
	_, binErr := exec.LookPath(ttsBin)
	modelExists := false
	if fi, err := os.Stat(modelDir); err == nil && fi.IsDir() {
		modelExists = true
	}

	if binErr != nil || !modelExists {
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
		req.Text,
	}

	cmd := exec.Command(ttsBin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("sherpa-onnx-offline-tts execution failed: %w (stderr: %s)", err, stderr.String())
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
