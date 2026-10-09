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

func TestTTSManagerModelAndVoices(t *testing.T) {
	dsp := audio.NewDSPProcessor(16000, 80.0, -42.0, true, 0.12)
	sim := engine.NewSimulatorEngine()
	mgr := NewTTSManager(sim, dsp)

	// Check model-specific voices
	v11Voices := mgr.VoicesForModel("kokoro-multi-lang-v1_1")
	if len(v11Voices) != 103 {
		t.Errorf("expected 103 voices for kokoro-multi-lang-v1_1, got %d", len(v11Voices))
	}

	v10Voices := mgr.VoicesForModel("kokoro-multi-lang-v1_0")
	if len(v10Voices) != 54 {
		t.Errorf("expected 54 voices for kokoro-multi-lang-v1_0, got %d", len(v10Voices))
	}

	v019Voices := mgr.VoicesForModel("kokoro-en-v0_19")
	if len(v019Voices) != 11 {
		t.Errorf("expected 11 voices for kokoro-en-v0_19, got %d", len(v019Voices))
	}

	// Test SetModel to v1.0
	if err := mgr.SetModel("kokoro-multi-lang-v1_0"); err != nil {
		t.Errorf("SetModel failed: %v", err)
	}

	activeV10, _ := mgr.ActiveModel()
	if activeV10 != "kokoro-multi-lang-v1_0" {
		t.Errorf("expected active model kokoro-multi-lang-v1_0, got %s", activeV10)
	}

	curVoicesV10 := mgr.Voices()
	if len(curVoicesV10) != 54 {
		t.Errorf("expected 54 voices after switching to v1_0, got %d", len(curVoicesV10))
	}

	// Test SetModel to v0.19
	if err := mgr.SetModel("kokoro-en-v0_19"); err != nil {
		t.Errorf("SetModel failed: %v", err)
	}

	active, _ := mgr.ActiveModel()
	if active != "kokoro-en-v0_19" {
		t.Errorf("expected active model kokoro-en-v0_19, got %s", active)
	}

	curVoices := mgr.Voices()
	if len(curVoices) != 11 {
		t.Errorf("expected 11 voices after switching to v0_19, got %d", len(curVoices))
	}
}
