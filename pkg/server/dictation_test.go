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
			name:     "Whitespace only",
			input:    "   \n  \t  ",
			expected: "",
		},
		{
			name:     "No punctuation spoken - clean phrase",
			input:    "patient is doing well",
			expected: "Patient is doing well",
		},
		{
			name:     "Spoken period - no semicolon appended",
			input:    "patient is doing well period",
			expected: "Patient is doing well.",
		},
		{
			name:     "Spoken full stop - no semicolon appended",
			input:    "vital signs are stable full stop",
			expected: "Vital signs are stable.",
		},
		{
			name:     "Spoken question mark - no semicolon appended",
			input:    "is there any fever question mark",
			expected: "Is there any fever?",
		},
		{
			name:     "Spoken exclamation mark - no semicolon appended",
			input:    "stat response required exclamation mark",
			expected: "Stat response required!",
		},
		{
			name:     "Spoken exclamation point - no semicolon appended",
			input:    "stat response required exclamation point",
			expected: "Stat response required!",
		},
		{
			name:     "Spoken colon",
			input:    "findings colon clear lungs period",
			expected: "Findings: clear lungs.",
		},
		{
			name:     "Spoken semicolon keyword in middle",
			input:    "patient stable semicolon clear lungs",
			expected: "Patient stable; clear lungs",
		},
		{
			name:     "Spoken semicolon keyword at end",
			input:    "patient stable semicolon",
			expected: "Patient stable;",
		},
		{
			name:     "Spoken comma inside phrase",
			input:    "headache comma nausea comma and fatigue",
			expected: "Headache, nausea, and fatigue",
		},
		{
			name:     "Spoken comma at end - no semicolon added",
			input:    "patient arrived comma",
			expected: "Patient arrived,",
		},
		{
			name:     "Spoken new line keyword",
			input:    "first note new line second note",
			expected: "First note\nSecond note",
		},
		{
			name:     "Spoken new paragraph keyword",
			input:    "first note new paragraph second note period",
			expected: "First note\n\nSecond note.",
		},
		{
			name:     "Leading new line keyword",
			input:    "new line follow up next week",
			expected: "\nFollow up next week",
		},
		{
			name:     "Multiple sentences with spoken periods",
			input:    "heart rate normal period blood pressure 120 over 80 period",
			expected: "Heart rate normal. Blood pressure 120 over 80.",
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

	// Turn 1: no punctuation spoken
	draft = AppendDictationPhrase(draft, "patient presented with mild fever")
	expected1 := "Patient presented with mild fever"
	if draft != expected1 {
		t.Fatalf("Turn 1 failed: got %q, expected %q", draft, expected1)
	}

	// Turn 2: spoken comma and period
	draft = AppendDictationPhrase(draft, "temperature 38 degrees comma pulse 85 period")
	expected2 := "Patient presented with mild fever temperature 38 degrees, pulse 85."
	if draft != expected2 {
		t.Fatalf("Turn 2 failed: got %q, expected %q", draft, expected2)
	}

	// Turn 3: spoken new line
	draft = AppendDictationPhrase(draft, "new line lungs are clear")
	expected3 := "Patient presented with mild fever temperature 38 degrees, pulse 85.\nLungs are clear"
	if draft != expected3 {
		t.Fatalf("Turn 3 failed: got %q, expected %q", draft, expected3)
	}

	// Turn 4: question mark
	draft = AppendDictationPhrase(draft, "any known drug allergies question mark")
	expected4 := "Patient presented with mild fever temperature 38 degrees, pulse 85.\nLungs are clear any known drug allergies?"
	if draft != expected4 {
		t.Fatalf("Turn 4 failed: got %q, expected %q", draft, expected4)
	}
}
