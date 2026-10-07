package engine

import (
	"strings"
	"testing"
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
		sid := mapVoiceToSID(tc.voice, true)
		if sid != tc.expectedSID {
			t.Errorf("for voice %q expected sid %d, got %d", tc.voice, tc.expectedSID, sid)
		}
		// SIDs 5, 6, 9, 10 are male in v0.19
		maleSIDs := map[int]bool{5: true, 6: true, 9: true, 10: true}
		if tc.isMale && !maleSIDs[sid] {
			t.Errorf("for male voice %q expected male SID (5,6,9,10), got female SID %d", tc.voice, sid)
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
		sid := mapVoiceToSID(tc.voice, false)
		if sid != tc.expectedSID {
			t.Errorf("for v1.1 voice %q expected sid %d, got %d", tc.voice, tc.expectedSID, sid)
		}
	}
}

func TestFindKokoroModelDir(t *testing.T) {
	// When empty string is passed, it should discover and prefer kokoro-multi-lang-v1_1
	modelDir, found := findKokoroModelDir("")
	if found {
		if !strings.Contains(modelDir, "kokoro-multi-lang-v1_1") {
			t.Errorf("expected kokoro-multi-lang-v1_1 to be preferred candidate, got: %s", modelDir)
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
