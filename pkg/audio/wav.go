package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// EncodeWAV encodes 16-bit PCM mono samples into a WAV byte slice.
func EncodeWAV(samples []float32, sampleRate int) ([]byte, error) {
	intSamples := Float32ToInt16(samples)
	dataSize := uint32(len(intSamples) * 2)
	totalSize := 36 + dataSize

	buf := new(bytes.Buffer)

	// RIFF header
	buf.WriteString("RIFF")
	if err := binary.Write(buf, binary.LittleEndian, totalSize); err != nil {
		return nil, err
	}
	buf.WriteString("WAVE")

	// fmt chunk
	buf.WriteString("fmt ")
	if err := binary.Write(buf, binary.LittleEndian, uint32(16)); err != nil { // Subchunk1Size (16 for PCM)
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil { // AudioFormat (1 for PCM)
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil { // NumChannels (1 mono)
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(sampleRate)); err != nil { // SampleRate
		return nil, err
	}
	byteRate := uint32(sampleRate * 1 * 2)
	if err := binary.Write(buf, binary.LittleEndian, byteRate); err != nil { // ByteRate
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(2)); err != nil { // BlockAlign (NumChannels * BitsPerSample/8)
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(16)); err != nil { // BitsPerSample (16)
		return nil, err
	}

	// data chunk
	buf.WriteString("data")
	if err := binary.Write(buf, binary.LittleEndian, dataSize); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, intSamples); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DecodeWAV decodes a WAV byte stream into float32 mono samples and sample rate.
func DecodeWAV(r io.Reader) (samples []float32, sampleRate int, err error) {
	var riffHeader [4]byte
	if _, err := io.ReadFull(r, riffHeader[:]); err != nil {
		return nil, 0, err
	}
	if string(riffHeader[:]) != "RIFF" {
		return nil, 0, errors.New("not a valid RIFF WAV file")
	}

	var fileSize uint32
	if err := binary.Read(r, binary.LittleEndian, &fileSize); err != nil {
		return nil, 0, err
	}

	var waveHeader [4]byte
	if _, err := io.ReadFull(r, waveHeader[:]); err != nil {
		return nil, 0, err
	}
	if string(waveHeader[:]) != "WAVE" {
		return nil, 0, errors.New("missing WAVE format marker")
	}

	var audioFormat, numChannels, bitsPerSample uint16
	var sRate uint32
	var dataBytes []byte

	// Read chunks until the data chunk. Chunks are padded to an even size.
	for dataBytes == nil {
		var chunkID [4]byte
		var chunkSize uint32
		if _, err := io.ReadFull(r, chunkID[:]); err != nil {
			break
		}
		if err := binary.Read(r, binary.LittleEndian, &chunkSize); err != nil {
			return nil, 0, err
		}
		switch string(chunkID[:]) {
		case "fmt ":
			if chunkSize < 16 {
				return nil, 0, errors.New("WAV fmt chunk too short")
			}
			var f struct {
				AudioFormat, NumChannels uint16
				SampleRate, ByteRate     uint32
				BlockAlign, Bits         uint16
			}
			if err := binary.Read(r, binary.LittleEndian, &f); err != nil {
				return nil, 0, err
			}
			audioFormat, numChannels, sRate, bitsPerSample = f.AudioFormat, f.NumChannels, f.SampleRate, f.Bits
			if _, err := io.CopyN(io.Discard, r, int64(chunkSize-16+chunkSize&1)); err != nil {
				return nil, 0, err
			}
		case "data":
			// Read what is there: streamed WAVs often state a size larger than the file.
			if dataBytes, err = io.ReadAll(io.LimitReader(r, int64(chunkSize))); err != nil {
				return nil, 0, err
			}
		default:
			if _, err := io.CopyN(io.Discard, r, int64(chunkSize)+int64(chunkSize&1)); err != nil {
				return nil, 0, errors.New("truncated WAV chunk " + string(chunkID[:]))
			}
		}
	}

	if dataBytes == nil {
		return nil, 0, errors.New("no data chunk found in WAV")
	}
	// 0xFFFE is WAVE_FORMAT_EXTENSIBLE; at 16 bits its subformat can only be PCM.
	if (audioFormat != 1 && audioFormat != 0xFFFE) || bitsPerSample != 16 {
		return nil, 0, errors.New("only 16-bit PCM WAV currently supported")
	}
	if numChannels == 0 || sRate == 0 {
		return nil, 0, errors.New("WAV fmt chunk has no channels or sample rate")
	}

	intSamples := BytesToInt16PCM(dataBytes)
	if ch := int(numChannels); ch > 1 {
		// Downmix to mono
		mono := make([]int16, len(intSamples)/ch)
		for i := range mono {
			var sum int32
			for _, v := range intSamples[i*ch : i*ch+ch] {
				sum += int32(v)
			}
			mono[i] = int16(sum / int32(ch))
		}
		intSamples = mono
	}

	return Int16ToFloat32(intSamples), int(sRate), nil
}

// GenerateChime generates a pleasant synthetic chime tone for cues or test audio.
func GenerateChime(sampleRate int, freq1, freq2 float64, durationSec float64) []float32 {
	totalSamples := int(float64(sampleRate) * durationSec)
	out := make([]float32, totalSamples)
	sr := float64(sampleRate)

	for i := 0; i < totalSamples; i++ {
		t := float64(i) / sr
		// Smooth exponential decay envelope
		envelope := math.Exp(-4.0 * t / durationSec)
		// Dual frequency harmonic chord
		s1 := math.Sin(2.0 * math.Pi * freq1 * t)
		s2 := 0.5 * math.Sin(2.0*math.Pi*freq2*t)
		out[i] = float32(0.4 * envelope * (s1 + s2))
	}
	return out
}

// Resample converts mono float32 audio between sample rates: linear interpolation when
// upsampling, a Hann-windowed sinc low-pass at 0.45 x toRate when downsampling (no aliasing).
func Resample(samples []float32, fromRate, toRate int) []float32 {
	if fromRate == toRate || len(samples) == 0 || fromRate <= 0 || toRate <= 0 {
		return samples
	}
	outLen := int(math.Round(float64(len(samples)) * float64(toRate) / float64(fromRate)))
	if outLen == 0 {
		return nil
	}
	out := make([]float32, outLen)
	ratio := float64(fromRate) / float64(toRate)
	n := len(samples)
	if ratio > 1 {
		cut := 0.45 / ratio             // cutoff in cycles per input sample
		half := int(math.Ceil(4 / cut)) // 8 zero crossings each side
		for i := range out {
			c := float64(i) * ratio
			var acc, wsum float64
			for j := max(0, int(c)-half); j <= min(n-1, int(c)+half); j++ {
				x := float64(j) - c
				h := 2 * cut
				if x != 0 {
					h = math.Sin(2*math.Pi*cut*x) / (math.Pi * x)
				}
				h *= 0.5 + 0.5*math.Cos(math.Pi*x/float64(half+1))
				acc += h * float64(samples[j])
				wsum += h
			}
			if wsum != 0 {
				out[i] = float32(acc / wsum) // unity gain at DC, also at the edges
			}
		}
		return out
	}
	for i := 0; i < outLen; i++ {
		srcPos := float64(i) * ratio
		srcIdx := int(srcPos)
		frac := float32(srcPos - float64(srcIdx))
		if srcIdx+1 < n {
			out[i] = samples[srcIdx]*(1.0-frac) + samples[srcIdx+1]*frac
		} else if srcIdx < n {
			out[i] = samples[srcIdx]
		}
	}
	return out
}
