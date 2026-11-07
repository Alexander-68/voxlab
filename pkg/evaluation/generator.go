package evaluation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"voxlab/pkg/audio"
	"voxlab/pkg/engine"
)

// BaseUtterance defines a unique Kokoro synthesis candidate.
type BaseUtterance struct {
	BaseID       string
	Description  string
	Mode         TestCaseMode
	Text         string
	Voice        string
	SpeakingRate float64
	Accent       string
	Gender       string
	Category     string
	Expected     string
}

// MatrixVariation defines a perturbation variant derived from a base utterance.
type MatrixVariation struct {
	Suffix      string
	Description string
	NoiseType   string
	SNR         *float64
	Expected    string // If empty, inherits BaseUtterance.Expected
}

// DefaultBaseUtterances returns the 16 curated base speech phrases spanning 12+ voices and 6 speaking rates.
func DefaultBaseUtterances() []BaseUtterance {
	return []BaseUtterance{
		{
			BaseID:       "CMD_SETTINGS_AF",
			Description:  "Open Settings (US Female, Normal)",
			Mode:         ModeCommand,
			Text:         "open settings",
			Voice:        "af_heart",
			SpeakingRate: 1.0,
			Accent:       "en-US",
			Gender:       "female",
			Category:     "command",
			Expected:     "NAV_SETTINGS",
		},
		{
			BaseID:       "CMD_SETTINGS_BM",
			Description:  "Open Settings (UK Male, Fast 1.2x)",
			Mode:         ModeCommand,
			Text:         "open settings",
			Voice:        "bm_george",
			SpeakingRate: 1.2,
			Accent:       "en-GB",
			Gender:       "male",
			Category:     "command",
			Expected:     "NAV_SETTINGS",
		},
		{
			BaseID:       "CMD_START_REC_AM",
			Description:  "Start Recording (US Male, Normal)",
			Mode:         ModeCommand,
			Text:         "start recording",
			Voice:        "am_adam",
			SpeakingRate: 1.0,
			Accent:       "en-US",
			Gender:       "male",
			Category:     "command",
			Expected:     "START_RECORDING",
		},
		{
			BaseID:       "CMD_START_REC_BF",
			Description:  "Start Recording (UK Female, Brisk 1.3x)",
			Mode:         ModeCommand,
			Text:         "start recording now",
			Voice:        "bf_emma",
			SpeakingRate: 1.3,
			Accent:       "en-GB",
			Gender:       "female",
			Category:     "command",
			Expected:     "START_RECORDING",
		},
		{
			BaseID:       "CMD_STOP_REC_AF",
			Description:  "Stop Recording (US Female)",
			Mode:         ModeCommand,
			Text:         "stop recording",
			Voice:        "af_bella",
			SpeakingRate: 1.0,
			Accent:       "en-US",
			Gender:       "female",
			Category:     "command",
			Expected:     "STOP_RECORDING",
		},
		{
			BaseID:       "CMD_DIM_BM",
			Description:  "Dim Screen (UK Male Narrator)",
			Mode:         ModeCommand,
			Text:         "dim screen",
			Voice:        "bm_lewis",
			SpeakingRate: 1.0,
			Accent:       "en-GB",
			Gender:       "male",
			Category:     "command",
			Expected:     "DISPLAY_DARK",
		},
		{
			BaseID:       "CMD_SCOPE_AF",
			Description:  "Show Endoscope on Main Monitor (US Female)",
			Mode:         ModeCommand,
			Text:         "show endoscope on the main monitor",
			Voice:        "af_nicole",
			SpeakingRate: 1.0,
			Accent:       "en-US",
			Gender:       "female",
			Category:     "command",
			Expected:     "ROUTE_VIDEO",
		},
		{
			BaseID:       "CMD_PREF_AM",
			Description:  "Open System Preferences (US Male, Deliberate 0.8x)",
			Mode:         ModeCommand,
			Text:         "open system preferences",
			Voice:        "am_adam",
			SpeakingRate: 0.8,
			Accent:       "en-US",
			Gender:       "male",
			Category:     "command",
			Expected:     "NAV_SETTINGS",
		},
		{
			BaseID:       "DICT_BENCH_BF",
			Description:  "Dictation: Voice Test Bench (UK Female)",
			Mode:         ModeDictation,
			Text:         "The voice pipeline test bench is operational and ready.",
			Voice:        "bf_emma",
			SpeakingRate: 1.0,
			Accent:       "en-GB",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "pipeline test bench",
		},
		{
			BaseID:       "DICT_VITALS_AM",
			Description:  "Dictation: Patient Vitals (US Male)",
			Mode:         ModeDictation,
			Text:         "Patient vital signs are stable with blood pressure one twenty over eighty.",
			Voice:        "am_michael",
			SpeakingRate: 1.0,
			Accent:       "en-US",
			Gender:       "male",
			Category:     "dictation",
			Expected:     "blood pressure",
		},
		{
			BaseID:       "DICT_INCISION_AF",
			Description:  "Dictation: Surgical Findings (US Female, 1.1x)",
			Mode:         ModeDictation,
			Text:         "Incision site clean with no signs of inflammation or erythema.",
			Voice:        "af_sky",
			SpeakingRate: 1.1,
			Accent:       "en-US",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "incision site",
		},
		{
			BaseID:       "DICT_MED_BF",
			Description:  "Dictation: Medication Dosage (UK Female, 0.9x)",
			Mode:         ModeDictation,
			Text:         "Administer two milligrams of midazolam intravenously.",
			Voice:        "bf_isabella",
			SpeakingRate: 0.9,
			Accent:       "en-GB",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "intravenously",
		},
		{
			BaseID:       "NEG_STOP_AF",
			Description:  "Negation: Do Not Stop Recording (US Female)",
			Mode:         ModeCommand,
			Text:         "do not stop recording",
			Voice:        "af_heart",
			SpeakingRate: 1.0,
			Accent:       "en-US",
			Gender:       "female",
			Category:     "negation",
			Expected:     "REJECT:NEGATION_DETECTED",
		},
		{
			BaseID:       "NEG_SETTINGS_AM",
			Description:  "Negation: Do Not Open Settings (US Male, 1.1x)",
			Mode:         ModeCommand,
			Text:         "do not open settings during procedure",
			Voice:        "am_adam",
			SpeakingRate: 1.1,
			Accent:       "en-US",
			Gender:       "male",
			Category:     "negation",
			Expected:     "REJECT:NEGATION_DETECTED",
		},
		{
			BaseID:       "DIST_LUNCH_AM",
			Description:  "Distractor: Lunch Time Chatter (US Male)",
			Mode:         ModeCommand,
			Text:         "what time is lunch scheduled for today",
			Voice:        "am_echo",
			SpeakingRate: 1.0,
			Accent:       "en-US",
			Gender:       "male",
			Category:     "distractor",
			Expected:     "REJECT:OUT_OF_DOMAIN",
		},
		{
			BaseID:       "CMD_ES_EF",
			Description:  "Spanish Accented Command: Open Settings",
			Mode:         ModeCommand,
			Text:         "open settings",
			Voice:        "ef_dora",
			SpeakingRate: 1.0,
			Accent:       "es",
			Gender:       "female",
			Category:     "command",
			Expected:     "NAV_SETTINGS",
		},
	}
}

