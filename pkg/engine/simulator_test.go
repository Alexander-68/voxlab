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

func TestSimulatorFlushReturnsPendingAndWakeIsIndependent(t *testing.T) {
	sim := NewSimulatorEngine()
	loud := make([]float32, 480)
	for i := range loud {
		loud[i] = 0.2
	}
	for range 14 {
		if _, err := sim.ProcessASRChunk(loud, false); err != nil {
			t.Fatal(err)
		}
	}
	// 14 ASR speech chunks must not prime the wake-word streak.
	if hit, _, _ := sim.DetectWakeWord(loud); hit {
		t.Fatal("ASR speech leaked into the wake-word counter")
	}
	if got := sim.FlushASR(); got != "open settings" {
		t.Fatalf("FlushASR = %q, want pending partial", got)
	}
	if got := sim.FlushASR(); got != "" {
		t.Fatalf("second FlushASR = %q, want empty (consumed)", got)
	}
	if s := clampSpeed(1e-300); s != 0.5 {
		t.Fatalf("clampSpeed(tiny) = %v", s)
	}
	if s := clampSpeed(math.NaN()); s != 1.0 {
		t.Fatalf("clampSpeed(NaN) = %v", s)
	}
}
