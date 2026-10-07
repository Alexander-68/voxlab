package audio

import (
	"bytes"
	"math"
	"testing"
)

func TestCalculateRMS(t *testing.T) {
	// Zero audio
	zeros := make([]float32, 100)
	rms, dbfs := CalculateRMS(zeros)
	if rms != 0.0 || dbfs != -100.0 {
		t.Errorf("expected 0 RMS and -100 dBFS for zeros, got %f, %f", rms, dbfs)
	}

	// Full scale DC (1.0)
	ones := make([]float32, 100)
	for i := range ones {
		ones[i] = 1.0
	}
	rms, dbfs = CalculateRMS(ones)
	if math.Abs(rms-1.0) > 1e-5 || math.Abs(dbfs-0.0) > 1e-4 {
		t.Errorf("expected 1.0 RMS and 0.0 dBFS, got %f, %f", rms, dbfs)
	}
}

func TestRingBuffer(t *testing.T) {
	rb := NewRingBuffer(5)

	// Write 3 samples
	rb.Write([]float32{1, 2, 3})
	if rb.Size() != 3 {
		t.Errorf("expected size 3, got %d", rb.Size())
	}
	snap := rb.ReadSnapshot()
	if len(snap) != 3 || snap[0] != 1 || snap[1] != 2 || snap[2] != 3 {
		t.Errorf("unexpected snapshot: %v", snap)
	}

	// Write 4 more samples (overflow capacity 5, total 7 written)
	rb.Write([]float32{4, 5, 6, 7})
	if rb.Size() != 5 {
		t.Errorf("expected size 5, got %d", rb.Size())
	}
	snap2 := rb.ReadSnapshot()
	// Oldest samples (1, 2) overwritten, remaining should be [3, 4, 5, 6, 7]
	expected := []float32{3, 4, 5, 6, 7}
	for i, v := range expected {
		if snap2[i] != v {
			t.Errorf("at index %d: expected %f, got %f", i, v, snap2[i])
		}
	}
}

func TestDSPNoiseGateAndEcho(t *testing.T) {
	dsp := NewDSPProcessor(16000, 80.0, -40.0, false, 0.1)

	// Low amplitude noise (-50 dBFS)
	lowChunk := make([]float32, 480)
	for i := range lowChunk {
		lowChunk[i] = 0.001
	}

	out, _, dbfs, passed := dsp.ProcessChunk(lowChunk)
	if passed {
		t.Errorf("expected low noise chunk to be gated, but passed with dbfs: %f", dbfs)
	}
	for i, v := range out {
		if v != 0 {
			t.Errorf("expected gated sample to be 0 at %d, got %f", i, v)
		}
	}

	// Speech amplitude chunk (-20 dBFS)
	speechChunk := make([]float32, 480)
	for i := range speechChunk {
		speechChunk[i] = float32(0.1 * math.Sin(2.0*math.Pi*440.0*float64(i)/16000.0))
	}
	_, _, _, passedSpeech := dsp.ProcessChunk(speechChunk)
	if !passedSpeech {
		t.Errorf("expected speech chunk to pass noise gate")
	}

	// Test half-duplex echo mute
	dsp.SetEchoMuted(true)
	_, _, _, passedDuringEcho := dsp.ProcessChunk(speechChunk)
	if passedDuringEcho {
		t.Errorf("expected chunk to be muted during echo mute")
	}
}

func TestWAVEncodeDecode(t *testing.T) {
	sampleRate := 16000
	samples := GenerateChime(sampleRate, 440, 880, 0.1)

	wavBytes, err := EncodeWAV(samples, sampleRate)
	if err != nil {
		t.Fatalf("failed to encode WAV: %v", err)
	}

	decoded, sRate, err := DecodeWAV(bytes.NewReader(wavBytes))
	if err != nil {
		t.Fatalf("failed to decode WAV: %v", err)
	}

	if sRate != sampleRate {
		t.Errorf("expected sample rate %d, got %d", sampleRate, sRate)
	}

	if len(decoded) != len(samples) {
		t.Errorf("expected %d samples, got %d", len(samples), len(decoded))
	}
}

func TestDSPHighPassAndAGC(t *testing.T) {
	dsp := NewDSPProcessor(16000, 80.0, -60.0, true, 0.2)

	if !dsp.IsHighPassEnabled() {
		t.Errorf("expected highpass to be enabled by default")
	}
	if !dsp.IsAGCEnabled() {
		t.Errorf("expected AGC to be enabled")
	}

	// Disable highpass
	dsp.SetHighPassEnabled(false)
	if dsp.IsHighPassEnabled() {
		t.Errorf("expected highpass to be disabled")
	}

	// DC signal: without highpass, output should preserve DC
	dcChunk := make([]float32, 100)
	for i := range dcChunk {
		dcChunk[i] = 0.05
	}
	out, _, _, _ := dsp.ProcessChunk(dcChunk)
	if math.Abs(float64(out[50])) < 0.01 {
		t.Errorf("expected DC to pass through when highpass is disabled")
	}

	// Re-enable highpass: DC should be attenuated towards 0
	dsp.SetHighPassEnabled(true)
	for k := 0; k < 5; k++ {
		out, _, _, _ = dsp.ProcessChunk(dcChunk)
	}
	// High-pass filter removes constant DC
	if math.Abs(float64(out[len(out)-1])) > 0.02 {
		t.Errorf("expected DC to be attenuated by highpass filter, got %f", out[len(out)-1])
	}

	// Test AGC toggle and CurrentGain
	dsp.SetAGCEnabled(false)
	if dsp.IsAGCEnabled() {
		t.Errorf("expected AGC to be disabled")
	}
	if dsp.CurrentGain() != 1.0 {
		t.Errorf("expected gain to be 1.0 when AGC is disabled")
	}
}

