package corpus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"voxlab/pkg/evaluation/metrics"
)

// TestCaseMode specifies whether audio is evaluated as a command or dictation.
type TestCaseMode = metrics.TestCaseMode

const (
	ModeCommand   = metrics.ModeCommand
	ModeDictation = metrics.ModeDictation
)

// EvalItem matches manifest item structure.
type EvalItem struct {
	ID           string       `json:"id"`
	Description  string       `json:"description"`
	Mode         TestCaseMode `json:"mode"`
	Text         string       `json:"text"`
	AltText      string       `json:"alt_text,omitempty"` // same words in written form ("2 mg"), scored as equally correct
	Voice        string       `json:"voice"`
	SpeakingRate float64      `json:"speaking_rate,omitempty"`
	SNR          *float64     `json:"snr_db,omitempty"`
	NoiseType    string       `json:"noise_type,omitempty"`
	Accent       string       `json:"accent,omitempty"`
	Category     string       `json:"category,omitempty"` // "command", "dictation", "negation", "distractor", "noise_gate"
	Expected     string       `json:"expected"`
	MaxErrorRate *float64     `json:"max_error_rate,omitempty"` // dictation pass limit; DefaultMaxErrorRate if unset
	WavFile      string       `json:"wav_file"`
	DurationSec  float64      `json:"duration_sec"`
}

// DefaultMaxErrorRate is the highest error rate at which a dictation item passes.
const DefaultMaxErrorRate = 0.35

// ScoreDictation grades a draft on one error rate: CER for CJK references, which have no word
// boundaries, otherwise WER. A key term in the draft is reported but cannot rescue a bad transcript.
func ScoreDictation(item EvalItem, draft string) (errorRate float64, passed, keyTermHit bool) {
	ref := BestReference(item, draft)
	errorRate = metrics.CalculateWER(ref, draft)
	if strings.ContainsFunc(ref, metrics.IsCJK) {
		errorRate = metrics.CalculateCER(ref, draft)
	}
	limit := DefaultMaxErrorRate
	if item.MaxErrorRate != nil {
		limit = *item.MaxErrorRate
	}
	keyTermHit = item.Expected != "" && strings.Contains(strings.ToLower(draft), strings.ToLower(item.Expected))
	return errorRate, draft != "" && errorRate <= limit, keyTermHit
}

// BestReference picks the accepted wording closest to the hypothesis. Models write numbers and units
// their own way, and "2 mg" for "two milligrams" is not a recognition error.
func BestReference(item EvalItem, hypothesis string) string {
	if item.AltText != "" && metrics.CalculateWER(item.AltText, hypothesis) < metrics.CalculateWER(item.Text, hypothesis) {
		return item.AltText
	}
	return item.Text
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
