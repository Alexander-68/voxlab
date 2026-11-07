package audio

import (
	"math"
	"testing"
)

func TestSoftLimit(t *testing.T) {
	// Overdriven samples
	overdriven := []float32{-2.0, -1.0, -0.5, 0.0, 0.5, 1.0, 2.0}
	limited := SoftLimit(overdriven)

	for i, v := range limited {
		if math.Abs(float64(v)) > 1.0 {
			t.Errorf("sample [%d] = %f exceeds 1.0", i, v)
		}
	}

	// Verify linear region below 0.95 is untouched
	if math.Abs(float64(limited[2])-(-0.5)) > 1e-6 {
		t.Errorf("expected -0.5 untouched, got %f", limited[2])
	}
	if math.Abs(float64(limited[4])-(0.5)) > 1e-6 {
		t.Errorf("expected 0.5 untouched, got %f", limited[4])
	}
}

func TestAddNoiseAtSNR(t *testing.T) {
	// Generate speech signal (440Hz + 880Hz sine)
	speech := make([]float32, 16000)
	for i := range speech {
		speech[i] = float32(0.3*math.Sin(2.0*math.Pi*440.0*float64(i)/16000.0) +
			0.2*math.Sin(2.0*math.Pi*880.0*float64(i)/16000.0))
	}
	origRMS, _ := CalculateRMS(speech)

	// Test with +20 dB SNR white noise
	snr20 := AddNoiseAtSNR(speech, NoiseWhite, 20.0, 16000, 42)
	if len(snr20) != len(speech) {
		t.Fatalf("length mismatch: got %d, expected %d", len(snr20), len(speech))
	}

	// Difference is the added noise
	noise20 := make([]float32, len(speech))
	for i := range speech {
		noise20[i] = snr20[i] - speech[i]
	}
	noiseRMS20, _ := CalculateRMS(noise20)
	measuredSNR20 := 20.0 * math.Log10(origRMS/noiseRMS20)
	if math.Abs(measuredSNR20-20.0) > 0.5 {
		t.Errorf("expected SNR ~20 dB, measured %f dB", measuredSNR20)
	}

	// Test with 0 dB SNR pink noise
	snr0 := AddNoiseAtSNR(speech, NoisePink, 0.0, 16000, 42)
	noise0 := make([]float32, len(speech))
	for i := range speech {
		noise0[i] = snr0[i] - speech[i]
	}
	noiseRMS0, _ := CalculateRMS(noise0)
	measuredSNR0 := 20.0 * math.Log10(origRMS/noiseRMS0)
	if math.Abs(measuredSNR0-0.0) > 0.5 {
		t.Errorf("expected SNR ~0 dB, measured %f dB", measuredSNR0)
	}

	// Test Clean returns copy
	clean := AddNoiseAtSNR(speech, NoiseClean, 20.0, 16000, 42)
	for i := range speech {
		if clean[i] != speech[i] {
			t.Errorf("clean noise modified sample %d", i)
			break
		}
	}
}

func TestPerturbationGenerators(t *testing.T) {
	n := 16000
	seed := int64(12345)

	// Typing noise
	typing := GenerateTypingNoise(n, 16000, seed)
	if len(typing) != n {
		t.Fatalf("typing length %d != %d", len(typing), n)
	}
	rmsTyping, _ := CalculateRMS(typing)
	if rmsTyping < 0.01 || rmsTyping > 0.6 {
		t.Errorf("unexpected typing RMS: %f", rmsTyping)
	}

	// Babble noise
	babble := GenerateBabbleNoise(n, 16000, seed)
	if len(babble) != n {
		t.Fatalf("babble length %d != %d", len(babble), n)
	}
	rmsBabble, _ := CalculateRMS(babble)
	if rmsBabble < 0.01 || rmsBabble > 0.6 {
		t.Errorf("unexpected babble RMS: %f", rmsBabble)
	}

	// Reverb
	rev := ApplyReverb(typing, 16000, 0.35, 0.4)
	if len(rev) != n {
		t.Fatalf("reverb length %d != %d", len(rev), n)
	}
	for i, s := range rev {
		if math.IsNaN(float64(s)) || math.IsInf(float64(s), 0) {
			t.Fatalf("reverb produced NaN/Inf at index %d", i)
		}
	}
}
