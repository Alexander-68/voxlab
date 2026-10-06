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
