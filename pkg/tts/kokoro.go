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

// AvailableKokoroVoices lists the primary pre-trained Kokoro speaker IDs.
var AvailableKokoroVoices = []VoiceProfile{
	{ID: "af_heart", Name: "Heart (Female)", Gender: "female", Description: "Warm, natural American English"},
	{ID: "af_alloy", Name: "Alloy (Female)", Gender: "female", Description: "Crisp and clear assistant tone"},
	{ID: "af_aoede", Name: "Aoede (Female)", Gender: "female", Description: "Melodic conversational voice"},
	{ID: "af_bella", Name: "Bella (Female)", Gender: "female", Description: "Friendly and expressive"},
	{ID: "am_adam", Name: "Adam (Male)", Gender: "male", Description: "Deep, calm American English"},
	{ID: "am_fenrir", Name: "Fenrir (Male)", Gender: "male", Description: "Commanding and authoritative"},
	{ID: "am_michael", Name: "Michael (Male)", Gender: "male", Description: "Neutral and professional"},
	{ID: "am_puck", Name: "Puck (Male)", Gender: "male", Description: "Energetic and lively"},
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
