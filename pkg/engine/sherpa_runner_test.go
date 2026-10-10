package engine

import (
	"strings"
	"testing"

	"voxlab/pkg/config"
)

func TestMapVoiceToSID(t *testing.T) {
	tests := []struct {
		voice       string
		expectedSID int
		isMale      bool
	}{
		{"af", 0, false},
		{"af_heart", 0, false},
		{"af_bella", 1, false},
		{"af_nicole", 2, false},
		{"af_sarah", 3, false},
		{"af_sky", 4, false},
		{"am_adam", 5, true},
		{"am_michael", 6, true},
		{"bf_emma", 7, false},
		{"bf_isabella", 8, false},
		{"bm_george", 9, true},
		{"bm_lewis", 10, true},
		// Aliases and heuristics
		{"am_fenrir", 9, true},
		{"am_puck", 10, true},
		{"random_male", 5, true},
	}

	for _, tc := range tests {
		sid := mapVoiceToSID(tc.voice, "v0_19")
		if sid != tc.expectedSID {
			t.Errorf("for voice %q expected sid %d, got %d", tc.voice, tc.expectedSID, sid)
		}
		// SIDs 5, 6, 9, 10 are male in v0.19
		maleSIDs := map[int]bool{5: true, 6: true, 9: true, 10: true}
		if tc.isMale && !maleSIDs[sid] {
			t.Errorf("for male voice %q expected male SID (5,6,9,10), got female SID %d", tc.voice, sid)
		}
	}

	v10Tests := []struct {
		voice       string
		expectedSID int
		isMale      bool
	}{
		{"af_alloy", 0, false},
		{"af_aoede", 1, false},
		{"af_bella", 2, false},
		{"af_heart", 3, false},
		{"heart", 3, false},
		{"af", 3, false},
		{"am_adam", 11, true},
		{"bf_emma", 21, false},
		{"bm_daniel", 24, true},
		{"ef_dora", 28, false},
		{"em_alex", 29, true},
		{"ff_siwis", 30, false},
		{"hf_alpha", 31, false},
		{"hm_omega", 33, true},
		{"if_sara", 35, false},
		{"im_nicola", 36, true},
		{"jf_alpha", 37, false},
		{"jm_kumo", 41, true},
		{"pf_dora", 42, false},
		{"pm_santa", 44, true},
		{"zf_xiaobei", 45, false},
		{"zm_yunjian", 49, true},
		{"em_santa", 53, true},
		{"santa", 53, true},
		{"53", 53, true},
	}
	for _, tc := range v10Tests {
		sid := mapVoiceToSID(tc.voice, "v1_0")
		if sid != tc.expectedSID {
			t.Errorf("for v1.0 voice %q expected sid %d, got %d", tc.voice, tc.expectedSID, sid)
		}
	}

	v11Tests := []struct {
		voice       string
		expectedSID int
		isMale      bool
	}{
		{"af_maple", 0, false},
		{"af_sol", 1, false},
		{"bf_vale", 2, false},
		{"zf_001", 3, false},
		{"zm_009", 58, true},
		{"zm_100", 102, true},
	}
	for _, tc := range v11Tests {
		sid := mapVoiceToSID(tc.voice, "v1_1")
		if sid != tc.expectedSID {
			t.Errorf("for v1.1 voice %q expected sid %d, got %d", tc.voice, tc.expectedSID, sid)
		}
	}
}

func TestFindKokoroModelDir(t *testing.T) {
	// When empty string is passed, it should discover and prefer kokoro-multi-lang-v1_0
	modelDir, found := findKokoroModelDir("")
	if found {
		if !strings.Contains(modelDir, "kokoro-multi-lang-v1_0") {
			t.Errorf("expected kokoro-multi-lang-v1_0 to be preferred candidate, got: %s", modelDir)
		}
	}

	// When an explicit valid directory is passed, it must be honored
	explicitDir, explicitFound := findKokoroModelDir("models/kokoro-en-v0_19")
	if explicitFound {
		if !strings.Contains(explicitDir, "kokoro-en-v0_19") {
			t.Errorf("expected explicit model dir to be retained, got: %s", explicitDir)
		}
	}

	// Test FindInstalledKokoroModels
	installed := FindInstalledKokoroModels("models")
	if len(installed) == 0 {
		t.Errorf("expected installed models to be found in models/ directory")
	}
}

func TestSynthesizeChinese(t *testing.T) {
	cfg := &config.AppConfig{}
	cfg.Engine.KokoroModelDir = "models/kokoro-multi-lang-v1_1"
	cfg.Engine.SherpaTtsBin = "sherpa-onnx-offline-tts"
	runner := NewSherpaRunner(cfg)
	res, err := runner.Synthesize(TTSRequest{
		Text:  "你好，世界。",
		Voice: "zf_001",
		Speed: 1.0,
	})
	if err != nil {
		t.Fatalf("Synthesize failed: %v", err)
	}
	if res.DurationSec < 0.8 {
		t.Errorf("expected synthesized audio duration > 0.8s for '你好，世界。', got %.2fs", res.DurationSec)
	}
}

