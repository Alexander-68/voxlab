package server

import (
	"testing"
)

func TestFormatDictationPhrase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Chinese full stop is not doubled",
			input:    "患者生命体征平稳。",
			expected: "患者生命体征平稳。",
		},
		{
			name:     "Whitespace only",
			input:    "   \n  \t  ",
			expected: "",
		},
		{
			name:     "No punctuation spoken - clean phrase gets auto period",
			input:    "patient is doing well",
			expected: "Patient is doing well.",
		},
		{
			name:     "Spoken period - no duplicate period",
			input:    "patient is doing well period",
			expected: "Patient is doing well.",
		},
		{
			name:     "Spoken full stop - no duplicate period",
			input:    "vital signs are stable full stop",
			expected: "Vital signs are stable.",
		},
		{
			name:     "Spoken question mark - no duplicate period",
			input:    "is there any fever question mark",
			expected: "Is there any fever?",
		},
		{
			name:     "Spoken exclamation mark - no duplicate period",
			input:    "stat response required exclamation mark",
			expected: "Stat response required!",
		},
		{
			name:     "Spoken exclamation point - no duplicate period",
			input:    "stat response required exclamation point",
			expected: "Stat response required!",
		},
		{
			name:     "Spoken colon",
			input:    "findings colon clear lungs period",
			expected: "Findings: clear lungs.",
		},
		{
			name:     "Spoken semicolon keyword in middle gets auto period at end",
			input:    "patient stable semicolon clear lungs",
			expected: "Patient stable; clear lungs.",
		},
		{
			name:     "Spoken semicolon keyword at end",
			input:    "patient stable semicolon",
			expected: "Patient stable;",
		},
		{
			name:     "Spoken comma inside phrase gets auto period at end",
			input:    "headache comma nausea comma and fatigue",
			expected: "Headache, nausea, and fatigue.",
		},
		{
			name:     "Spoken comma at end - preserved",
			input:    "patient arrived comma",
			expected: "Patient arrived,",
		},
		{
			name:     "Spoken new line keyword",
			input:    "first note new line second note",
			expected: "First note\nSecond note.",
		},
		{
			name:     "Spoken new paragraph keyword",
			input:    "first note new paragraph second note period",
			expected: "First note\n\nSecond note.",
		},
		{
			name:     "Leading new line keyword",
			input:    "new line follow up next week",
			expected: "\nFollow up next week.",
		},
		{
			name:     "Multiple sentences with spoken periods",
			input:    "heart rate normal period blood pressure 120 over 80 period",
			expected: "Heart rate normal. Blood pressure 120 over 80.",
		},
		{
			name:     "Decimal dose and time survive",
			input:    "Give 2.5 mg at 10:30.",
			expected: "Give 2.5 mg at 10:30.",
		},
		{
			name:     "Thousands separator survives",
			input:    "Dose was 1,000 units.",
			expected: "Dose was 1,000 units.",
		},
		{
			name:     "Abbreviation is not split",
			input:    "Use a mild analgesic, e.g. paracetamol.",
			expected: "Use a mild analgesic, e.g. Paracetamol.",
		},
		{
			name:     "Spoken period after a decimal still converts",
			input:    "give 2.5 mg period",
			expected: "Give 2.5 mg.",
		},
		{
			name:     "Spoken period in punctuated ASR output",
			input:    "The patient is stable period, the wound is clean.",
			expected: "The patient is stable. The wound is clean.",
		},
		{
			name:     "Spoken period at the end of punctuated ASR output",
			input:    "The patient is stable, period.",
			expected: "The patient is stable.",
		},
		{
			name:     "Spoken colon in punctuated ASR output",
			input:    "Findings colon clear lungs.",
			expected: "Findings: clear lungs.",
		},
		{
			name:     "Keywords around a line break in punctuated ASR output",
			input:    "Follow up in two weeks period new paragraph, no further action period.",
			expected: "Follow up in two weeks.\n\nNo further action.",
		},
		{
			name:     "Spoken period opening a phrase after a pause",
			input:    "period new paragraph follow up",
			expected: ".\n\nFollow up.",
		},
		{
			name:     "Noun period ending a sentence",
			input:    "It was a long period.",
			expected: "It was a long period.",
		},
		{
			name:     "Punctuated text keeps the organ",
			input:    "Colon biopsy taken.",
			expected: "Colon biopsy taken.",
		},
		{
			name:     "Punctuated text keeps the noun period",
			input:    "The postoperative period was uneventful.",
			expected: "The postoperative period was uneventful.",
		},
		{
			name:     "Unpunctuated noun period after a modifier",
			input:    "the postoperative period was uneventful",
			expected: "The postoperative period was uneventful.",
		},
		{
			name:     "Unpunctuated noun period before of",
			input:    "after a long period of observation",
			expected: "After a long period of observation.",
		},
		{
			name:     "Unpunctuated organ at phrase start",
			input:    "colon biopsy taken",
			expected: "Colon biopsy taken.",
		},
		{
			name:     "Unpunctuated organ after an anatomical modifier",
			input:    "polyp in the sigmoid colon period",
			expected: "Polyp in the sigmoid colon.",
		},
		{
			name:     "Lone spoken keyword still converts",
			input:    "period",
			expected: ".",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatDictationPhrase(tc.input)
			if got != tc.expected {
				t.Errorf("\nInput:    %q\nExpected: %q\nGot:      %q", tc.input, tc.expected, got)
			}
		})
	}
}

func TestAppendDictationPhrase(t *testing.T) {
	draft := ""

	// Turn 1: no punctuation spoken -> auto period on pause
	draft = AppendDictationPhrase(draft, "patient presented with mild fever")
	expected1 := "Patient presented with mild fever."
	if draft != expected1 {
		t.Fatalf("Turn 1 failed: got %q, expected %q", draft, expected1)
	}

	// Turn 2: spoken comma and period
	draft = AppendDictationPhrase(draft, "temperature 38 degrees comma pulse 85 period")
	expected2 := "Patient presented with mild fever. Temperature 38 degrees, pulse 85."
	if draft != expected2 {
		t.Fatalf("Turn 2 failed: got %q, expected %q", draft, expected2)
	}

	// Turn 3: spoken new line
	draft = AppendDictationPhrase(draft, "new line lungs are clear")
	expected3 := "Patient presented with mild fever. Temperature 38 degrees, pulse 85.\nLungs are clear."
	if draft != expected3 {
		t.Fatalf("Turn 3 failed: got %q, expected %q", draft, expected3)
	}

	// Turn 4: question mark
	draft = AppendDictationPhrase(draft, "any known drug allergies question mark")
	expected4 := "Patient presented with mild fever. Temperature 38 degrees, pulse 85.\nLungs are clear. Any known drug allergies?"
	if draft != expected4 {
		t.Fatalf("Turn 4 failed: got %q, expected %q", draft, expected4)
	}
}
