package audio

import (
	"bytes"
	"encoding/binary"
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

	sine := func(amp float64) []float32 {
		c := make([]float32, 480)
		for i := range c {
			c[i] = float32(amp * math.Sin(2.0*math.Pi*440.0*float64(i)/16000.0))
		}
		return c
	}
	lowChunk := sine(0.003)  // ~-53 dBFS
	speechChunk := sine(0.1) // ~-23 dBFS

	// Fresh gate, quiet input: closed, attenuated to the floor (not zeroed) once the ramp settles
	if _, _, dbfs, passed := dsp.ProcessChunk(lowChunk); passed {
		t.Errorf("expected low noise chunk to be gated, but passed with dbfs: %f", dbfs)
	}
	outLow, _, _, _ := dsp.ProcessChunk(lowChunk)
	inRMS, _ := CalculateRMS(lowChunk)
	outRMS, _ := CalculateRMS(outLow)
	if outRMS == 0 || outRMS > inRMS*dsp.gateFloor*1.2 {
		t.Errorf("expected gated chunk attenuated to ~%.2fx, got %.3fx", dsp.gateFloor, outRMS/inRMS)
	}

	if _, _, _, passedSpeech := dsp.ProcessChunk(speechChunk); !passedSpeech {
		t.Errorf("expected speech chunk to pass noise gate")
	}

	// Hold: quiet chunks right after speech stay open for ~300ms (10 chunks), then close
	for i := 0; i < 10; i++ {
		if _, _, _, passed := dsp.ProcessChunk(lowChunk); !passed {
			t.Fatalf("expected gate held open on quiet chunk %d after speech", i)
		}
	}
	if _, _, _, passed := dsp.ProcessChunk(lowChunk); passed {
		t.Errorf("expected gate to close after hold expired")
	}

	// Test half-duplex echo mute
	dsp.SetEchoMuted(true)
	_, _, _, passedDuringEcho := dsp.ProcessChunk(speechChunk)
	if passedDuringEcho {
		t.Errorf("expected chunk to be muted during echo mute")
	}

	// Test threshold <= GateOffDBFS completely turns gate OFF
	dsp.SetEchoMuted(false)
	dsp.SetGateThreshold(GateOffDBFS)
	if dsp.IsGateEnabled() {
		t.Errorf("expected gate to be disabled when threshold <= GateOffDBFS")
	}
	outNoise, _, _, passedOff := dsp.ProcessChunk(lowChunk)
	if !passedOff {
		t.Errorf("expected low noise chunk to pass when gate is OFF")
	}
	for i, v := range outNoise {
		if v == 0 {
			t.Errorf("expected noise chunk samples to remain untouched when gate is OFF at %d", i)
			break
		}
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
	dsp := NewDSPProcessor(16000, 80.0, GateOffDBFS, true, 0.2)

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

func TestResample(t *testing.T) {
	// 1 second of 440Hz sine wave at 24000 Hz
	inRate := 24000
	outRate := 16000
	dur := 1.0
	inSamples := make([]float32, int(float64(inRate)*dur))
	for i := range inSamples {
		inSamples[i] = float32(math.Sin(2.0 * math.Pi * 440.0 * float64(i) / float64(inRate)))
	}

	res := Resample(inSamples, inRate, outRate)
	expectedLen := int(float64(outRate) * dur)
	if len(res) != expectedLen {
		t.Fatalf("expected resampled length %d, got %d", expectedLen, len(res))
	}

	// Downsampling keeps a 1 kHz tone and suppresses an 11 kHz one, which would alias to 5 kHz.
	rms := func(freq float64) float64 {
		in := make([]float32, 48000)
		for i := range in {
			in[i] = float32(math.Sin(2 * math.Pi * freq * float64(i) / 48000))
		}
		out := Resample(in, 48000, 16000)
		var sum float64
		for _, v := range out[1000:15000] {
			sum += float64(v) * float64(v)
		}
		return math.Sqrt(sum / 14000)
	}
	if r := rms(1000); r < 0.68 || r > 0.73 {
		t.Errorf("1 kHz tone RMS after downsampling = %.3f, want about 0.707", r)
	}
	if r := rms(11000); r > 0.01 {
		t.Errorf("11 kHz tone RMS after downsampling = %.3f, want < 0.01 (aliasing)", r)
	}

	// Verify identity resampling (same rate)
	same := Resample(inSamples, inRate, inRate)
	if len(same) != len(inSamples) {
		t.Errorf("expected same length for identical rates")
	}

	// Verify empty slice
	empty := Resample(nil, inRate, outRate)
	if empty != nil {
		t.Errorf("expected nil for empty slice")
	}
}

// TestDecodeWAVEdgeCases: an odd-sized chunk with its pad byte before fmt, stereo, and a data
// size larger than the file (as streamed WAVs write it).
func TestDecodeWAVEdgeCases(t *testing.T) {
	le := binary.LittleEndian
	var b []byte
	b = append(b, "RIFF\x00\x00\x00\x00WAVE"...)
	b = append(b, "LIST\x03\x00\x00\x00abc\x00"...)
	b = append(b, "fmt \x10\x00\x00\x00"...)
	b = le.AppendUint16(b, 1)
	b = le.AppendUint16(b, 2)
	b = le.AppendUint32(b, 8000)
	b = le.AppendUint32(b, 32000)
	b = le.AppendUint16(b, 4)
	b = le.AppendUint16(b, 16)
	b = append(b, "data\xff\xff\xff\xff"...)
	for _, v := range []int16{1000, 3000, -2000, -4000} {
		b = le.AppendUint16(b, uint16(v))
	}
	s, rate, err := DecodeWAV(bytes.NewReader(b))
	if err != nil || rate != 8000 || len(s) != 2 {
		t.Fatalf("DecodeWAV = %d samples, rate %d, err %v; want 2 samples at 8000", len(s), rate, err)
	}
	if want := Int16ToFloat32([]int16{2000, -3000}); s[0] != want[0] || s[1] != want[1] {
		t.Errorf("downmix = %v, want %v", s, want)
	}
}