func TestSynthesizeKokoroV10(t *testing.T) {
	cfg := &config.AppConfig{}
	cfg.Engine.KokoroModelDir = "models/kokoro-multi-lang-v1_0"
	cfg.Engine.SherpaTtsBin = "sherpa-onnx-offline-tts"
	runner := NewSherpaRunner(cfg)

	// Test heart voice (sid 3)
	res, err := runner.Synthesize(TTSRequest{
		Text:  "Testing Kokoro v1.0 voice model.",
		Voice: "af_heart",
		Speed: 1.0,
	})
	if err != nil {
		t.Fatalf("Synthesize failed with af_heart: %v", err)
	}
	if res.DurationSec < 0.8 {
		t.Errorf("expected synthesized audio duration > 0.8s, got %.2fs", res.DurationSec)
	}

	// Test Spanish voice em_santa (sid 53)
	resEs, err := runner.Synthesize(TTSRequest{
		Text:  "Hola mundo, probando la voz en español.",
		Voice: "em_santa",
		Speed: 1.0,
	})
	if err != nil {
		t.Fatalf("Synthesize failed with em_santa: %v", err)
	}
	if resEs.DurationSec < 0.8 {
		t.Errorf("expected synthesized audio duration > 0.8s, got %.2fs", resEs.DurationSec)
	}
}

func TestKokoroFP16Model(t *testing.T) {
	// 1. Test model list contains FP16 entry
	installed := FindInstalledKokoroModels("models")
	hasFP16 := false
	for _, m := range installed {
		if m == "kokoro-multi-lang-v1_0 (FP16)" {
			hasFP16 = true
			break
		}
	}
	if !hasFP16 {
		t.Errorf("expected 'kokoro-multi-lang-v1_0 (FP16)' in installed models, got: %v", installed)
	}

	// 2. Test SetTTSModel with FP16
	cfg := &config.AppConfig{}
	cfg.Engine.KokoroModelDir = "models/kokoro-multi-lang-v1_1"
	cfg.Engine.SherpaTtsBin = "sherpa-onnx-offline-tts"
	runner := NewSherpaRunner(cfg)

	if err := runner.SetTTSModel("kokoro-multi-lang-v1_0 (FP16)"); err != nil {
		t.Fatalf("failed to set model to FP16: %v", err)
	}

	modelInfo, isNeural := runner.TTSModelInfo()
	if !strings.Contains(modelInfo, "FP16") {
		t.Errorf("expected model info to contain FP16, got: %s", modelInfo)
	}
	if !isNeural {
		t.Errorf("expected neural to be true for FP16 model")
	}

	// 3. Test Synthesize with FP16
	res, err := runner.Synthesize(TTSRequest{
		Text:  "Testing Kokoro v1.0 FP16 half-precision model.",
		Voice: "af_heart",
		Speed: 1.0,
	})
	if err != nil {
		t.Fatalf("Synthesize failed with FP16 model: %v", err)
	}
	if res.DurationSec < 0.8 {
		t.Errorf("expected duration > 0.8s, got %.2fs", res.DurationSec)
	}
}

func TestKokoroINT8Model(t *testing.T) {
	// 1. Test model list contains INT8 entry
	installed := FindInstalledKokoroModels("models")
	hasINT8 := false
	for _, m := range installed {
		if m == "kokoro-multi-lang-v1_1 (INT8)" {
			hasINT8 = true
			break
		}
	}
	if !hasINT8 {
		t.Errorf("expected 'kokoro-multi-lang-v1_1 (INT8)' in installed models, got: %v", installed)
	}

	// 2. Test SetTTSModel with INT8
	cfg := &config.AppConfig{}
	cfg.Engine.KokoroModelDir = "models/kokoro-multi-lang-v1_1"
	cfg.Engine.SherpaTtsBin = "sherpa-onnx-offline-tts"
	runner := NewSherpaRunner(cfg)

	if err := runner.SetTTSModel("kokoro-multi-lang-v1_1 (INT8)"); err != nil {
		t.Fatalf("failed to set model to INT8: %v", err)
	}

	modelInfo, isNeural := runner.TTSModelInfo()
	if !strings.Contains(modelInfo, "INT8") {
		t.Errorf("expected model info to contain INT8, got: %s", modelInfo)
	}
	if !isNeural {
		t.Errorf("expected neural to be true for INT8 model")
	}

	// 3. Test Synthesize with INT8
	res, err := runner.Synthesize(TTSRequest{
		Text:  "Testing Kokoro v1.1 INT8 quantized model.",
		Voice: "af_maple",
		Speed: 1.0,
	})
	if err != nil {
		t.Fatalf("Synthesize failed with INT8 model: %v", err)
	}
	if res.DurationSec < 0.8 {
		t.Errorf("expected duration > 0.8s, got %.2fs", res.DurationSec)
	}
}
