package evaluation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// TestCaseMode specifies whether audio is evaluated as a command or dictation.
type TestCaseMode string

const (
	ModeCommand   TestCaseMode = "command"
	ModeDictation TestCaseMode = "dictation"
)

// EvalItem matches manifest item structure.
type EvalItem struct {
	ID           string       `json:"id"`
	Description  string       `json:"description"`
	Mode         TestCaseMode `json:"mode"`
	Text         string       `json:"text"`
	Voice        string       `json:"voice"`
	SpeakingRate float64      `json:"speaking_rate,omitempty"`
	SNR          *float64     `json:"snr_db,omitempty"`
	NoiseType    string       `json:"noise_type,omitempty"`
	Accent       string       `json:"accent,omitempty"`
	Category     string       `json:"category,omitempty"` // "command", "dictation", "negation", "distractor", "noise_gate"
	Expected     string       `json:"expected"`
	WavFile      string       `json:"wav_file"`
	DurationSec  float64      `json:"duration_sec"`
}

// EvalManifest represents a suite of test items.
type EvalManifest struct {
	Version string     `json:"version"`
	Tier    string     `json:"tier"`
	Items   []EvalItem `json:"items"`
}

// LoadManifest reads an evaluation manifest from disk.
func LoadManifest(path string) (*EvalManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed reading manifest at %s: %w", path, err)
	}
	var m EvalManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed unmarshaling manifest JSON: %w", err)
	}
	return &m, nil
}

// ResolveWavPath gets the full path to a test WAV file relative to a manifest directory.
func ResolveWavPath(manifestDir string, item EvalItem) string {
	return filepath.Join(manifestDir, item.WavFile)
}

// CalculateWER computes Word Error Rate between a reference string and hypothesis string.
func CalculateWER(reference, hypothesis string) float64 {
	refWords := normalizeAndSplitWords(reference)
	hypWords := normalizeAndSplitWords(hypothesis)

	if len(refWords) == 0 {
		if len(hypWords) == 0 {
			return 0.0
		}
		return 1.0
	}

	d := make([][]int, len(refWords)+1)
	for i := range d {
		d[i] = make([]int, len(hypWords)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}

	for i := 1; i <= len(refWords); i++ {
		for j := 1; j <= len(hypWords); j++ {
			cost := 1
			if refWords[i-1] == hypWords[j-1] {
				cost = 0
			}
			d[i][j] = min(
				d[i-1][j]+1,      // deletion
				d[i][j-1]+1,      // insertion
				d[i-1][j-1]+cost, // substitution
			)
		}
	}

	return float64(d[len(refWords)][len(hypWords)]) / float64(len(refWords))
}

// CalculateCER computes Character Error Rate between a reference string and hypothesis string.
func CalculateCER(reference, hypothesis string) float64 {
	refRunes := normalizeAndFilterRunes(reference)
	hypRunes := normalizeAndFilterRunes(hypothesis)

	if len(refRunes) == 0 {
		if len(hypRunes) == 0 {
			return 0.0
		}
		return 1.0
	}

	d := make([][]int, len(refRunes)+1)
	for i := range d {
		d[i] = make([]int, len(hypRunes)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}

	for i := 1; i <= len(refRunes); i++ {
		for j := 1; j <= len(hypRunes); j++ {
			cost := 1
			if refRunes[i-1] == hypRunes[j-1] {
				cost = 0
			}
			d[i][j] = min(
				d[i-1][j]+1,      // deletion
				d[i][j-1]+1,      // insertion
				d[i-1][j-1]+cost, // substitution
			)
		}
	}

	return float64(d[len(refRunes)][len(hypRunes)]) / float64(len(refRunes))
}

func normalizeAndSplitWords(s string) []string {
	var clean strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			clean.WriteRune(r)
		}
	}
	return strings.Fields(clean.String())
}

func normalizeAndFilterRunes(s string) []rune {
	var out []rune
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out = append(out, r)
		}
	}
	return out
}
