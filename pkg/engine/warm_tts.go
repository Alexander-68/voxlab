package engine

import (
	"voxlab/pkg/audio"
)

// WarmTTS represents an in-memory, persistent Text-to-Speech engine.
type WarmTTS interface {
	// Synthesize generates speech for the given text, speaker ID, and speed without process restarts.
	Synthesize(text string, sid int, speed float64) (*TTSResult, error)
	// SynthesizeStream generates speech and emits intermediate chunks via onChunk as soon as each clause/sentence completes.
	SynthesizeStream(text string, sid int, speed float64, onChunk func(chunk TTSChunk) error) (*TTSResult, error)
	// IsWarm returns true if the engine is initialized and ready in memory.
	IsWarm() bool
	// ModelKey returns the identifier of the currently loaded model.
	ModelKey() string
	// Close releases any allocated resources.
	Close() error
}

// float32SliceToTTSResult converts raw float32 samples into a TTSResult with encoded WAV bytes.
func float32SliceToTTSResult(samples []float32, sampleRate int, latencyMs int64) (*TTSResult, error) {
	wavBytes, err := audio.EncodeWAV(samples, sampleRate)
	if err != nil {
		return nil, err
	}
	duration := float64(len(samples)) / float64(sampleRate)
	return &TTSResult{
		AudioSamples: samples,
		WAVBytes:     wavBytes,
		SampleRate:   sampleRate,
		DurationSec:  duration,
		LatencyMs:    latencyMs,
	}, nil
}
