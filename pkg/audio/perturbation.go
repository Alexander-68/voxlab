package audio

import (
	"math"
	"math/rand"
)

// NoiseType defines acoustic perturbation types.
const (
	NoiseClean  = "clean"
	NoiseWhite  = "white"
	NoisePink   = "pink"
	NoiseTyping = "typing"
	NoiseBabble = "babble"
	NoiseReverb = "reverb"
)

// SoftLimit smoothly compresses samples exceeding 0.95 amplitude without hard digital clipping.
func SoftLimit(samples []float32) []float32 {
	out := make([]float32, len(samples))
	const threshold = 0.95
	const margin = 1.0 - threshold

	for i, s := range samples {
		abs := math.Abs(float64(s))
		if abs <= threshold {
			out[i] = s
		} else {
			excess := (abs - threshold) / margin
			compressed := threshold + margin*math.Tanh(excess)
			if s < 0 {
				out[i] = float32(-compressed)
			} else {
				out[i] = float32(compressed)
			}
		}
	}
	return out
}

// AddNoiseAtSNR mixes acoustic noise into speech samples at a designated Signal-to-Noise Ratio (dB).
// If snrDB >= 100 or noiseType == NoiseClean, the speech is returned unaltered.
func AddNoiseAtSNR(speech []float32, noiseType string, snrDB float64, sampleRate int, seed int64) []float32 {
	if len(speech) == 0 || noiseType == NoiseClean || noiseType == "" || snrDB >= 100.0 {
		dup := make([]float32, len(speech))
		copy(dup, speech)
		return dup
	}

	if noiseType == NoiseReverb {
		// Reverb is a spatial reflection degradation rather than additive noise
		// 0.35s decay with mix proportional to SNR (lower SNR = wetter room)
		wet := float32(math.Max(0.15, math.Min(0.70, 0.70-snrDB*0.02)))
		return ApplyReverb(speech, sampleRate, 0.40, wet)
	}

	speechRMS, _ := CalculateRMS(speech)
	var noise []float32

	switch noiseType {
	case NoisePink:
		noise = GeneratePinkNoise(len(speech), seed)
	case NoiseTyping:
		noise = GenerateTypingNoise(len(speech), sampleRate, seed)
	case NoiseBabble:
		noise = GenerateBabbleNoise(len(speech), sampleRate, seed)
	case NoiseWhite:
		fallthrough
	default:
		noise = GenerateWhiteNoise(len(speech), seed)
	}

	noiseRMS, _ := CalculateRMS(noise)
	if noiseRMS < 1e-7 {
		noiseRMS = 1e-7
	}

	// Calculate target noise RMS
	var targetNoiseRMS float64
	if speechRMS > 1e-5 {
		// Target SNR = 20 * log10(speechRMS / targetNoiseRMS)
		// => targetNoiseRMS = speechRMS / 10^(snrDB / 20)
		targetNoiseRMS = speechRMS * math.Pow(10.0, -snrDB/20.0)
	} else {
		// Pure background noise (e.g. -46 dBFS)
		targetNoiseRMS = math.Pow(10.0, -46.0/20.0)
	}

	scale := targetNoiseRMS / noiseRMS
	mixed := make([]float32, len(speech))
	for i := range speech {
		mixed[i] = speech[i] + float32(float64(noise[i])*scale)
	}

	return SoftLimit(mixed)
}

// GenerateWhiteNoise produces zero-mean white noise.
func GenerateWhiteNoise(length int, seed int64) []float32 {
	rng := rand.New(rand.NewSource(seed))
	out := make([]float32, length)
	for i := range out {
		// Standard normal distribution via Box-Muller
		u1 := rng.Float64()
		u2 := rng.Float64()
		for u1 <= 1e-9 {
			u1 = rng.Float64()
		}
		z := math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
		out[i] = float32(z * 0.3)
	}
	return out
}

// GeneratePinkNoise produces 1/f filtered pink noise using Paul Kellet's filter.
func GeneratePinkNoise(length int, seed int64) []float32 {
	rng := rand.New(rand.NewSource(seed))
	out := make([]float32, length)

	var b0, b1, b2, b3, b4, b5, b6 float64

	for i := range out {
		white := (rng.Float64()*2.0 - 1.0) * 0.5
		b0 = 0.99886*b0 + white*0.0555179
		b1 = 0.99332*b1 + white*0.0750759
		b2 = 0.96900*b2 + white*0.1538520
		b3 = 0.86650*b3 + white*0.3104856
		b4 = 0.55000*b4 + white*0.5329522
		b5 = -0.7616*b5 - white*0.0168980
		pink := b0 + b1 + b2 + b3 + b4 + b5 + b6 + white*0.5362
		b6 = white * 0.115926
		out[i] = float32(pink * 0.25)
	}
	return out
}

// GenerateTypingNoise generates realistic mechanical keyboard clicking transients.
func GenerateTypingNoise(length int, sampleRate int, seed int64) []float32 {
	rng := rand.New(rand.NewSource(seed))
	out := make([]float32, length)

	// Mechanical typing at ~55 WPM is approx one stroke every 150-250ms
	pos := int(float64(sampleRate) * (0.05 + rng.Float64()*0.10))

	for pos < length {
		// Key stroke parameters
		// Click resonance frequency (high click ~ 2500 - 4500 Hz, low thump ~ 250 - 450 Hz)
		freqHigh := 2500.0 + rng.Float64()*2000.0
		freqLow := 250.0 + rng.Float64()*200.0
		clickLen := int(float64(sampleRate) * (0.025 + rng.Float64()*0.02)) // 25-45ms total click transient

		clickEnd := min(pos+clickLen, length)
		amp := 0.4 + rng.Float64()*0.4

		for j := pos; j < clickEnd; j++ {
			t := float64(j-pos) / float64(sampleRate)
			// Sharp exponential decay
			decayHigh := math.Exp(-t / 0.005)
			decayLow := math.Exp(-t / 0.018)

			burst := math.Sin(2.0*math.Pi*freqHigh*t)*decayHigh*0.7 +
				math.Sin(2.0*math.Pi*freqLow*t)*decayLow*0.5 +
				(rng.Float64()*2.0-1.0)*decayHigh*0.3

			out[j] += float32(burst * amp)
		}

		// Interval to next keystroke (120ms to 280ms)
		interval := int(float64(sampleRate) * (0.12 + rng.Float64()*0.16))
		pos += interval
	}

	return out
}

