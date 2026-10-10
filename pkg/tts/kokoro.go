package tts

import (
	"strings"
	"sync"
	"time"

	"voxlab/pkg/audio"
	"voxlab/pkg/engine"
)

// VoiceProfile aliases engine.VoiceProfile.
type VoiceProfile = engine.VoiceProfile

// AvailableKokoroVoices references engine.AvailableKokoroV11Voices.
var AvailableKokoroVoices = engine.AvailableKokoroV11Voices

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

// SynthesizeStream generates streaming speech audio chunks without muting input.
func (m *TTSManager) SynthesizeStream(req engine.TTSRequest, onChunk func(chunk engine.TTSChunk) error) (*engine.TTSResult, error) {
	return m.engine.SynthesizeStream(req, onChunk)
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

// Voices returns the list of available Kokoro voices for the currently active model.
func (m *TTSManager) Voices() []VoiceProfile {
	model, _ := m.ActiveModel()
	return m.VoicesForModel(model)
}

// VoicesForModel returns the voice profiles matching a given model name.
func (m *TTSManager) VoicesForModel(modelName string) []VoiceProfile {
	low := strings.ToLower(modelName)
	if strings.Contains(low, "v0_19") || strings.Contains(low, "v0.19") {
		return engine.AvailableKokoroV019Voices
	}
	if strings.Contains(low, "v1_0") || strings.Contains(low, "v1.0") {
		return engine.AvailableKokoroV10Voices
	}
	return engine.AvailableKokoroV11Voices
}

// ActiveModel returns the current TTS model name and neural capability.
func (m *TTSManager) ActiveModel() (string, bool) {
	if m.engine != nil {
		return m.engine.TTSModelInfo()
	}
	return "None", false
}

// SetModel switches the active TTS model in the speech engine.
func (m *TTSManager) SetModel(modelName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.engine != nil {
		return m.engine.SetTTSModel(modelName)
	}
	return nil
}

// InstalledModels returns the list of installed TTS model directory names.
func (m *TTSManager) InstalledModels() []string {
	if m.engine != nil {
		return m.engine.InstalledTTSModels()
	}
	return nil
}
