package audio

import (
	"math"
	"os"
	"testing"

	"voxlab/pkg/config"
)

func TestBypassEnhancer(t *testing.T) {
	b := NewBypassEnhancer(false)
	if b.IsEnabled() {
		t.Errorf("expected disabled")
	}
	b.SetEnabled(true)
	if !b.IsEnabled() {
		t.Errorf("expected enabled")
	}
	samples := []float32{0.1, 0.2, -0.3}
	out := b.ProcessChunk(samples)
	if len(out) != len(samples) || out[0] != samples[0] {
		t.Errorf("expected identical passthrough")
	}
	if err := b.SwitchModel("gtcrn"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if b.ModelType() != "gtcrn" {
		t.Errorf("expected modelType gtcrn, got %s", b.ModelType())
	}
}

func TestSpeechEnhancerFactoryAndProcessing(t *testing.T) {
	cfg := config.EnhancerConfig{
		Enabled:    true,
		ModelType:  "gtcrn",
		ModelPath:  "../../models/gtcrn_simple.onnx",
		NumThreads: 1,
		Provider:   "cpu",
	}

	enhancer := NewSpeechEnhancer("../../bin", "../../models", cfg)
	if enhancer == nil {
		t.Fatalf("enhancer must not be nil")
	}
	defer enhancer.Close()

	// If model exists, verify processing
	if _, err := os.Stat(cfg.ModelPath); err == nil {
		if !enhancer.IsEnabled() {
			t.Logf("enhancer not enabled or not on windows; skipping live processing check")
			return
		}

		// Feed a 480-sample chunk (30ms @ 16kHz)
		chunk := make([]float32, 480)
		for i := range chunk {
			chunk[i] = float32(math.Sin(float64(i) * 0.1))
		}

		out := enhancer.ProcessChunk(chunk)
		if len(out) != 480 {
			t.Errorf("expected 480 samples output, got %d", len(out))
		}

		// Test switching to dpdfnet if available
		if _, err := os.Stat("../../models/dpdfnet_baseline.onnx"); err == nil {
			err := enhancer.SwitchModel("dpdfnet")
			if err != nil {
				t.Errorf("SwitchModel(dpdfnet) failed: %v", err)
			}
			out2 := enhancer.ProcessChunk(chunk)
			if len(out2) != 480 {
				t.Errorf("expected 480 samples output for dpdfnet, got %d", len(out2))
			}
		}
	}
}
