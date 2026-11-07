package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"voxlab/pkg/audio"
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
		if err := generateTier1(t1Dir, runner); err != nil {
			runner.Close() // log.Fatal skips the deferred Close
			log.Fatalf("failed generating Tier 1 corpus: %v", err)
		}
	}

	if *tierFlag == "2" || *tierFlag == "all" {
		t2Dir := *outFlag
		if t2Dir == "" || *tierFlag == "all" {
			t2Dir = filepath.Join("data", "test_samples", "eval_tier2")
		}
		log.Printf("=== Generating Tier 2 Deep Acoustic Matrix in %s ===", t2Dir)
		manifest, err := evaluation.GenerateTier2Corpus(t2Dir, runner)
		if err != nil {
			runner.Close() // log.Fatal skips the deferred Close
			log.Fatalf("failed generating Tier 2 corpus: %v", err)
		}
		log.Printf("Successfully generated Tier 2 Deep Acoustic Matrix: %d items in %s", len(manifest.Items), t2Dir)
	}
}

func generateTier1(outDir string, runner engine.SpeechEngine) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	exact := 0.0
	cases := []evaluation.EvalItem{
		{ID: "T1_CMD_SETTINGS", Description: "Command: Open Settings (US Female voice)", Mode: evaluation.ModeCommand,
			Text: "open settings", Voice: "af_heart", Expected: "NAV_SETTINGS", WavFile: "t1_cmd_settings.wav"},
		{ID: "T1_CMD_START_REC", Description: "Command: Start Recording (US Male voice)", Mode: evaluation.ModeCommand,
			Text: "start recording", Voice: "am_adam", Expected: "START_RECORDING", WavFile: "t1_cmd_start_recording.wav"},
		{ID: "T1_CMD_NEGATION", Description: "Command: Negation Rejection (US Female voice)", Mode: evaluation.ModeCommand,
			Text: "do not stop recording", Voice: "af_heart", Expected: "REJECT:NEGATION_DETECTED", WavFile: "t1_cmd_negation.wav"},
		{ID: "T1_DICT_BENCH", Description: "Dictation: Technical phrase (British Female voice)", Mode: evaluation.ModeDictation,
			Text: "The voice pipeline test bench is operational and ready.", Voice: "bf_emma", Expected: "pipeline test bench",
			MaxErrorRate: &exact, WavFile: "t1_dict_bench.wav"},
	}

	manifest := evaluation.EvalManifest{Version: "1.0", Tier: "tier-1-smoke"}
	for _, c := range cases {
		samples, err := evaluation.LoadOrSynthesizeClip(filepath.Join(outDir, c.WavFile), runner, engine.TTSRequest{
			Text: c.Text, Voice: c.Voice, Speed: 1.0, SampleRate: 24000,
		})
		if err != nil {
			return fmt.Errorf("%s: %w", c.ID, err)
		}
		c.DurationSec = float64(len(samples)) / 16000.0
		manifest.Items = append(manifest.Items, c)
	}

	// Low-level hiss plus faint periodic clicks, both below the gate.
	noise := make([]float32, 16000*2)
	for i := range noise {
		noise[i] = float32((float64(i%997)/997.0 - 0.5) * 0.006)
		if i%3200 == 0 {
			noise[i] += 0.015
		}
	}
	noiseBytes, err := audio.EncodeWAV(noise, 16000)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "t1_noise_below_gate.wav"), noiseBytes, 0644); err != nil {
		return err
	}
	manifest.Items = append(manifest.Items, evaluation.EvalItem{
		ID:          "T1_NOISE_GATE",
		Description: "Acoustic Noise: Low-level typing/room chatter below gate threshold",
		Mode:        evaluation.ModeCommand,
		Voice:       "synthetic",
		Expected:    "DROPPED_BY_GATE",
		WavFile:     "t1_noise_below_gate.wav",
		DurationSec: 2.0,
	})

	manBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "manifest.json"), manBytes, 0644)
}
