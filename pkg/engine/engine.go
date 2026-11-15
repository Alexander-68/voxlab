package engine

// ASRResult holds streaming or final speech recognition output.
type ASRResult struct {
	IsFinal    bool     `json:"is_final"`
	Transcript string   `json:"transcript"`
	Tokens     []string `json:"tokens,omitempty"`
	Confidence float64  `json:"confidence,omitempty"`
	StartSec   float64  `json:"-"` // first/last token time within the ASR stream (0 = unknown)
	EndSec     float64  `json:"-"`
}

// TTSRequest holds parameters for Kokoro text-to-speech synthesis.
type TTSRequest struct {
	Text       string  `json:"text"`
	Voice      string  `json:"voice"`
	Speed      float64 `json:"speed"`
	SampleRate int     `json:"sample_rate"`
	Lang       string  `json:"lang,omitempty"` // "" follows the voice; "en" reads English text in a foreign voice
}

// TTSResult holds synthesized audio waveform and metrics.
type TTSResult struct {
	AudioSamples []float32 `json:"-"`
	WAVBytes     []byte    `json:"-"`
	SampleRate   int       `json:"sample_rate"`
	DurationSec  float64   `json:"duration_sec"`
	LatencyMs    int64     `json:"latency_ms"`
	Simulated    bool      `json:"simulated"` // tones from the simulator (including SherpaRunner fallbacks), not speech
}

// TTSChunk represents an incrementally synthesized audio fragment for low-latency streaming playback.
type TTSChunk struct {
	Index        int       `json:"index"`
	IsLast       bool      `json:"is_last"`
	AudioSamples []float32 `json:"-"`
	WAVBytes     []byte    `json:"-"`
	SampleRate   int       `json:"sample_rate"`
	DurationSec  float64   `json:"duration_sec"`
	LatencyMs    int64     `json:"latency_ms"`
}

// SpeechEngine defines the interface for VAD, KWS, ASR, and TTS.
// The ASR methods share streaming state; callers must serialize
// ProcessASRChunk/FlushASR/ResetASR. Implementations that own subprocesses or
// native handles should also implement io.Closer (Server.Stop closes them).
type SpeechEngine interface {
	Name() string
	DetectVAD(chunk []float32) (isVoice bool, prob float32)
	DetectWakeWord(chunk []float32) (detected bool, keyword string, confidence float64)
	ProcessASRChunk(chunk []float32, isDictation bool) (*ASRResult, error)
	// FlushASR finalizes and consumes the in-progress utterance and returns its text
	// ("" when nothing was pending). It may block while the engine drains pending audio.
	FlushASR() string
	ResetASR()
	Synthesize(req TTSRequest) (*TTSResult, error)
	// SynthesizeStream emits audio through onChunk; a non-nil onChunk error aborts
	// synthesis and is returned (wrapped) to the caller.
	SynthesizeStream(req TTSRequest, onChunk func(chunk TTSChunk) error) (*TTSResult, error)
	TTSModelInfo() (modelName string, isNeural bool)
	SetTTSModel(modelName string) error
	InstalledTTSModels() []string
}
