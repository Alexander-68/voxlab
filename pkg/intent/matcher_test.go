package intent

import (
	"testing"
)

func TestIntentMatcher(t *testing.T) {
	cat := DefaultCatalog()
	matcher := NewIntentMatcher(cat, 0.75, 0.10)

	// 1. Exact / strong match
	dec := matcher.Match("open settings")
	if dec.Decision != "ACCEPT" || dec.IntentID != "NAV_SETTINGS" {
		t.Errorf("expected ACCEPT NAV_SETTINGS, got %s %s (conf: %f)", dec.Decision, dec.IntentID, dec.Confidence)
	}

	// 2. Paraphrase / alias match
	dec2 := matcher.Match("go to preferences")
	if dec2.Decision != "ACCEPT" || dec2.IntentID != "NAV_SETTINGS" {
		t.Errorf("expected ACCEPT NAV_SETTINGS for preferences, got %s %s", dec2.Decision, dec2.IntentID)
	}

	// 3. Slot extraction
	dec3 := matcher.Match("show endoscope on the main monitor")
	if dec3.Decision != "ACCEPT" || dec3.IntentID != "ROUTE_VIDEO" {
		t.Fatalf("expected ACCEPT ROUTE_VIDEO, got %s %s", dec3.Decision, dec3.IntentID)
	}
	if dec3.Slots["source"] != "endoscope" {
		t.Errorf("expected source endoscope, got %s", dec3.Slots["source"])
	}
	if dec3.Slots["destination"] != "main_monitor" {
		t.Errorf("expected destination main_monitor, got %s", dec3.Slots["destination"])
	}

	// 4. Distractor suppression (room chatter)
	dec4 := matcher.Match("what time is lunch")
	if dec4.Decision != "REJECT" || dec4.Reason != "OUT_OF_DOMAIN_CHATTER" {
		t.Errorf("expected REJECT OUT_OF_DOMAIN_CHATTER, got %s %s", dec4.Decision, dec4.Reason)
	}

	// 5. Negation rejection
	dec5 := matcher.Match("do not stop recording")
	if dec5.Decision != "REJECT" || dec5.Reason != "NEGATION_DETECTED" {
		t.Errorf("expected REJECT NEGATION_DETECTED, got %s %s", dec5.Decision, dec5.Reason)
	}
}

// Utterances from linux-wpe-voice-recipe.md sections 4 and 13 that must never execute an action.
func TestUtterancesThatMustNotAct(t *testing.T) {
	tests := []struct {
		text, reason string // an empty reason accepts any non-ACCEPT decision
	}{
		{"don't stop recording", "NEGATION_DETECTED"},
		{"Don't stop recording.", "NEGATION_DETECTED"},
		{"don’t stop recording", "NEGATION_DETECTED"},
		{"DON'T STOP RECORDING", "NEGATION_DETECTED"},
		{"dont stop recording", "NEGATION_DETECTED"},
		{"Hey MediCapture, don't stop recording", "NEGATION_DETECTED"},
		{"won't stop recording", "NEGATION_DETECTED"},
		{"can't stop recording", "NEGATION_DETECTED"},
		{"you shouldn't start recording", "NEGATION_DETECTED"},
		{"do not stop recording", "NEGATION_DETECTED"},
		{"never start recording", "NEGATION_DETECTED"},
		{"start—no, stop recording", "NEGATION_DETECTED"},
		{"Hey MediCapture, start—no, stop recording", "NEGATION_DETECTED"},
		{"John said start recording yesterday", "NARRATION_DETECTED"},
		{"she told me to stop recording", "NARRATION_DETECTED"},
		{"I started recording yesterday", ""},
		{"Hey MediCapture, what time is lunch?", ""},
		{"Put that one over there", ""},
		{"show camera on the monitor", ""},
	}
	for name, catalog := range testCatalogs(t) {
		matcher := NewIntentMatcher(catalog, 0.75, 0.10)
		for _, tc := range tests {
			t.Run(name+"/"+tc.text, func(t *testing.T) {
				dec := matcher.Match(tc.text)
				if dec.Decision == "ACCEPT" {
					t.Fatalf("executed %s at confidence %.2f", dec.IntentID, dec.Confidence)
				}
				if tc.reason != "" && dec.Reason != tc.reason {
					t.Errorf("got %s %s, want reason %s", dec.Decision, dec.Reason, tc.reason)
				}
			})
		}
	}
}

// testCatalogs covers the built-in catalog and the one the server actually loads.
func testCatalogs(t *testing.T) map[string]*CommandCatalog {
	t.Helper()
	shipped, err := LoadCatalog("../../data/commands.json")
	if err != nil {
		t.Fatalf("loading shipped catalog: %v", err)
	}
	return map[string]*CommandCatalog{"default": DefaultCatalog(), "shipped": shipped}
}

// Negation and narration words reject unconditionally, so a command containing one could never run.
func TestCatalogExamplesAvoidRejectionWords(t *testing.T) {
	for name, catalog := range testCatalogs(t) {
		for _, cmd := range catalog.Commands {
			for _, ex := range cmd.Examples {
				for _, tok := range Tokenize(ex) {
					if negationWords[tok] || narrationWords[tok] {
						t.Errorf("%s: %s example %q contains rejection word %q", name, cmd.ID, ex, tok)
					}
				}
			}
		}
	}
}

// Guards the negation and narration lists against rejecting real commands.
func TestPlainCommandsStillAccepted(t *testing.T) {
	matcher := NewIntentMatcher(DefaultCatalog(), 0.75, 0.10)
	for text, want := range map[string]string{
		"stop recording":       "STOP_RECORDING",
		"Stop recording.":      "STOP_RECORDING",
		"start recording":      "START_RECORDING",
		"please open settings": "NAV_SETTINGS",
		"skip track":           "MEDIA_NEXT",
	} {
		if dec := matcher.Match(text); dec.Decision != "ACCEPT" || dec.IntentID != want {
			t.Errorf("%q: got %s %s (%s), want ACCEPT %s", text, dec.Decision, dec.IntentID, dec.Reason, want)
		}
	}
}
