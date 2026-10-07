package audio

import (
	"encoding/binary"
	"math"
	"sync"
	"sync/atomic"
)

// DSPProcessor manages real-time filtering, gating, and level measurements.
type DSPProcessor struct {
	mu           sync.Mutex
	sampleRate   float64
	cutoffHz     float64
	highpassEnabled bool
	alpha           float64
	prevInput       float64
	prevOutput      float64
	gateDBFS        float64
	agcEnabled      bool
	targetRMS       float64
	currentGain     float64
	echoMuted       int32 // atomic 1 if muted during speaker playback
}

// NewDSPProcessor creates a new DSPProcessor with the specified settings.
func NewDSPProcessor(sampleRate int, cutoffHz float64, gateDBFS float64, agcEnabled bool, targetRMS float64) *DSPProcessor {
	sr := float64(sampleRate)
	// Calculate first-order highpass filter alpha: RC = 1 / (2*pi*fc), alpha = RC / (RC + dt)
	dt := 1.0 / sr
	rc := 1.0 / (2.0 * math.Pi * cutoffHz)
	alpha := rc / (rc + dt)

	return &DSPProcessor{
		sampleRate:      sr,
		cutoffHz:        cutoffHz,
		highpassEnabled: true,
		alpha:           alpha,
		gateDBFS:        gateDBFS,
		agcEnabled:      agcEnabled,
		targetRMS:       targetRMS,
		currentGain:     1.0,
	}
}

// SetHighPassEnabled enables or disables the 80Hz de-rumble high-pass filter.
func (p *DSPProcessor) SetHighPassEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.highpassEnabled = enabled
	p.prevInput = 0
	p.prevOutput = 0
}

// IsHighPassEnabled returns true if the high-pass filter is active.
func (p *DSPProcessor) IsHighPassEnabled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.highpassEnabled
}

// SetEchoMuted enables or disables the half-duplex echo suppression latch.
func (p *DSPProcessor) SetEchoMuted(muted bool) {
	if muted {
		atomic.StoreInt32(&p.echoMuted, 1)
	} else {
		atomic.StoreInt32(&p.echoMuted, 0)
	}
}

// SetGateThreshold updates the noise floor gate cutoff level in dBFS.
func (p *DSPProcessor) SetGateThreshold(dbfs float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gateDBFS = dbfs
}

// GateThreshold returns the active noise gate threshold in dBFS.
func (p *DSPProcessor) GateThreshold() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.gateDBFS
}

// SetAGCEnabled toggles automatic gain control.
func (p *DSPProcessor) SetAGCEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.agcEnabled = enabled
	if !enabled {
		p.currentGain = 1.0
	}
}

// IsAGCEnabled returns whether AGC is currently active.
func (p *DSPProcessor) IsAGCEnabled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.agcEnabled
}

// CurrentGain returns the current software AGC multiplier.
func (p *DSPProcessor) CurrentGain() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.agcEnabled {
		return 1.0
	}
	return p.currentGain
}

// IsEchoMuted returns true if the processor is currently muting mic due to speaker playback.
func (p *DSPProcessor) IsEchoMuted() bool {
	return atomic.LoadInt32(&p.echoMuted) == 1
}

// CalculateRMS returns the root-mean-square amplitude and corresponding dBFS value.
func CalculateRMS(samples []float32) (rms float64, dbfs float64) {
	if len(samples) == 0 {
		return 0.0, -100.0
	}
	var sumSquares float64
	for _, s := range samples {
		v := float64(s)
		sumSquares += v * v
	}
	meanSquare := sumSquares / float64(len(samples))
	rms = math.Sqrt(meanSquare)
	if rms <= 1e-9 {
		dbfs = -100.0
	} else {
		dbfs = 20.0 * math.Log10(rms)
	}
	return rms, dbfs
}

// ProcessChunk applies highpass filtering, noise floor gating, and optional AGC to a chunk.
// Returns the processed samples, the RMS, the dBFS level, and whether the chunk passed the noise gate.
func (p *DSPProcessor) ProcessChunk(samples []float32) (processed []float32, rms float64, dbfs float64, passedGate bool) {
	if len(samples) == 0 {
		return samples, 0, -100, false
	}

	// If echo muted, drop/zero chunk immediately
	if p.IsEchoMuted() {
		zeros := make([]float32, len(samples))
		return zeros, 0, -100, false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	out := make([]float32, len(samples))

	// Step 1: Highpass filter (DC removal and rumble elimination)
	if p.highpassEnabled {
		alpha := p.alpha
		prevIn := p.prevInput
		prevOut := p.prevOutput

		for i, s := range samples {
			in := float64(s)
			filtered := alpha * (prevOut + in - prevIn)
			prevIn = in
			prevOut = filtered
			out[i] = float32(filtered)
		}
		p.prevInput = prevIn
		p.prevOutput = prevOut
	} else {
		copy(out, samples)
	}

	// Step 2: Measure energy
	rms, dbfs = CalculateRMS(out)

	// Step 3: Noise gate check
	if dbfs < p.gateDBFS {
		// Silence below noise floor
		for i := range out {
			out[i] = 0.0
		}
		return out, rms, dbfs, false
	}

	// Step 4: Automatic Gain Control (AGC) if enabled
	if p.agcEnabled && rms > 1e-4 {
		targetGain := p.targetRMS / rms
		// Limit gain range between 0.2x and 5.0x to prevent noise explosion
		if targetGain > 5.0 {
			targetGain = 5.0
		} else if targetGain < 0.2 {
			targetGain = 0.2
		}

		// Smooth gain changes (one-pole filter on gain)
		p.currentGain = 0.9*p.currentGain + 0.1*targetGain

		for i, v := range out {
			amplified := float64(v) * p.currentGain
			// Soft-clip to [-1.0, 1.0]
			if amplified > 1.0 {
				amplified = 1.0
			} else if amplified < -1.0 {
				amplified = -1.0
			}
			out[i] = float32(amplified)
		}
	}

	return out, rms, dbfs, true
}

// BytesToInt16PCM converts 16-bit little-endian byte array to []int16.
func BytesToInt16PCM(data []byte) []int16 {
	numSamples := len(data) / 2
	samples := make([]int16, numSamples)
	for i := 0; i < numSamples; i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
	}
	return samples
}

// Int16ToFloat32 converts []int16 to []float32 in range [-1.0, 1.0].
func Int16ToFloat32(samples []int16) []float32 {
	out := make([]float32, len(samples))
	for i, s := range samples {
		out[i] = float32(s) / 32768.0
	}
	return out
}

// Float32ToInt16 converts []float32 in range [-1.0, 1.0] to []int16.
func Float32ToInt16(samples []float32) []int16 {
	out := make([]int16, len(samples))
	for i, s := range samples {
		v := s
		if v > 1.0 {
			v = 1.0
		} else if v < -1.0 {
			v = -1.0
		}
		out[i] = int16(v * 32767.0)
	}
	return out
}

// BytesToFloat32PCM converts 16-bit little-endian byte array directly to []float32.
func BytesToFloat32PCM(data []byte) []float32 {
	return Int16ToFloat32(BytesToInt16PCM(data))
}

// Float32ToBytesPCM converts []float32 to 16-bit little-endian byte array.
func Float32ToBytesPCM(samples []float32) []byte {
	out := make([]byte, len(samples)*2)
	intSamples := Float32ToInt16(samples)
	for i, s := range intSamples {
		binary.LittleEndian.PutUint16(out[i*2:i*2+2], uint16(s))
	}
	return out
}