// GenerateBabbleNoise simulates multi-talker conversational murmur (speech babble).
func GenerateBabbleNoise(length int, sampleRate int, seed int64) []float32 {
	rng := rand.New(rand.NewSource(seed))
	out := make([]float32, length)

	// Babble is composed of several formant frequency bands with slow syllabic modulation (3 - 6 Hz)
	formants := []struct {
		freq      float64
		bandwidth float64
		modFreq   float64
	}{
		{freq: 350.0, bandwidth: 80.0, modFreq: 3.5},
		{freq: 720.0, bandwidth: 120.0, modFreq: 4.2},
		{freq: 1250.0, bandwidth: 180.0, modFreq: 3.1},
		{freq: 2100.0, bandwidth: 250.0, modFreq: 5.0},
		{freq: 2900.0, bandwidth: 300.0, modFreq: 4.6},
	}

	dt := 1.0 / float64(sampleRate)

	for _, f := range formants {
		phase := rng.Float64() * 2.0 * math.Pi
		modPhase := rng.Float64() * 2.0 * math.Pi
		q := f.freq / f.bandwidth

		// Filter white noise with 2nd order bandpass
		omega := 2.0 * math.Pi * f.freq * dt
		alpha := math.Sin(omega) / (2.0 * q)
		b0 := alpha
		b2 := -alpha
		a0 := 1.0 + alpha
		a1 := -2.0 * math.Cos(omega)
		a2 := 1.0 - alpha

		normB0 := b0 / a0
		normB2 := b2 / a0
		normA1 := a1 / a0
		normA2 := a2 / a0

		var x1, x2, y1, y2 float64

		for i := 0; i < length; i++ {
			t := float64(i) * dt
			white := rng.Float64()*2.0 - 1.0
			filtered := normB0*white + normB2*x2 - normA1*y1 - normA2*y2
			x2 = x1
			x1 = white
			y2 = y1
			y1 = filtered

			// Syllabic modulation envelope
			mod := 0.6 + 0.4*math.Sin(2.0*math.Pi*f.modFreq*t+modPhase)
			// Carrier oscillation
			carrier := math.Sin(2.0*math.Pi*f.freq*t + phase)

			out[i] += float32((filtered*0.6 + carrier*0.4) * mod * 0.25)
		}
	}

	return out
}

// ApplyReverb applies Schroeder acoustic reverberation with specified decay time (RT60).
func ApplyReverb(samples []float32, sampleRate int, decaySec float64, wetMix float32) []float32 {
	if len(samples) == 0 || wetMix <= 0 {
		dup := make([]float32, len(samples))
		copy(dup, samples)
		return dup
	}

	sr := float64(sampleRate)

	// 4 Parallel Feedback Comb Filters
	combDelaysMs := []float64{29.7, 37.1, 41.5, 43.7}
	type combFilter struct {
		buffer []float64
		idx    int
		gain   float64
	}

	combs := make([]combFilter, len(combDelaysMs))
	for i, delayMs := range combDelaysMs {
		delaySamples := int(delayMs * sr / 1000.0)
		if delaySamples < 1 {
			delaySamples = 1
		}
		// Feedback gain g for RT60 decay: g = 10^(-3 * delay / RT60)
		gain := math.Pow(10.0, -3.0*(delayMs/1000.0)/decaySec)
		combs[i] = combFilter{
			buffer: make([]float64, delaySamples),
			gain:   gain,
		}
	}

	// 2 Series All-pass Filters for diffusion
	allpassDelaysMs := []float64{5.0, 1.7}
	allpassGain := 0.7
	type allpassFilter struct {
		buffer []float64
		idx    int
	}
	allpasses := make([]allpassFilter, len(allpassDelaysMs))
	for i, delayMs := range allpassDelaysMs {
		delaySamples := int(delayMs * sr / 1000.0)
		if delaySamples < 1 {
			delaySamples = 1
		}
		allpasses[i] = allpassFilter{
			buffer: make([]float64, delaySamples),
		}
	}

	out := make([]float32, len(samples))
	dryMix := 1.0 - wetMix

	for i, s := range samples {
		in := float64(s)

		// Sum comb outputs
		var combSum float64
		for c := range combs {
			cb := &combs[c]
			delayed := cb.buffer[cb.idx]
			combSum += delayed
			cb.buffer[cb.idx] = in + delayed*cb.gain
			cb.idx = (cb.idx + 1) % len(cb.buffer)
		}
		combSum *= 0.25 // normalize sum of 4 combs

		// Series allpass diffusion
		apOut := combSum
		for a := range allpasses {
			ap := &allpasses[a]
			delayed := ap.buffer[ap.idx]
			temp := apOut + delayed*allpassGain
			ap.buffer[ap.idx] = temp
			apOut = -allpassGain*temp + delayed
			ap.idx = (ap.idx + 1) % len(ap.buffer)
		}

		wet := float32(apOut)
		out[i] = s*dryMix + wet*wetMix
	}

	return SoftLimit(out)
}
