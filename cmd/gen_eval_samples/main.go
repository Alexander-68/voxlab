package main

import (
	"flag"
	"log"
	"path/filepath"

	"voxlab/pkg/config"
	"voxlab/pkg/engine"
	"voxlab/pkg/evaluation"
)

func main() {
	tierFlag := flag.String("tier", "2", "Evaluation tier to generate: '1', '2', or 'all'")
	outFlag := flag.String("out", "", "Custom output directory (default: data/test_samples/eval or eval_tier2)")
	flag.Parse()

	cfg := config.DefaultConfig()
	cfg.KWS.Enabled = false
	runner := engine.NewSherpaRunner(cfg)
	defer runner.Close()

	if *tierFlag == "1" || *tierFlag == "all" {
		t1Dir := *outFlag
		if t1Dir == "" || *tierFlag == "all" {
			t1Dir = filepath.Join("data", "test_samples", "eval")
		}
		log.Printf("=== Generating Tier 1 Smoke Corpus in %s ===", t1Dir)
		manifest, err := evaluation.GenerateTier1Corpus(t1Dir, runner)
		if err != nil {
			runner.Close()
			log.Fatalf("failed generating Tier 1 corpus: %v", err)
		}
		log.Printf("Successfully generated Tier 1 Smoke Corpus: %d items in %s", len(manifest.Items), t1Dir)
	}

	if *tierFlag == "2" || *tierFlag == "all" {
		t2Dir := *outFlag
		if t2Dir == "" || *tierFlag == "all" {
			t2Dir = filepath.Join("data", "test_samples", "eval_tier2")
		}
		log.Printf("=== Generating Tier 2 Deep Acoustic Matrix in %s ===", t2Dir)
		manifest, err := evaluation.GenerateTier2Corpus(t2Dir, runner)
		if err != nil {
			runner.Close()
			log.Fatalf("failed generating Tier 2 corpus: %v", err)
		}
		log.Printf("Successfully generated Tier 2 Deep Acoustic Matrix: %d items in %s", len(manifest.Items), t2Dir)
	}
}
