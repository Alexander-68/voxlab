package audio

import "voxlab/pkg/audio/denoiser"

// Re-export denoiser contracts and constructors for audio package compatibility.
type SpeechEnhancer = denoiser.SpeechEnhancer
type BypassEnhancer = denoiser.BypassEnhancer

var (
	NewSpeechEnhancer = denoiser.NewSpeechEnhancer
	NewBypassEnhancer = denoiser.NewBypassEnhancer
)
