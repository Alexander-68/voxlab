package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"voxlab/pkg/config"
	"voxlab/pkg/engine"
	"voxlab/pkg/evaluation"
	"voxlab/pkg/server"
)

func main() { os.Exit(run()) }

// run returns the exit code, so deferred cleanup stops the ASR daemons on every path.
func run() int {
	tierFlag := flag.String("tier", "2", "Evaluation tier to benchmark: '1' (smoke) or '2' (deep acoustic matrix)")
	manifestFlag := flag.String("manifest", "", "Custom path to manifest.json (overrides default tier directory)")
	quickFlag := flag.Bool("quick", false, "Quick run: limit to first 12 cases for fast turnaround")
	outFlag := flag.String("out", "", "Path to write JSON evaluation report")
	timeoutSec := flag.Int("timeout", 1800, "Maximum benchmark execution timeout in seconds")
	flag.Parse()

	tier := *tierFlag
	if tier != "1" && tier != "2" {
		tier = "2"
	}

	manifestPath := *manifestFlag
	if manifestPath == "" {
		if tier == "1" {
			manifestPath = filepath.Join("data", "test_samples", "eval", "manifest.json")
		} else {
			manifestPath = filepath.Join("data", "test_samples", "eval_tier2", "manifest.json")
		}
	}

	manifestDir := filepath.Dir(manifestPath)
	manifest, err := evaluation.LoadManifest(manifestPath)
	if err != nil {
		log.Fatalf("[Error] Failed loading manifest at %s: %v\nRun 'go run ./cmd/gen_eval_samples -tier %s' first to generate corpus.", manifestPath, err, tier)
	}

	if *quickFlag && len(manifest.Items) > 12 {
		manifest.Items = manifest.Items[:12]
	}

	fmt.Println("=========================================================================================")
	fmt.Printf(" VoxLab Synthetic Round-Trip Acoustic Benchmark Harness (Tier %s)\n", tier)
	fmt.Printf(" Manifest: %s (%d test cases)\n", manifestPath, len(manifest.Items))
	fmt.Println("=========================================================================================")

	// 1. Initialize server & Sherpa warm native speech recognition
	cfg := config.DefaultConfig()
	cfg.Intent.CatalogPath = filepath.Join("data", "commands.json")

	srv, err := server.NewServer(cfg)
	if err != nil {
		log.Printf("[Fatal] Failed initializing VoxLab server: %v", err)
		return 1
	}
	defer func() { _ = srv.Stop(context.Background()) }()

	runner, ok := srv.SherpaEngine().(*engine.SherpaRunner)
	isSherpa := ok && runner.Name() == "sherpa_onnx_native"
	if isSherpa {
		fmt.Print(">> Warming up streaming ASR and Parakeet Ultra 2nd pass...")
		for i := 0; i < 300 && !(runner.ASRReady() && runner.ASRModelInfo()["second_pass_loaded"] == true); i++ {
			time.Sleep(100 * time.Millisecond)
		}
		if !runner.ASRReady() {
			log.Print("Sherpa warm ASR not ready within timeout")
			return 1
		}
		if err := srv.SetASREngineMode("sherpa"); err != nil {
			log.Printf("failed setting ASR engine mode to sherpa: %v", err)
			return 1
		}
		fmt.Println(" [READY]")
	} else {
		fmt.Println(">> Notice: Native Sherpa models not detected; using simulator ASR engine.")
	}

	srv.StartBackgroundLoops()
	evalRunner := srv.EvalRunner()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeoutSec)*time.Second)
	defer cancel()

	fmt.Printf("\n>> Executing %d test cases...\n", len(manifest.Items))
	fmt.Printf("%-6s | %-32s | %-8s | %-12s | %-6s | %-8s | %-16s | %s\n",
		"IDX", "TEST ID", "MODE", "SNR/NOISE", "VERDICT", "WER", "LATENCY", "OUTCOME")
	fmt.Println(strings.Repeat("-", 108))

	report, err := evalRunner.EvaluateSuite(ctx, manifest, manifestDir, func(res *evaluation.ItemResult, cur, tot int) {
		verdict := "PASS"
		if !res.Passed {
			verdict = "FAIL"
		}

		snrDesc := "Clean"
		if res.SNR != nil {
			snrDesc = fmt.Sprintf("%+.0fdB %s", *res.SNR, res.NoiseType)
		} else if res.NoiseType != "" && res.NoiseType != "clean" {
			snrDesc = res.NoiseType
		}

		outcome := res.IntentID
		if res.Decision == "REJECT" {
			outcome = fmt.Sprintf("REJECT:%s", res.Reason)
		} else if res.Decision == "SILENCE" {
			outcome = "(gate silence)"
		} else if res.Mode == evaluation.ModeDictation {
			outcome = truncateString(res.Transcript, 25)
		}

		fmt.Printf("[%2d/%2d] | %-32s | %-8s | %-12s | %-6s | %6.2f | %8s | %s\n",
			cur, tot, truncateString(res.ID, 32), res.Mode, truncateString(snrDesc, 12),
			verdict, res.WER, fmt.Sprintf("%dms", res.Duration.Milliseconds()), outcome)
	})

	if err != nil {
		fmt.Printf("\n[Error] Benchmark did not finish: %v\n", err)
	}

	// 2. Render ASCII Reporting Tables
	printReportSummary(report)

	// 3. Save report JSON
	outPath := *outFlag
	if outPath == "" {
		outPath = filepath.Join("data", fmt.Sprintf("eval_report_%s.json", manifest.Tier)) // the name the server reads
	}
	reportJSON, saveErr := json.MarshalIndent(report, "", "  ")
	if saveErr == nil {
		saveErr = os.WriteFile(outPath, reportJSON, 0644)
	}
	if saveErr != nil {
		fmt.Printf("\n[Error] Could not save report to %s: %v\n", outPath, saveErr)
		return 1
	}
	fmt.Printf("\nFull JSON evaluation report saved to: %s\n\n", outPath)

	minPass := 0.80 // same bar as the Tier 2 test
	if tier == "1" {
		minPass = 1.0
	}
	if err != nil || report.PassRate < minPass {
		fmt.Printf("[FAIL] Pass rate %.1f%%, required %.0f%%\n", report.PassRate*100, minPass*100)
		return 1
	}
	return 0
}

