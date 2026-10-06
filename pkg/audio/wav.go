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

	// Read chunks until data chunk found
	for {
		var chunkID [4]byte
		var chunkSize uint32
		if _, err := io.ReadFull(r, chunkID[:]); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, 0, err
		}
		if err := binary.Read(r, binary.LittleEndian, &chunkSize); err != nil {
			return nil, 0, err
		}

		cID := string(chunkID[:])
		if cID == "fmt " {
			if err := binary.Read(r, binary.LittleEndian, &audioFormat); err != nil {
				return nil, 0, err
			}
			if err := binary.Read(r, binary.LittleEndian, &numChannels); err != nil {
				return nil, 0, err
			}
			if err := binary.Read(r, binary.LittleEndian, &sRate); err != nil {
				return nil, 0, err
			}
			var byteRate uint32
			var blockAlign uint16
			_ = binary.Read(r, binary.LittleEndian, &byteRate)
			_ = binary.Read(r, binary.LittleEndian, &blockAlign)
			if err := binary.Read(r, binary.LittleEndian, &bitsPerSample); err != nil {
				return nil, 0, err
			}
			// Skip any extra format bytes
			if chunkSize > 16 {
				skip := make([]byte, chunkSize-16)
				_, _ = io.ReadFull(r, skip)
			}
		} else if cID == "data" {
			dataBytes = make([]byte, chunkSize)
			if _, err := io.ReadFull(r, dataBytes); err != nil {
				return nil, 0, err
			}
			break
		} else {
			// Skip unknown chunk
			skip := make([]byte, chunkSize)
			_, _ = io.ReadFull(r, skip)
		}
	}

	if dataBytes == nil {
		return nil, 0, errors.New("no data chunk found in WAV")
	}

	// Convert 16-bit PCM to float32 mono
	if bitsPerSample != 16 {
		return nil, 0, errors.New("only 16-bit PCM WAV currently supported")
	}

	intSamples := BytesToInt16PCM(dataBytes)
	if numChannels == 2 {
		// Downmix stereo to mono
		monoCount := len(intSamples) / 2
		monoS := make([]int16, monoCount)
		for i := 0; i < monoCount; i++ {
			left := int32(intSamples[i*2])
			right := int32(intSamples[i*2+1])
			monoS[i] = int16((left + right) / 2)
		}
		intSamples = monoS
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
		s2 := 0.5 * math.Sin(2.0 * math.Pi * freq2 * t)
		out[i] = float32(0.4 * envelope * (s1 + s2))
	}
	return out
}
