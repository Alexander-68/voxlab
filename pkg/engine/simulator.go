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
	activeModel      string
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
	return s.SynthesizeStream(req, nil)
}

// SynthesizeStream generates speech audio incrementally for low-latency streaming playback.
func (s *SimulatorEngine) SynthesizeStream(req TTSRequest, onChunk func(chunk TTSChunk) error) (*TTSResult, error) {
	startTime := time.Now()

	sampleRate := req.SampleRate
	if sampleRate <= 0 {
		sampleRate = 24000
	}

	speed := req.Speed
	if speed <= 0 {
		speed = 1.0
	}

	cleanText := strings.TrimSpace(req.Text)
	if cleanText == "" {
		cleanText = "Hello"
	}

	// Split text into natural phrase chunks by punctuation (. , ! ? ;) for streaming
	var parts []string
	var current strings.Builder
	for _, r := range cleanText {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' || r == ';' || r == ',' {
			p := strings.TrimSpace(current.String())
			if len(p) > 0 {
				parts = append(parts, p)
			}
			current.Reset()
		}
	}
	if rem := strings.TrimSpace(current.String()); len(rem) > 0 {
		parts = append(parts, rem)
	}
	if len(parts) == 0 {
		parts = []string{cleanText}
	}

	var allSamples []float32
	basePitch := 220.0
	if strings.HasPrefix(req.Voice, "am_") {
		basePitch = 130.0
	}
	f1 := basePitch * 2.2
	f2 := basePitch * 3.8
	sr := float64(sampleRate)

	for idx, part := range parts {
		charCount := len(part)
		if charCount < 3 {
			charCount = 3
		}
		durationSec := (float64(charCount) * 0.065) / speed
		if durationSec < 0.4 {
			durationSec = 0.4
		}

		chunkTotalSamples := int(float64(sampleRate) * durationSec)
		chunkSamples := make([]float32, chunkTotalSamples)

		for i := 0; i < chunkTotalSamples; i++ {
			t := float64(i) / sr
			env := 1.0
			if t < 0.05 {
				env = t / 0.05
			} else if t > durationSec-0.08 {
				env = (durationSec - t) / 0.08
			}
			pitch := basePitch + 15.0*math.Sin(2.0*math.Pi*2.5*t)
			v0 := math.Sin(2.0 * math.Pi * pitch * t)
			v1 := 0.4 * math.Sin(2.0*math.Pi*f1*t)
			v2 := 0.2 * math.Sin(2.0*math.Pi*f2*t)
			chunkSamples[i] = float32(0.35 * env * (v0 + v1 + v2))
		}

		allSamples = append(allSamples, chunkSamples...)

		if onChunk != nil {
			wavBytes, err := audio.EncodeWAV(chunkSamples, sampleRate)
			if err != nil {
				return nil, fmt.Errorf("failed to encode chunk WAV: %w", err)
			}
			lat := time.Since(startTime).Milliseconds()
			isLast := idx == len(parts)-1
			c := TTSChunk{
				Index:        idx,
				IsLast:       isLast,
				AudioSamples: chunkSamples,
				WAVBytes:     wavBytes,
				SampleRate:   sampleRate,
				DurationSec:  durationSec,
				LatencyMs:    lat,
			}
			if err := onChunk(c); err != nil {
				return nil, err
			}
		}
	}

	totalWav, err := audio.EncodeWAV(allSamples, sampleRate)
	if err != nil {
		return nil, fmt.Errorf("failed to encode WAV: %w", err)
	}

	totalDuration := float64(len(allSamples)) / float64(sampleRate)
	latency := time.Since(startTime).Milliseconds()

	return &TTSResult{
		AudioSamples: allSamples,
		WAVBytes:     totalWav,
		SampleRate:   sampleRate,
		DurationSec:  totalDuration,
		LatencyMs:    latency,
	}, nil
}

// TTSModelInfo returns simulator model information.
func (s *SimulatorEngine) TTSModelInfo() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeModel != "" {
		return s.activeModel, false
	}
	return "Harmonic Simulator (Mock)", false
}

// SetTTSModel updates simulator model.
func (s *SimulatorEngine) SetTTSModel(modelName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeModel = modelName
	return nil
}

// InstalledTTSModels returns available installed models or default mock.
func (s *SimulatorEngine) InstalledTTSModels() []string {
	models := FindInstalledKokoroModels("models")
	if len(models) == 0 {
		return []string{"simulator-mock"}
	}
	return models
}
