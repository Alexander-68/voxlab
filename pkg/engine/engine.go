package engine

// ASRResult holds streaming or final speech recognition output.
type ASRResult struct {
	IsFinal    bool     `json:"is_final"`
	Transcript string   `json:"transcript"`
	Tokens     []string `json:"tokens,omitempty"`
	Confidence float64  `json:"confidence,omitempty"`
}

// TTSRequest holds parameters for Kokoro text-to-speech synthesis.
type TTSRequest struct {
	Text       string  `json:"text"`
	Voice      string  `json:"voice"`
	Speed      float64 `json:"speed"`
	SampleRate int     `json:"sample_rate"`
}

// TTSResult holds synthesized audio waveform and metrics.
type TTSResult struct {
	AudioSamples []float32 `json:"-"`
	WAVBytes     []byte    `json:"-"`
	SampleRate   int       `json:"sample_rate"`
	DurationSec  float64   `json:"duration_sec"`
	LatencyMs    int64     `json:"latency_ms"`
}

// SpeechEngine defines the interface for VAD, KWS, ASR, and TTS.
type SpeechEngine interface {
	Name() string
	DetectVAD(chunk []float32) (isVoice bool, prob float32)
	DetectWakeWord(chunk []float32) (detected bool, keyword string, confidence float64)
	ProcessASRChunk(chunk []float32, isDictation bool) (*ASRResult, error)
	ResetASR()
	Synthesize(req TTSRequest) (*TTSResult, error)
}