func printReportSummary(r *evaluation.SuiteReport) {
	fmt.Println("\n=========================================================================================")
	fmt.Printf(" BENCHMARK REPORT: Tier %s Evaluation\n", r.Tier)
	fmt.Println("=========================================================================================")
	fmt.Printf(" Total Tests:       %-4d | Passed:            %-4d | Failed:        %d\n", r.TotalCases, r.Passed, r.Failed)
	fmt.Printf(" Pass Rate:         %5.1f%% | Mean WER:          %5.2f | Mean CER:      %5.2f\n", r.PassRate*100.0, r.AvgWER, r.AvgCER)
	fmt.Printf(" Intent Accuracy:   %6s | Rejection Prec:    %6s | Gate Prec:     %6s\n", pct(r.IntentAccuracy), pct(r.RejectionAccuracy), pct(r.GatePrecision))
	fmt.Printf(" Total Elapsed:     %v\n", r.Duration.Round(time.Millisecond))

	// SNR Degradation Ladder Table
	if len(r.SNRCurve) > 0 {
		fmt.Println("\n--- [SNR Degradation Ladder] ---")
		fmt.Printf("%-10s | %-6s | %-8s | %-8s | %-8s | %-12s | %-10s\n",
			"SNR LEVEL", "COUNT", "PASSED", "PASS %", "AVG WER", "INTENT ACC", "AVG LATENCY")
		fmt.Println(strings.Repeat("-", 76))
		for _, pt := range r.SNRCurve {
			fmt.Printf("%-10s | %-6d | %-8d | %6.1f%% | %8.3f | %10.1f%% | %8.0f ms\n",
				pt.Label, pt.Count, pt.Passed, pt.PassRate*100.0, pt.AvgWER, pt.IntentAccuracy*100.0, pt.AvgLatencyMs)
		}
	}

	// Acoustic Latency Percentiles
	fmt.Println("\n--- [Acoustic Latencies & Percentiles] ---")
	fmt.Printf("%-24s | %-8s | %-8s | %-8s | %-8s | %-8s | %-8s\n",
		"METRIC", "AVG", "P50", "P90", "P95", "P99", "MAX")
	fmt.Println(strings.Repeat("-", 80))
	printLatencyLine("TTFT (Time to First Token)", r.LatencyStats.TTFT)
	printLatencyLine("Silence Endpoint Delay", r.LatencyStats.EndpointDelay)
	printLatencyLine("2nd-Pass Refinement Overhead", r.LatencyStats.SecondPass)
	printLatencyLine("Total Pipeline Turnaround", r.LatencyStats.TotalDuration)

	// Acoustic Condition Breakdown
	if len(r.NoiseBreakdown) > 0 {
		fmt.Println("\n--- [Acoustic Noise Condition Breakdown] ---")
		fmt.Printf("%-16s | %-6s | %-8s | %-8s | %-8s | %-8s\n",
			"CONDITION", "COUNT", "PASSED", "PASS %", "AVG WER", "AVG CER")
		fmt.Println(strings.Repeat("-", 68))
		for k, g := range r.NoiseBreakdown {
			fmt.Printf("%-16s | %-6d | %-8d | %6.1f%% | %8.3f | %8.3f\n",
				k, g.Count, g.Passed, g.PassRate*100.0, g.AvgWER, g.AvgCER)
		}
	}

	// Voice Profile Breakdown
	if len(r.VoiceBreakdown) > 0 {
		fmt.Println("\n--- [Voice Profile & Accent Breakdown] ---")
		fmt.Printf("%-16s | %-6s | %-8s | %-8s | %-8s | %-8s\n",
			"VOICE", "COUNT", "PASSED", "PASS %", "AVG WER", "AVG CER")
		fmt.Println(strings.Repeat("-", 68))
		for k, g := range r.VoiceBreakdown {
			fmt.Printf("%-16s | %-6d | %-8d | %6.1f%% | %8.3f | %8.3f\n",
				k, g.Count, g.Passed, g.PassRate*100.0, g.AvgWER, g.AvgCER)
		}
	}
	fmt.Println("=========================================================================================")
}

func printLatencyLine(label string, p evaluation.LatencyPercentiles) {
	fmt.Printf("%-24s | %6.0fms | %6.0fms | %6.0fms | %6.0fms | %6.0fms | %6.0fms\n",
		label, p.AvgMs, p.P50Ms, p.P90Ms, p.P95Ms, p.P99Ms, p.MaxMs)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func pct(v *float64) string {
	if v == nil {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%%", *v*100.0)
}
