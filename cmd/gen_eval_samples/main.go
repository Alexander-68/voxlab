package main

import (
	"encoding/json"
	"log"
	"math"
	"os"
	"path/filepath"

	"voxlab/pkg/audio"
	"voxlab/pkg/config"
	"voxlab/pkg/engine"
)

type EvalManifestItem struct {
	ID          string  `json:"id"`
	Description string  `json:"description"`
	Mode        string  `json:"mode"`
	Text        string  `json:"text"`
	Voice       string  `json:"voice"`
	Expected    string  `json:"expected"`
	WavFile     string  `json:"wav_file"`
	DurationSec float64 `json:"duration_sec"`
}

type EvalManifest struct {
	Version string             `json:"version"`
	Tier    string             `json:"tier"`
	Items   []EvalManifestItem `json:"items"`
}

func main() {
	outDir := filepath.Join("data", "test_samples", "eval")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("failed creating eval sample dir: %v", err)
	}

	cfg := config.DefaultConfig()
	// Disable continuous KWS and streaming ASR server warmup during sample generation
	cfg.KWS.Enabled = false
	runner := engine.NewSherpaRunner(cfg)
	defer runner.Close()

	cases := []struct {
		id       string
		desc     string
		mode     string
		text     string
		voice    string
		expected string
		filename string
	}{
		{
			id:       "T1_CMD_SETTINGS",
			desc:     "Command: Open Settings (US Female voice)",
			mode:     "command",
			text:     "open settings",
			voice:    "af_heart",
			expected: "NAV_SETTINGS",
			filename: "t1_cmd_settings.wav",
		},
		{
			id:       "T1_CMD_START_REC",
			desc:     "Command: Start Recording (US Male voice)",
			mode:     "command",
			text:     "start recording",
			voice:    "am_adam",
			expected: "START_RECORDING",
			filename: "t1_cmd_start_recording.wav",
		},
		{
			id:       "T1_CMD_NEGATION",
			desc:     "Command: Negation Rejection (US Female voice)",
			mode:     "command",
			text:     "do not stop recording",
			voice:    "af_heart",
			expected: "REJECT:NEGATION_DETECTED",
			filename: "t1_cmd_negation.wav",
		},
		{
			id:       "T1_DICT_BENCH",
			desc:     "Dictation: Technical phrase (British Female voice)",
			mode:     "dictation",
			text:     "The voice pipeline test bench is operational and ready.",
			voice:    "bf_emma",
			expected: "pipeline test bench",
			filename: "t1_dict_bench.wav",
		},
	}

	manifest := EvalManifest{
		Version: "1.0",
		Tier:    "tier-1-smoke",
		Items:   make([]EvalManifestItem, 0),
	}

	for _, c := range cases {
		log.Printf("Synthesizing [%s] with voice %s: %q...", c.id, c.voice, c.text)
		res, err := runner.Synthesize(engine.TTSRequest{
			Text:       c.text,
			Voice:      c.voice,
			Speed:      1.0,
			SampleRate: 24000,
		})
		if err != nil {
			log.Fatalf("synthesis failed for %s: %v", c.id, err)
		}

		// Resample Kokoro 24kHz audio to 16kHz for ASR
		resampled16k := audio.Resample(res.AudioSamples, 24000, 16000)

		// Add 200ms of leading silence and 1000ms of trailing silence for responsive endpointing
		leadSilence := make([]float32, 16000*200/1000)
		trailSilence := make([]float32, 16000*1000/1000) // 1.0s trailing silence > 0.72s endpoint
		padded := append(leadSilence, resampled16k...)
		padded = append(padded, trailSilence...)

		wavBytes, err := audio.EncodeWAV(padded, 16000)
		if err != nil {
			log.Fatalf("failed encoding WAV for %s: %v", c.id, err)
		}

		wavPath := filepath.Join(outDir, c.filename)
		if err := os.WriteFile(wavPath, wavBytes, 0644); err != nil {
			log.Fatalf("failed saving WAV to %s: %v", wavPath, err)
		}

		dur := float64(len(padded)) / 16000.0
		log.Printf("   -> Saved %s (%.2f s, %d bytes)", wavPath, dur, len(wavBytes))

		manifest.Items = append(manifest.Items, EvalManifestItem{
			ID:          c.id,
			Description: c.desc,
			Mode:        c.mode,
			Text:        c.text,
			Voice:       c.voice,
			Expected:    c.expected,
			WavFile:     c.filename,
			DurationSec: dur,
		})
	}

	// Case 5: Noise below threshold (-50 dBFS white noise + subtle periodic clicks)
	noiseSamples := make([]float32, 16000*2) // 2.0s
	for i := range noiseSamples {
		// Low-level white noise (~ -50 dBFS peak amp 0.003)
		v := float32((float64(i%997)/997.0 - 0.5) * 0.006)
		if i%3200 == 0 { // small typing click
			v += 0.015 * float32(math.Exp(-float64(i%160)/20.0))
		}
		noiseSamples[i] = v
	}
	noiseBytes, _ := audio.EncodeWAV(noiseSamples, 16000)
	noisePath := filepath.Join(outDir, "t1_noise_below_gate.wav")
	_ = os.WriteFile(noisePath, noiseBytes, 0644)
	manifest.Items = append(manifest.Items, EvalManifestItem{
		ID:          "T1_NOISE_GATE",
		Description: "Acoustic Noise: Low-level typing/room chatter below gate threshold",
		Mode:        "command",
		Text:        "",
		Voice:       "synthetic",
		Expected:    "DROPPED_BY_GATE",
		WavFile:     "t1_noise_below_gate.wav",
		DurationSec: 2.0,
	})
	log.Printf("   -> Saved %s (2.00 s, noise below gate)", noisePath)

	// Save manifest JSON
	manBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		log.Fatalf("failed marshaling manifest: %v", err)
	}
	manPath := filepath.Join(outDir, "manifest.json")
	if err := os.WriteFile(manPath, manBytes, 0644); err != nil {
		log.Fatalf("failed saving manifest: %v", err)
	}
	log.Printf("Successfully generated Tier-1 evaluation corpus in %s", outDir)
}
