package tts

import (
	"testing"
	"time"

	"voxlab/pkg/audio"
	"voxlab/pkg/engine"
)

func TestTTSManagerEchoSuppression(t *testing.T) {
	dsp := audio.NewDSPProcessor(16000, 80.0, -42.0, true, 0.12)
	sim := engine.NewSimulatorEngine()
	mgr := NewTTSManager(sim, dsp)
	mgr.tailDelayMs = 50 * time.Millisecond

	// 1. Initial state: echo should not be muted
	if dsp.IsEchoMuted() {
		t.Errorf("expected echo to be unmuted initially")
	}

	// 2. Synthesize does NOT mute echo (synthesis is model computation only)
	res, err := mgr.Synthesize(engine.TTSRequest{
		Text:       "Hello world",
		Voice:      "af_heart",
		Speed:      1.0,
		SampleRate: 24000,
	})
	if err != nil {
		t.Fatalf("synthesize failed: %v", err)
	}
	if res == nil || len(res.WAVBytes) == 0 {
		t.Fatalf("expected valid WAV bytes")
	}

	if dsp.IsEchoMuted() {
		t.Errorf("expected echo to remain UNMUTED during/after Synthesize (playback has not started)")
	}

	// 3. Physical playback begins -> SetSpeaking(true) mutes echo
	mgr.SetSpeaking(true)
	if !dsp.IsEchoMuted() {
		t.Errorf("expected echo to be MUTED when speaker playback is active")
	}
	if !mgr.IsSpeaking() {
		t.Errorf("expected IsSpeaking() to be true")
	}

	// 4. Physical playback finishes -> SetSpeaking(false) keeps muted for tail delay, then unmutes
	mgr.SetSpeaking(false)
	if !dsp.IsEchoMuted() {
		t.Errorf("expected echo to remain muted immediately after playback ends (tail delay active)")
	}

	// Wait for tail delay to elapse
	time.Sleep(75 * time.Millisecond)
	if dsp.IsEchoMuted() {
		t.Errorf("expected echo to be UNMUTED after tail delay elapsed")
	}
	if mgr.IsSpeaking() {
		t.Errorf("expected IsSpeaking() to be false")
	}
}
