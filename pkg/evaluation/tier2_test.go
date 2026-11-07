package evaluation_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"voxlab/pkg/config"
	"voxlab/pkg/engine"
	"voxlab/pkg/evaluation"
	"voxlab/pkg/server"
)

func TestTier2DeepAcousticMatrix(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "data", "test_samples", "eval_tier2", "manifest.json")
	manifest, err := evaluation.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("failed loading Tier-2 manifest: %v", err)
	}

	if len(manifest.Items) == 0 {
		t.Fatal("Tier-2 manifest contains 0 items")
	}

	// In short mode (go test -short), evaluate a representative 10-item slice of clean, SNR ladder, and perturbations
	itemsToTest := manifest.Items
	if testing.Short() {
		if len(itemsToTest) > 10 {
			itemsToTest = itemsToTest[:10]
		}
		t.Logf("[Short Mode] Limiting Tier-2 test run to %d representative items", len(itemsToTest))
	}

	cfg := config.DefaultConfig()
	cfg.StaticDir = filepath.Join("..", "..", "web")
	cfg.Intent.CatalogPath = filepath.Join("..", "..", "data", "commands.json")

	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("failed creating server: %v", err)
	}
	defer func() { _ = srv.Stop(context.Background()) }()

	runner, ok := srv.SherpaEngine().(*engine.SherpaRunner)
	isSherpa := ok && runner.Name() == "sherpa_onnx_native"
	if isSherpa {
		for i := 0; i < 300 && !(runner.ASRReady() && runner.ASRModelInfo()["second_pass_loaded"] == true); i++ {
			time.Sleep(100 * time.Millisecond)
		}
		if !runner.ASRReady() {
			t.Skip("Sherpa warm ASR not ready within timeout")
		}
		if err := srv.SetASREngineMode("sherpa"); err != nil {
			t.Fatalf("failed setting ASR engine mode: %v", err)
		}
	} else {
		t.Log("[Notice] Sherpa native models not present; testing with simulator engine")
	}

	srv.StartBackgroundLoops()

	evalDir := filepath.Join("..", "..", "data", "test_samples", "eval_tier2")
	evalRunner := evaluation.NewEvalRunner(srv.HandleWebSocket)

	t.Logf("=== Starting Tier-2 Deep Acoustic Matrix Evaluation (%d cases) ===", len(itemsToTest))
	subManifest := &evaluation.EvalManifest{
		Version: manifest.Version,
		Tier:    manifest.Tier,
		Items:   itemsToTest,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	report, err := evalRunner.EvaluateSuite(ctx, subManifest, evalDir, func(res *evaluation.ItemResult, cur, tot int) {
		status := "PASS"
		if !res.Passed {
			status = "FAIL"
		}
		t.Logf("[%d/%d] %s [%s] mode=%s wer=%.2f cer=%.2f latency=%v (transcript=%q intent=%q reason=%q)",
			cur, tot, status, res.ID, res.Mode, res.WER, res.CER, res.Duration, res.Transcript, res.IntentID, res.Reason)
	})
	if err != nil && err != context.Canceled {
		t.Fatalf("Tier-2 evaluation failed with error: %v", err)
	}

	t.Logf("=== Tier-2 Deep Acoustic Matrix Finished: %d/%d Passed (PassRate=%.1f%%, AvgWER=%.2f, IntentAcc=%.1f%%) ===",
		report.Passed, report.TotalCases, report.PassRate*100.0, report.AvgWER, report.IntentAccuracy*100.0)

	if report.PassRate < 0.80 {
		t.Fatalf("Tier-2 pass rate below acceptable threshold: got %.1f%%", report.PassRate*100.0)
	}
}

func TestCalculateCER(t *testing.T) {
	// Identical
	if cer := evaluation.CalculateCER("open settings", "open settings"); cer != 0.0 {
		t.Errorf("expected 0.0 CER for identical strings, got %f", cer)
	}

	// 1 substitution out of 12 chars = ~0.083
	if cer := evaluation.CalculateCER("open settings", "open settingz"); mathAbs(cer-1.0/12.0) > 0.01 {
		t.Errorf("expected ~0.083 CER, got %f", cer)
	}

	// Normalization
	if cer := evaluation.CalculateCER("Open settings!", "open settings"); cer != 0.0 {
		t.Errorf("expected 0.0 CER after normalization, got %f", cer)
	}
}

func mathAbs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

