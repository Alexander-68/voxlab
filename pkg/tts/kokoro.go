package tts

import (
	"sync"
	"time"

	"voxlab/pkg/audio"
	"voxlab/pkg/engine"
)

// VoiceProfile holds metadata for a Kokoro speaker voice.
type VoiceProfile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Gender      string `json:"gender"`
	Description string `json:"description"`
}

// AvailableKokoroVoices lists the pre-trained Kokoro speaker IDs available in model weights.
var AvailableKokoroVoices = []VoiceProfile{
	{ID: "af", Name: "Heart (Female)", Gender: "female", Description: "Warm, natural American English (Default)"},
	{ID: "am_adam", Name: "Adam (Male)", Gender: "male", Description: "Deep, crisp American English male"},
	{ID: "am_michael", Name: "Michael (Male)", Gender: "male", Description: "Neutral and professional American male"},
	{ID: "bm_george", Name: "George (British Male)", Gender: "male", Description: "Refined, classic British English male"},
	{ID: "bm_lewis", Name: "Lewis (British Male)", Gender: "male", Description: "Deep, resonant British narrator male"},
	{ID: "af_bella", Name: "Bella (Female)", Gender: "female", Description: "Friendly and expressive American female"},
	{ID: "af_nicole", Name: "Nicole (Female)", Gender: "female", Description: "Calm, clear American narrator female"},
	{ID: "af_sarah", Name: "Sarah (Female)", Gender: "female", Description: "Casual and bright American female"},
	{ID: "af_sky", Name: "Sky (Female)", Gender: "female", Description: "Soft, melodic conversational American voice"},
	{ID: "bf_emma", Name: "Emma (British Female)", Gender: "female", Description: "Polite, articulate British English female"},
	{ID: "bf_isabella", Name: "Isabella (British Female)", Gender: "female", Description: "Warm British conversational female"},
}

// TTSManager coordinates speech synthesis requests and half-duplex echo suppression.
type TTSManager struct {
	mu           sync.Mutex
	engine       engine.SpeechEngine
	dsp          *audio.DSPProcessor
	isSpeaking   bool
	tailDelayMs  time.Duration
}

// NewTTSManager creates an initialized TTS manager.
func NewTTSManager(eng engine.SpeechEngine, dsp *audio.DSPProcessor) *TTSManager {
	return &TTSManager{
		engine:      eng,
		dsp:         dsp,
		tailDelayMs: 150 * time.Millisecond,
	}
}

// Synthesize generates speech audio without muting input (since speaker playback has not started yet).
func (m *TTSManager) Synthesize(req engine.TTSRequest) (*engine.TTSResult, error) {
	return m.engine.Synthesize(req)
}

// SetSpeaking manages half-duplex echo suppression during actual audio playback through speakers.
func (m *TTSManager) SetSpeaking(speaking bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isSpeaking = speaking
	if speaking {
		if m.dsp != nil {
			m.dsp.SetEchoMuted(true)
		}
	} else {
		// Asynchronously clear echo mute after tail delay to suppress acoustic room reverberation
		go func() {
			time.Sleep(m.tailDelayMs)
			m.mu.Lock()
			if !m.isSpeaking && m.dsp != nil {
				m.dsp.SetEchoMuted(false)
			}
			m.mu.Unlock()
		}()
	}
}

// IsSpeaking returns true if TTS is currently active.
func (m *TTSManager) IsSpeaking() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isSpeaking
}

// Voices returns the list of available Kokoro voices.
func (m *TTSManager) Voices() []VoiceProfile {
	return AvailableKokoroVoices
}

// ActiveModel returns the current TTS model name and neural capability.
func (m *TTSManager) ActiveModel() (string, bool) {
	if m.engine != nil {
		return m.engine.TTSModelInfo()
	}
	return "None", false
}
