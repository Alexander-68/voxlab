package evaluation

import (
	"testing"
	"time"
)

func TestScoreDictation(t *testing.T) {
	exact := 0.0
	tests := []struct {
		name       string
		item       EvalItem
		draft      string
		wantPass   bool
		wantKeyHit bool
	}{
		{
			name:       "exact transcript",
			item:       EvalItem{Text: "Blood pressure is stable.", Expected: "blood pressure"},
			draft:      "blood pressure is stable",
			wantPass:   true,
			wantKeyHit: true,
		},
		{
			// Real output that used to pass at WER 1.00 on the substring alone.
			name:       "key term does not rescue a wrong transcript",
			item:       EvalItem{Text: "Somministrati due milligrammi di midazolam per via endovenosa", Expected: "midazolam"},
			draft:      "Sa ministre do Milligrammi de midazolam per Vire and overnose",
			wantPass:   false,
			wantKeyHit: true,
		},
		{
			name:     "low CER does not rescue a high WER",
			item:     EvalItem{Text: "the the the the"},
			draft:    "thee thee thee thee",
			wantPass: false,
		},
		{
			name:       "CJK uses characters",
			item:       EvalItem{Text: "生命体征平稳", Expected: "生命体征"},
			draft:      "生命体征平稳",
			wantPass:   true,
			wantKeyHit: true,
		},
		{
			name:     "CJK beyond the limit fails",
			item:     EvalItem{Text: "生命体征平稳"},
			draft:    "生命",
			wantPass: false,
		},
		{
			name:     "empty draft fails",
			item:     EvalItem{Text: "open settings"},
			draft:    "",
			wantPass: false,
		},
		{
			name:     "item limit overrides the default",
			item:     EvalItem{Text: "one two three four five", MaxErrorRate: &exact},
			draft:    "one two three four six",
			wantPass: false,
		},
		{
			name:     "default limit allows the same draft",
			item:     EvalItem{Text: "one two three four five"},
			draft:    "one two three four six",
			wantPass: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rate, passed, keyHit := ScoreDictation(tc.item, tc.draft)
			if passed != tc.wantPass || keyHit != tc.wantKeyHit {
				t.Fatalf("rate %.2f: passed=%v keyHit=%v, want passed=%v keyHit=%v", rate, passed, keyHit, tc.wantPass, tc.wantKeyHit)
			}
		})
	}
}

func TestSpeechEndIgnoresTrailingSilence(t *testing.T) {
	samples := make([]float32, sampleRate*2)
	for i := range sampleRate { // first second is speech, second is padding
		samples[i] = 0.5
	}
	if got := speechEnd(samples); got < 960*time.Millisecond || got > 1050*time.Millisecond {
		t.Fatalf("speech end = %v, want about 1s", got)
	}
}
