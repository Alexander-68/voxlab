//go:build !windows

package denoiser

import (
	"voxlab/pkg/config"
)

func newPlatformSpeechEnhancer(dllDir string, modelDir string, cfg config.EnhancerConfig) (SpeechEnhancer, error) {
	return NewBypassEnhancer(false), nil // passthrough: never report denoising as active
}
