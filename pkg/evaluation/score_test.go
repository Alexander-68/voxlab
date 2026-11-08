package evaluation

import (
	"testing"
	"time"

	"voxlab/pkg/audio"
)

func TestScoreDictationReExport(t *testing.T) {
	item := EvalItem{Text: "Blood pressure is stable.", Expected: "blood pressure"}
	draft := "blood pressure is stable"
	rate, passed, keyHit := ScoreDictation(item, draft)
	if !passed || !keyHit || rate != 0.0 {
		t.Fatalf("expected perfect pass, got rate=%.2f passed=%v keyHit=%v", rate, passed, keyHit)
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

func TestUntestedKPIsAreNotPerfect(t *testing.T) {
	r := ComputeSuiteReport("t", []ItemResult{{Class: "ACCEPT", Mode: ModeCommand, Passed: true}}, time.Second)
	if r.IntentAccuracy == nil || *r.IntentAccuracy != 1 {
		t.Fatalf("intent accuracy = %v, want 1", r.IntentAccuracy)
	}
	if r.GatePrecision != nil || r.RejectionAccuracy != nil {
		t.Fatalf("no gate or rejection items, got %v and %v", r.GatePrecision, r.RejectionAccuracy)
	}
}

func TestScaleToDBFSReExport(t *testing.T) {
	samples := make([]float32, sampleRate)
	for i := range chunkSamples {
		samples[i] = 0.5
	}
	// Verify audio CalculateRMS integration
	if rms, _ := audio.CalculateRMS(samples[:chunkSamples]); rms == 0 {
		t.Fatalf("expected non-zero rms")
	}
}