func ptrFloat(f float64) *float64 { return &f }

// BuildTier2MatrixDefinitions specifies the 66 test combinations across the SNR ladder and perturbations.
func BuildTier2MatrixDefinitions() ([]BaseUtterance, map[string][]MatrixVariation) {
	bases := DefaultBaseUtterances()
	variations := make(map[string][]MatrixVariation)

	// Full SNR Ladder variations (+20, +15, +10, +5, 0, -5 dB)
	fullSNRLadder := []MatrixVariation{
		{Suffix: "SNR20", Description: "White Noise SNR +20 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(20.0)},
		{Suffix: "SNR15_PINK", Description: "Pink Noise SNR +15 dB", NoiseType: audio.NoisePink, SNR: ptrFloat(15.0)},
		{Suffix: "SNR10", Description: "White Noise SNR +10 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(10.0)},
		{Suffix: "SNR05", Description: "White Noise SNR +5 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(5.0)},
		{Suffix: "SNR00", Description: "White Noise SNR 0 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(0.0)},
		{Suffix: "SNRM05", Description: "White Noise SNR -5 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(-5.0)},
	}

	// Mid SNR variations (+20, +10, +5, 0 dB)
	midSNRLadder := []MatrixVariation{
		{Suffix: "SNR20", Description: "White Noise SNR +20 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(20.0)},
		{Suffix: "SNR10", Description: "White Noise SNR +10 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(10.0)},
		{Suffix: "SNR05", Description: "White Noise SNR +5 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(5.0)},
		{Suffix: "SNR00", Description: "White Noise SNR 0 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(0.0)},
	}

	// Light SNR variations (+20, +10, +5 dB)
	lightSNRLadder := []MatrixVariation{
		{Suffix: "SNR20", Description: "White Noise SNR +20 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(20.0)},
		{Suffix: "SNR10", Description: "White Noise SNR +10 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(10.0)},
		{Suffix: "SNR05", Description: "White Noise SNR +5 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(5.0)},
	}

	// Assign variations to representative base utterances
	variations["CMD_SETTINGS_AF"] = append(fullSNRLadder,
		MatrixVariation{Suffix: "TYPING_15", Description: "Typing Clacks SNR +15 dB", NoiseType: audio.NoiseTyping, SNR: ptrFloat(15.0)},
		MatrixVariation{Suffix: "TYPING_05", Description: "Typing Clacks SNR +5 dB", NoiseType: audio.NoiseTyping, SNR: ptrFloat(5.0)},
		MatrixVariation{Suffix: "BABBLE_15", Description: "Room Babble SNR +15 dB", NoiseType: audio.NoiseBabble, SNR: ptrFloat(15.0)},
		MatrixVariation{Suffix: "BABBLE_05", Description: "Room Babble SNR +5 dB", NoiseType: audio.NoiseBabble, SNR: ptrFloat(5.0)},
		MatrixVariation{Suffix: "REVERB", Description: "Room Reverberation (RT60 ~0.4s)", NoiseType: audio.NoiseReverb, SNR: ptrFloat(15.0)},
	)

	variations["CMD_START_REC_AM"] = append(fullSNRLadder,
		MatrixVariation{Suffix: "TYPING_15", Description: "Typing Clacks SNR +15 dB", NoiseType: audio.NoiseTyping, SNR: ptrFloat(15.0)},
		MatrixVariation{Suffix: "TYPING_05", Description: "Typing Clacks SNR +5 dB", NoiseType: audio.NoiseTyping, SNR: ptrFloat(5.0)},
		MatrixVariation{Suffix: "BABBLE_15", Description: "Room Babble SNR +15 dB", NoiseType: audio.NoiseBabble, SNR: ptrFloat(15.0)},
		MatrixVariation{Suffix: "BABBLE_05", Description: "Room Babble SNR +5 dB", NoiseType: audio.NoiseBabble, SNR: ptrFloat(5.0)},
		MatrixVariation{Suffix: "REVERB", Description: "Room Reverberation (RT60 ~0.4s)", NoiseType: audio.NoiseReverb, SNR: ptrFloat(15.0)},
	)

	variations["DICT_BENCH_BF"] = append(fullSNRLadder,
		MatrixVariation{Suffix: "TYPING_15", Description: "Typing Clacks SNR +15 dB", NoiseType: audio.NoiseTyping, SNR: ptrFloat(15.0)},
		MatrixVariation{Suffix: "TYPING_05", Description: "Typing Clacks SNR +5 dB", NoiseType: audio.NoiseTyping, SNR: ptrFloat(5.0)},
		MatrixVariation{Suffix: "BABBLE_15", Description: "Room Babble SNR +15 dB", NoiseType: audio.NoiseBabble, SNR: ptrFloat(15.0)},
		MatrixVariation{Suffix: "BABBLE_05", Description: "Room Babble SNR +5 dB", NoiseType: audio.NoiseBabble, SNR: ptrFloat(5.0)},
		MatrixVariation{Suffix: "REVERB", Description: "Room Reverberation (RT60 ~0.4s)", NoiseType: audio.NoiseReverb, SNR: ptrFloat(15.0)},
	)

	variations["DICT_VITALS_AM"] = append(midSNRLadder,
		MatrixVariation{Suffix: "TYPING_15", Description: "Typing Clacks SNR +15 dB", NoiseType: audio.NoiseTyping, SNR: ptrFloat(15.0)},
		MatrixVariation{Suffix: "TYPING_05", Description: "Typing Clacks SNR +5 dB", NoiseType: audio.NoiseTyping, SNR: ptrFloat(5.0)},
		MatrixVariation{Suffix: "REVERB", Description: "Room Reverberation (RT60 ~0.4s)", NoiseType: audio.NoiseReverb, SNR: ptrFloat(15.0)},
	)

	variations["DICT_INCISION_AF"] = []MatrixVariation{
		{Suffix: "BABBLE_15", Description: "Room Babble SNR +15 dB", NoiseType: audio.NoiseBabble, SNR: ptrFloat(15.0)},
		{Suffix: "BABBLE_05", Description: "Room Babble SNR +5 dB", NoiseType: audio.NoiseBabble, SNR: ptrFloat(5.0)},
	}

	variations["NEG_STOP_AF"] = lightSNRLadder
	variations["CMD_ES_EF"] = lightSNRLadder

	return bases, variations
}

// GenerateTier2Corpus synthesizes base audio clips and creates the full acoustic degradation matrix.
func GenerateTier2Corpus(outDir string, runner engine.SpeechEngine) (*EvalManifest, error) {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("failed creating tier2 output directory: %w", err)
	}

	bases, variationsMap := BuildTier2MatrixDefinitions()

	manifest := &EvalManifest{
		Version: "2.0",
		Tier:    "tier-2-deep-matrix",
		Items:   make([]EvalItem, 0),
	}

	// Step 1: Synthesize base WAV files
	baseSamples := make(map[string][]float32)

	log.Printf("[Tier2 Generator] Synthesizing %d base utterances across Kokoro voices...", len(bases))
	for _, b := range bases {
		baseWavFilename := fmt.Sprintf("base_%s.wav", strings.ToLower(b.BaseID))
		baseWavPath := filepath.Join(outDir, baseWavFilename)

		var samples16k []float32
		// If base WAV exists on disk, reuse it to avoid re-synthesis overhead
		if wavData, err := os.ReadFile(baseWavPath); err == nil {
			if s, rate, err := audio.DecodeWAV(bytes.NewReader(wavData)); err == nil && len(s) > 0 {
				if rate != 16000 {
					s = audio.Resample(s, rate, 16000)
				}
				samples16k = s
			}
		}

		if len(samples16k) == 0 {
			if runner == nil {
				return nil, fmt.Errorf("speech runner required to synthesize base utterance %s", b.BaseID)
			}
			log.Printf("   -> Synthesizing base [%s] voice=%s rate=%.1f: %q", b.BaseID, b.Voice, b.SpeakingRate, b.Text)
			res, err := runner.Synthesize(engine.TTSRequest{
				Text:       b.Text,
				Voice:      b.Voice,
				Speed:      b.SpeakingRate,
				SampleRate: 24000,
			})
			if err != nil {
				return nil, fmt.Errorf("synthesis failed for %s: %w", b.BaseID, err)
			}

			resampled := audio.Resample(res.AudioSamples, 24000, 16000)
			leadSilence := make([]float32, 16000*200/1000)   // 200ms
			trailSilence := make([]float32, 16000*1000/1000) // 1000ms
			samples16k = append(leadSilence, resampled...)
			samples16k = append(samples16k, trailSilence...)

			wavBytes, err := audio.EncodeWAV(samples16k, 16000)
			if err != nil {
				return nil, fmt.Errorf("failed encoding base wav %s: %w", b.BaseID, err)
			}
			if err := os.WriteFile(baseWavPath, wavBytes, 0644); err != nil {
				return nil, fmt.Errorf("failed writing base wav %s: %w", b.BaseID, err)
			}
		}

		baseSamples[b.BaseID] = samples16k

		// Add clean base item to manifest
		dur := float64(len(samples16k)) / 16000.0
		manifest.Items = append(manifest.Items, EvalItem{
			ID:           fmt.Sprintf("T2_CLEAN_%s", b.BaseID),
			Description:  fmt.Sprintf("[Clean] %s", b.Description),
			Mode:         b.Mode,
			Text:         b.Text,
			Voice:        b.Voice,
			SpeakingRate: b.SpeakingRate,
			Accent:       b.Accent,
			Category:     b.Category,
			NoiseType:    audio.NoiseClean,
			Expected:     b.Expected,
			WavFile:      baseWavFilename,
			DurationSec:  dur,
		})
	}

	// Step 2: Generate acoustic perturbations (SNR ladder, typing, babble, reverb)
	seed := int64(1001)
	for _, b := range bases {
		cleanSamples := baseSamples[b.BaseID]
		vars, hasVars := variationsMap[b.BaseID]
		if !hasVars {
			continue
		}

		for _, v := range vars {
			seed++
			perturbedSamples := audio.AddNoiseAtSNR(cleanSamples, v.NoiseType, *v.SNR, 16000, seed)
			filename := fmt.Sprintf("t2_%s_%s.wav", strings.ToLower(b.BaseID), strings.ToLower(v.Suffix))
			wavPath := filepath.Join(outDir, filename)

			wavBytes, err := audio.EncodeWAV(perturbedSamples, 16000)
			if err != nil {
				return nil, fmt.Errorf("failed encoding perturbed wav %s: %w", filename, err)
			}
			if err := os.WriteFile(wavPath, wavBytes, 0644); err != nil {
				return nil, fmt.Errorf("failed saving perturbed wav %s: %w", filename, err)
			}

			exp := b.Expected
			if v.Expected != "" {
				exp = v.Expected
			}
			dur := float64(len(perturbedSamples)) / 16000.0
			manifest.Items = append(manifest.Items, EvalItem{
				ID:           fmt.Sprintf("T2_%s_%s", b.BaseID, v.Suffix),
				Description:  fmt.Sprintf("[%s] %s (%s)", v.Description, b.Description, b.Voice),
				Mode:         b.Mode,
				Text:         b.Text,
				Voice:        b.Voice,
				SpeakingRate: b.SpeakingRate,
				Accent:       b.Accent,
				Category:     b.Category,
				SNR:          v.SNR,
				NoiseType:    v.NoiseType,
				Expected:     exp,
				WavFile:      filename,
				DurationSec:  dur,
			})
		}
	}

	// Step 3: Add sub-gate ambient noise clips (DROPPED_BY_GATE)
	// Sub-gate 1: Ambient room hiss (-48 dBFS)
	hissSamples := audio.GenerateWhiteNoise(16000*2, 777)
	for i := range hissSamples {
		hissSamples[i] *= 0.003 // ~ -48 dBFS
	}
	hissBytes, _ := audio.EncodeWAV(hissSamples, 16000)
	hissPath := filepath.Join(outDir, "t2_subgate_hiss.wav")
	_ = os.WriteFile(hissPath, hissBytes, 0644)
	manifest.Items = append(manifest.Items, EvalItem{
		ID:          "T2_SUBGATE_HISS",
		Description: "[Sub-Gate] Low-level ambient room hiss (-48 dBFS)",
		Mode:        ModeCommand,
		Text:        "",
		Voice:       "synthetic",
		NoiseType:   audio.NoiseWhite,
		Category:    "noise_gate",
		Expected:    "DROPPED_BY_GATE",
		WavFile:     "t2_subgate_hiss.wav",
		DurationSec: 2.0,
	})

	// Sub-gate 2: Light keyboard clicks below -42 dBFS gate
	clicksSamples := audio.GenerateTypingNoise(16000*2, 16000, 888)
	for i := range clicksSamples {
		clicksSamples[i] *= 0.012 // Low peak amp
	}
	clicksBytes, _ := audio.EncodeWAV(clicksSamples, 16000)
	clicksPath := filepath.Join(outDir, "t2_subgate_clicks.wav")
	_ = os.WriteFile(clicksPath, clicksBytes, 0644)
	manifest.Items = append(manifest.Items, EvalItem{
		ID:          "T2_SUBGATE_CLICKS",
		Description: "[Sub-Gate] Distant keyboard typing clicks below gate threshold",
		Mode:        ModeCommand,
		Text:        "",
		Voice:       "synthetic",
		NoiseType:   audio.NoiseTyping,
		Category:    "noise_gate",
		Expected:    "DROPPED_BY_GATE",
		WavFile:     "t2_subgate_clicks.wav",
		DurationSec: 2.0,
	})

	// Save manifest JSON
	manBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed marshaling tier2 manifest: %w", err)
	}
	manPath := filepath.Join(outDir, "manifest.json")
	if err := os.WriteFile(manPath, manBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed saving tier2 manifest JSON: %w", err)
	}

	log.Printf("[Tier2 Generator] Successfully created %d test items in %s", len(manifest.Items), outDir)
	return manifest, nil
}


