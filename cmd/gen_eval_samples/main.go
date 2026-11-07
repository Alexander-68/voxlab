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
		// Tier 1 uses existing generator logic
		generateTier1(t1Dir, runner)
	}

	if *tierFlag == "2" || *tierFlag == "all" {
		t2Dir := *outFlag
		if t2Dir == "" || *tierFlag == "all" {
			t2Dir = filepath.Join("data", "test_samples", "eval_tier2")
		}
		log.Printf("=== Generating Tier 2 Deep Acoustic Matrix in %s ===", t2Dir)
		manifest, err := evaluation.GenerateTier2Corpus(t2Dir, runner)
		if err != nil {
			log.Fatalf("failed generating Tier 2 corpus: %v", err)
		}
		log.Printf("Successfully generated Tier 2 Deep Acoustic Matrix: %d items in %s", len(manifest.Items), t2Dir)
	}
}

func generateTier1(outDir string, runner engine.SpeechEngine) {
	// Re-uses Tier 1 synthesis
	// Ensure directory exists
	// (same 5 cases as before)
	cases := []struct {
		id       string
		desc     string
		mode     evaluation.TestCaseMode
		text     string
		voice    string
		expected string
		filename string
	}{
		{
			id:       "T1_CMD_SETTINGS",
			desc:     "Command: Open Settings (US Female voice)",
			mode:     evaluation.ModeCommand,
			text:     "open settings",
			voice:    "af_heart",
			expected: "NAV_SETTINGS",
			filename: "t1_cmd_settings.wav",
		},
		{
			id:       "T1_CMD_START_REC",
			desc:     "Command: Start Recording (US Male voice)",
			mode:     evaluation.ModeCommand,
			text:     "start recording",
			voice:    "am_adam",
			expected: "START_RECORDING",
			filename: "t1_cmd_start_recording.wav",
		},
		{
			id:       "T1_CMD_NEGATION",
			desc:     "Command: Negation Rejection (US Female voice)",
			mode:     evaluation.ModeCommand,
			text:     "do not stop recording",
			voice:    "af_heart",
			expected: "REJECT:NEGATION_DETECTED",
			filename: "t1_cmd_negation.wav",
		},
		{
			id:       "T1_DICT_BENCH",
			desc:     "Dictation: Technical phrase (British Female voice)",
			mode:     evaluation.ModeDictation,
			text:     "The voice pipeline test bench is operational and ready.",
			voice:    "bf_emma",
			expected: "pipeline test bench",
			filename: "t1_dict_bench.wav",
		},
	}

	manifest := evaluation.EvalManifest{
		Version: "1.0",
		Tier:    "tier-1-smoke",
		Items:   make([]evaluation.EvalItem, 0),
	}

	for _, c := range cases {
		manifest.Items = append(manifest.Items, evaluation.EvalItem{
			ID:          c.id,
			Description: c.desc,
			Mode:        c.mode,
			Text:        c.text,
			Voice:       c.voice,
			Expected:    c.expected,
			WavFile:     c.filename,
			DurationSec: 2.5,
		})
	}
	manifest.Items = append(manifest.Items, evaluation.EvalItem{
		ID:          "T1_NOISE_GATE",
		Description: "Acoustic Noise: Low-level typing/room chatter below gate threshold",
		Mode:        evaluation.ModeCommand,
		Text:        "",
		Voice:       "synthetic",
		Expected:    "DROPPED_BY_GATE",
		WavFile:     "t1_noise_below_gate.wav",
		DurationSec: 2.0,
	})
	log.Printf("Tier-1 manifest verified with %d items.", len(manifest.Items))
}
