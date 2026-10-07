package engine

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"voxlab/pkg/audio"
)

// SimulatorEngine simulates ASR, VAD, KWS, and TTS for instant test bench operation.
type SimulatorEngine struct {
	mu               sync.Mutex
	speechChunks     int
	silenceChunks    int
	lastChunkTime    time.Time
	partialIndex     int
	activeTranscript string
	samplePhrases    []string
	dictationPhrases []string
	phraseIdx        int
}

// NewSimulatorEngine creates an initialized testbench simulator.
func NewSimulatorEngine() *SimulatorEngine {
	return &SimulatorEngine{
		samplePhrases: []string{
			"open settings",
			"start recording",
			"dim the screen",
			"show endoscope on the main monitor",
			"what time is lunch", // Distractor
			"volume up",
		},
		dictationPhrases: []string{
			"patient preparation completed successfully",
			"camera one experienced brief interference during movement",
			"biopsy sample collected and labeled at mark two",
		},
	}
}

func (s *SimulatorEngine) Name() string { return "testbench_simulator" }

// DetectVAD simulates voice activity detection based on chunk RMS energy.
func (s *SimulatorEngine) DetectVAD(chunk []float32) (bool, float32) {
	_, dbfs := audio.CalculateRMS(chunk)
	if dbfs > -38.0 {
		prob := float32(0.70 + (dbfs+38.0)*0.015)
		if prob > 0.98 {
			prob = 0.98
		}
		return true, prob
	}
	return false, 0.15
}

// DetectWakeWord simulates keyword spotting.
func (s *SimulatorEngine) DetectWakeWord(chunk []float32) (bool, string, float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, dbfs := audio.CalculateRMS(chunk)
	if dbfs > -32.0 {
		s.speechChunks++
		// If continuous speech sustained for ~15 chunks (~450ms, length of "hey voxlab")
		if s.speechChunks >= 15 {
			s.speechChunks = 0
			return true, "hey voxlab", 0.94
		}
	} else {
		if s.speechChunks > 0 {
			s.speechChunks--
		}
	}
	return false, "", 0.0
}

// ProcessASRChunk simulates streaming token hypotheses and endpointing.
func (s *SimulatorEngine) ProcessASRChunk(chunk []float32, isDictation bool) (*ASRResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, dbfs := audio.CalculateRMS(chunk)
	isVoice := dbfs > -38.0

	if isVoice {
		s.speechChunks++
		s.silenceChunks = 0

		targetPhrase := s.samplePhrases[s.phraseIdx%len(s.samplePhrases)]
		if isDictation {
			targetPhrase = s.dictationPhrases[s.phraseIdx%len(s.dictationPhrases)]
		}

		words := strings.Fields(targetPhrase)
		// Reveal words gradually as speech chunks arrive
		wordsToReveal := (s.speechChunks / 6) + 1
		if wordsToReveal > len(words) {
			wordsToReveal = len(words)
		}
		currentPartial := strings.Join(words[:wordsToReveal], " ")

		return &ASRResult{
			IsFinal:    false,
			Transcript: currentPartial,
			Tokens:     words[:wordsToReveal],
			Confidence: 0.88,
		}, nil
	}

	// Silence chunk
	if s.speechChunks > 0 {
		s.silenceChunks++
		// 800ms trailing silence = ~26 chunks of 30ms
		if s.silenceChunks >= 20 {
			targetPhrase := s.samplePhrases[s.phraseIdx%len(s.samplePhrases)]
			if isDictation {
				targetPhrase = s.dictationPhrases[s.phraseIdx%len(s.dictationPhrases)]
			}
			s.phraseIdx++
			s.speechChunks = 0
			s.silenceChunks = 0

			return &ASRResult{
				IsFinal:    true,
				Transcript: targetPhrase,
				Tokens:     strings.Fields(targetPhrase),
				Confidence: 0.93,
			}, nil
		}
	}

	return nil, nil
}

// ResetASR clears accumulated streaming state.
func (s *SimulatorEngine) ResetASR() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.speechChunks = 0
	s.silenceChunks = 0
}

// Synthesize simulates Kokoro TTS by generating rich harmonic vocalic waveforms.
func (s *SimulatorEngine) Synthesize(req TTSRequest) (*TTSResult, error) {
	startTime := time.Now()

	sampleRate := req.SampleRate
	if sampleRate <= 0 {
		sampleRate = 24000
	}

	// Approximate speech duration: ~80ms per character with speed adjustment
	charCount := len(req.Text)
	if charCount < 5 {
		charCount = 5
	}
	speed := req.Speed
	if speed <= 0 {
		speed = 1.0
	}
	durationSec := (float64(charCount) * 0.065) / speed
	if durationSec < 0.8 {
		durationSec = 0.8
	}

	totalSamples := int(float64(sampleRate) * durationSec)
	samples := make([]float32, totalSamples)
	sr := float64(sampleRate)

	// Voice pitch variation based on speaker name
	basePitch := 220.0 // Default female pitch (e.g. af_heart)
	if strings.HasPrefix(req.Voice, "am_") {
		basePitch = 130.0 // Male pitch (e.g. am_adam)
	}

	// Formant synthesis simulation
	f1 := basePitch * 2.2
	f2 := basePitch * 3.8

	for i := 0; i < totalSamples; i++ {
		t := float64(i) / sr
		// Amplitude envelope with soft attack and release
		env := 1.0
		if t < 0.05 {
			env = t / 0.05
		} else if t > durationSec-0.1 {
			env = (durationSec - t) / 0.1
		}
		// Pitch cadence vibrato
		pitch := basePitch + 15.0*math.Sin(2.0*math.Pi*2.5*t)
		v0 := math.Sin(2.0 * math.Pi * pitch * t)
		v1 := 0.4 * math.Sin(2.0*math.Pi*f1*t)
		v2 := 0.2 * math.Sin(2.0*math.Pi*f2*t)

		// Modulation
		samples[i] = float32(0.35 * env * (v0 + v1 + v2))
	}

	wavBytes, err := audio.EncodeWAV(samples, sampleRate)
	if err != nil {
		return nil, fmt.Errorf("failed to encode WAV: %w", err)
	}

	latency := time.Since(startTime).Milliseconds()

	return &TTSResult{
		AudioSamples: samples,
		WAVBytes:     wavBytes,
		SampleRate:   sampleRate,
		DurationSec:  durationSec,
		LatencyMs:    latency,
	}, nil
}

// TTSModelInfo returns simulator model information.
func (s *SimulatorEngine) TTSModelInfo() (string, bool) {
	return "Harmonic Simulator (Mock)", false
}
