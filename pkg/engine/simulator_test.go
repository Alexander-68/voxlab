package engine

import (
	"math"
	"testing"
)

func TestSimulatorEngine(t *testing.T) {
	sim := NewSimulatorEngine()

	// VAD test on quiet vs loud chunk
	quietChunk := make([]float32, 480)
	isVoice, prob := sim.DetectVAD(quietChunk)
	if isVoice {
		t.Errorf("expected quiet chunk to not be detected as voice (prob: %f)", prob)
	}

	loudChunk := make([]float32, 480)
	for i := range loudChunk {
		loudChunk[i] = float32(0.2 * math.Sin(2.0*math.Pi*300.0*float64(i)/16000.0))
	}
	isVoiceLoud, probLoud := sim.DetectVAD(loudChunk)
	if !isVoiceLoud || probLoud < 0.70 {
		t.Errorf("expected loud chunk to be detected as voice (prob: %f)", probLoud)
	}

	// TTS Synthesize test
	req := TTSRequest{
		Text:       "VoxLab voice pipeline initialized.",
		Voice:      "af_heart",
		Speed:      1.0,
		SampleRate: 24000,
	}
	res, err := sim.Synthesize(req)
	if err != nil {
		t.Fatalf("simulator synthesis failed: %v", err)
	}
	if len(res.WAVBytes) < 100 {
		t.Errorf("expected valid WAV bytes, got length %d", len(res.WAVBytes))
	}
	if res.DurationSec <= 0.0 {
		t.Errorf("expected positive duration, got %f", res.DurationSec)
	}
}
