package audio

import (
	"log"
	"sync"

	"voxlab/pkg/config"
)

// SpeechEnhancer defines the contract for real-time neural speech enhancement.
type SpeechEnhancer interface {
	// Name returns a human-readable display name of the active enhancer.
	Name() string
	// ModelType returns "gtcrn", "dpdfnet", or "bypass".
	ModelType() string
	// ModelPath returns the path to the loaded model on disk.
	ModelPath() string
	// IsEnabled returns true if speech enhancement is currently active.
	IsEnabled() bool
	// SetEnabled toggles speech enhancement on or off.
	SetEnabled(enabled bool)
	// SwitchModel changes the active model architecture ("gtcrn" or "dpdfnet").
	SwitchModel(modelType string) error
	// ProcessChunk enhances a chunk of PCM float32 samples.
	// If bypassed or disabled, returns the original chunk with zero overhead.
	ProcessChunk(chunk []float32) []float32
	// Reset clears internal state and buffers (e.g. on audio source switch).
	Reset()
	// Close releases any native model resources.
	Close() error
}

// BypassEnhancer is a no-op speech enhancer used as a fallback or mock.
type BypassEnhancer struct {
	mu        sync.Mutex
	enabled   bool
	modelType string
}

// NewBypassEnhancer returns a new BypassEnhancer.
func NewBypassEnhancer(enabled bool) *BypassEnhancer {
	return &BypassEnhancer{
		enabled:   enabled,
		modelType: "bypass",
	}
}

func (b *BypassEnhancer) Name() string {
	return "Bypass / Passthrough"
}

func (b *BypassEnhancer) ModelType() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.modelType
}

func (b *BypassEnhancer) ModelPath() string {
	return ""
}

func (b *BypassEnhancer) IsEnabled() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.enabled
}

func (b *BypassEnhancer) SetEnabled(enabled bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.enabled = enabled
}

func (b *BypassEnhancer) SwitchModel(modelType string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.modelType = modelType
	return nil
}

func (b *BypassEnhancer) ProcessChunk(chunk []float32) []float32 {
	return chunk
}

func (b *BypassEnhancer) Reset() {}

func (b *BypassEnhancer) Close() error {
	return nil
}

// NewSpeechEnhancer creates a speech enhancer. On supported platforms (e.g. Windows with C-API DLL),
// it initializes an in-process neural denoiser; otherwise it falls back to BypassEnhancer.
func NewSpeechEnhancer(dllDir string, modelDir string, cfg config.EnhancerConfig) SpeechEnhancer {
	enhancer, err := newPlatformSpeechEnhancer(dllDir, modelDir, cfg)
	if err != nil {
		log.Printf("[SpeechEnhancer] Native denoiser initialization skipped (%v); using bypass enhancer", err)
		return NewBypassEnhancer(cfg.Enabled)
	}
	return enhancer
}
