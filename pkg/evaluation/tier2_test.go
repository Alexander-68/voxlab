package evaluation_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"voxlab/pkg/config"
	"voxlab/pkg/engine"
	"voxlab/pkg/evaluation"
	"voxlab/pkg/server"
)

// requireDeepRun skips tests that feed Tier 2 audio in real time, which takes over 10 minutes.
// Run them with: VOXLAB_TIER2=1 go test ./pkg/evaluation/ -timeout 40m
func requireDeepRun(t *testing.T) {
	t.Helper()
	if os.Getenv("VOXLAB_TIER2") == "" {
		t.Skip("on-demand tier: set VOXLAB_TIER2=1")
	}
}

func TestTier2DeepAcousticMatrix(t *testing.T) {
	requireDeepRun(t)
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
		t.Skip("Sherpa models not installed; the simulator cannot transcribe real audio")
	}

	srv.StartBackgroundLoops()

	evalDir := filepath.Join("..", "..", "data", "test_samples", "eval_tier2")
	evalRunner := srv.EvalRunner()

	t.Logf("=== Starting Tier-2 Deep Acoustic Matrix Evaluation (%d cases) ===", len(itemsToTest))
	subManifest := &evaluation.EvalManifest{
		Version: manifest.Version,
		Tier:    manifest.Tier,
		Items:   itemsToTest,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
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

func TestMultilingualDictations(t *testing.T) {
	requireDeepRun(t)
	manifestPath := filepath.Join("..", "..", "data", "test_samples", "eval_tier2", "manifest.json")
	manifest, err := evaluation.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("failed loading Tier-2 manifest: %v", err)
	}

	// Select representative clean multilingual items
	selectedIDs := map[string]bool{
		"T2_CLEAN_DICT_ES_VITALS_EF":   true,
		"T2_CLEAN_DICT_PT_VITALS_PF":   true,
		"T2_CLEAN_DICT_FR_VITALS_FF":   true,
		"T2_CLEAN_DICT_IT_VITALS_IF":   true,
		"T2_CLEAN_DICT_ZH_VITALS_XB":   true,
		"T2_CLEAN_DICT_ZH_INCISION_YJ": true,
	}

	var itemsToTest []evaluation.EvalItem
	for _, item := range manifest.Items {
		if selectedIDs[item.ID] {
			itemsToTest = append(itemsToTest, item)
		}
	}

	if len(itemsToTest) == 0 {
		t.Fatalf("no multilingual items found in manifest")
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
		t.Skip("Sherpa models not installed; the simulator cannot transcribe real audio")
	}

	srv.StartBackgroundLoops()

	evalDir := filepath.Join("..", "..", "data", "test_samples", "eval_tier2")
	evalRunner := srv.EvalRunner()

	t.Logf("=== Starting Multilingual Dictation Test (%d cases) ===", len(itemsToTest))
	subManifest := &evaluation.EvalManifest{
		Version: manifest.Version,
		Tier:    manifest.Tier,
		Items:   itemsToTest,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	report, err := evalRunner.EvaluateSuite(ctx, subManifest, evalDir, func(res *evaluation.ItemResult, cur, tot int) {
		status := "PASS"
		if !res.Passed {
			status = "FAIL"
		}
		t.Logf("[%d/%d] %s [%s] lang=%s wer=%.2f cer=%.2f latency=%v (ref=%q -> hyp=%q)",
			cur, tot, status, res.ID, res.Accent, res.WER, res.CER, res.Duration, res.Description, res.Transcript)
	})
	if err != nil && err != context.Canceled {
		t.Fatalf("Multilingual evaluation failed with error: %v", err)
	}

	t.Logf("=== Multilingual Dictation Test Finished: %d/%d Passed (PassRate=%.1f%%, AvgWER=%.2f, AvgCER=%.2f) ===",
		report.Passed, report.TotalCases, report.PassRate*100.0, report.AvgWER, report.AvgCER)

	if report.TotalCases != len(itemsToTest) {
		t.Errorf("evaluated %d of %d items", report.TotalCases, len(itemsToTest))
	}
	for _, res := range report.Results {
		if res.Transcript == "" {
			t.Errorf("%s produced no transcript", res.ID)
		}
	}
}

