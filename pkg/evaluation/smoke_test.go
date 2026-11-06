package evaluation

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"voxlab/pkg/config"
	"voxlab/pkg/engine"
	"voxlab/pkg/server"
)

func TestTier1SmokeSuite(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "data", "test_samples", "eval", "manifest.json")
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("failed loading Tier-1 manifest: %v", err)
	}

	if len(manifest.Items) == 0 {
		t.Fatal("manifest contains 0 items")
	}

	// Create test server configuration
	cfg := config.DefaultConfig()
	cfg.StaticDir = filepath.Join("..", "..", "web")
	cfg.Intent.CatalogPath = filepath.Join("..", "..", "data", "commands.json")

	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("failed creating server: %v", err)
	}
	defer func() { _ = srv.Stop(context.Background()) }()

	// If Sherpa native runner is available, ensure models are warm
	runner, ok := srv.SherpaEngine().(*engine.SherpaRunner)
	isSherpa := ok && runner.Name() == "sherpa_onnx_native"
	if isSherpa {
		// Wait for warm streaming ASR and second-pass ASR to accept connections (up to 30s)
		for i := 0; i < 300 && !(runner.ASRReady() && runner.ASRModelInfo()["second_pass_loaded"] == true); i++ {
			time.Sleep(100 * time.Millisecond)
		}
		if !runner.ASRReady() {
			t.Skip("Sherpa warm ASR not ready within timeout")
		}
		if err := srv.SetASREngineMode("sherpa"); err != nil {
			t.Fatalf("failed setting ASR engine mode to sherpa: %v", err)
		}
	} else {
		t.Log("[Notice] Sherpa native models not present; testing with simulator engine")
	}

	srv.StartBackgroundLoops()

	evalDir := filepath.Join("..", "..", "data", "test_samples", "eval")
	evalRunner := NewEvalRunner(srv)

	t.Logf("=== Starting Tier-1 Smoke Suite (%d cases) ===", len(manifest.Items))

	passedCount := 0
	for _, item := range manifest.Items {
		wavPath := ResolveWavPath(evalDir, item)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		res, err := evalRunner.EvaluateItem(ctx, item, wavPath)
		cancel()

		if err != nil {
			t.Errorf("[%s] Execution error: %v", item.ID, err)
			continue
		}

		if !res.Passed {
			t.Errorf("[%s] FAILED: expected %q, got intent=%q reason=%q transcript=%q",
				item.ID, item.Expected, res.IntentID, res.Reason, res.Transcript)
		} else {
			passedCount++
			t.Logf("PASS [%s] (%s, %s): text=%q -> got intent=%q dec=%s latency=%v WER=%.2f",
				item.ID, item.Mode, item.Voice, item.Text, res.IntentID, res.Decision, res.Duration, res.WER)
		}
	}

	t.Logf("=== Tier-1 Smoke Suite Finished: %d/%d Passed ===", passedCount, len(manifest.Items))
	if passedCount != len(manifest.Items) {
		t.Fatalf("Tier-1 Smoke Suite did not pass completely (%d/%d)", passedCount, len(manifest.Items))
	}
}

func TestCalculateWER(t *testing.T) {
	// Identical
	if wer := CalculateWER("open settings", "open settings"); wer != 0.0 {
		t.Errorf("expected 0.0 WER for identical strings, got %f", wer)
	}

	// 1 substitution out of 2 words = 0.5
	if wer := CalculateWER("open settings", "open window"); wer != 0.5 {
		t.Errorf("expected 0.5 WER for 1 substitution, got %f", wer)
	}

	// Case & punctuation normalization
	if wer := CalculateWER("Open settings!", "open settings"); wer != 0.0 {
		t.Errorf("expected 0.0 WER after normalization, got %f", wer)
	}
}
