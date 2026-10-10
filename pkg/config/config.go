package config

import (
	"encoding/json"
	"os"
)

// AudioConfig defines audio format and DSP parameters.
type AudioConfig struct {
	SampleRate            int     `json:"sample_rate"`              // 16000 Hz standard for ASR
	Channels              int     `json:"channels"`                 // 1 (mono)
	ChunkSamples          int     `json:"chunk_samples"`            // 480 samples = 30ms @ 16kHz
	NoiseGateThresholdDBFS float64 `json:"noise_gate_threshold_dbfs"` // e.g. -42.0 dBFS
	HighPassCutoffHz      float64 `json:"high_pass_cutoff_hz"`      // e.g. 80.0 Hz
	AGCEnabled            bool           `json:"agc_enabled"`
	TargetRMS             float64        `json:"target_rms"`               // e.g. 0.12 (-18 dBFS)
	Enhancer              EnhancerConfig `json:"enhancer"`
}

// EnhancerConfig defines neural speech enhancement / denoising parameters.
type EnhancerConfig struct {
	Enabled    bool   `json:"enabled"`               // Whether neural denoising is active (default false)
	ModelType  string `json:"model_type"`            // "gtcrn" or "dpdfnet" (default "gtcrn")
	ModelPath  string `json:"model_path,omitempty"`  // Path to model, e.g. "models/gtcrn_simple.onnx"
	NumThreads int    `json:"num_threads,omitempty"` // Number of threads (default 1)
	Provider   string `json:"provider,omitempty"`    // Execution provider: "cpu"
}

// KWSConfig defines keyword spotting parameters.
type KWSConfig struct {
	Enabled             bool    `json:"enabled"`
	Keyword             string  `json:"keyword"`              // e.g. "hey voxlab"
	Threshold           float64 `json:"threshold"`            // e.g. 0.40
	PreRollDurationSec  float64 `json:"pre_roll_duration_sec"`// e.g. 1.0s circular buffer
}

// IntentConfig defines semantic matching and distractor suppression thresholds.
type IntentConfig struct {
	ConfidenceThreshold float64 `json:"confidence_threshold"` // e.g. 0.80
	MarginThreshold     float64 `json:"margin_threshold"`     // e.g. 0.12 (top1 - top2)
	CatalogPath         string  `json:"catalog_path"`         // e.g. "data/commands.json"
}

// DictationConfig defines voice annotation settings.
type DictationConfig struct {
	SilenceEndpointMs int     `json:"silence_endpoint_ms"` // e.g. 800ms trailing silence
	MaxDurationSec    float64 `json:"max_duration_sec"`    // e.g. 30.0s session cap
}

// EngineConfig configures Sherpa-ONNX model paths and binaries.
type EngineConfig struct {
	Mode           string `json:"mode"`             // "simulator" or "sherpa"
	ModelDir       string `json:"model_dir"`        // e.g. "models"
	SherpaTtsBin   string `json:"sherpa_tts_bin"`   // e.g. "sherpa-onnx-offline-tts"
	SherpaKwsBin   string `json:"sherpa_kws_bin"`   // e.g. "sherpa-onnx-keyword-spotter"
	SherpaAsrBin   string `json:"sherpa_asr_bin"`   // e.g. "sherpa-onnx-online-websocket-server"
	KokoroModelDir  string `json:"kokoro_model_dir"`  // e.g. "models/kokoro-multi-lang-v1_0"
	KokoroModelFile string `json:"kokoro_model_file,omitempty"` // e.g. "model.fp16.onnx", "model.onnx"
	ZipformerDir    string `json:"zipformer_dir"`     // e.g. "models/sherpa-onnx-streaming-zipformer-en-2023-06-26"
	NumThreads      int    `json:"num_threads,omitempty"`     // Number of inference threads (default 4)
	Provider        string `json:"provider,omitempty"`        // Execution provider: "cpu", "directml", "cuda" (default "cpu")
}

// TTSConfig defines text-to-speech parameters.
type TTSConfig struct {
	DefaultVoice string  `json:"default_voice"` // e.g. "af_heart"
	DefaultSpeed float64 `json:"default_speed"` // e.g. 1.0
	SampleRate   int     `json:"sample_rate"`   // 24000 Hz for Kokoro
	EchoMute     bool    `json:"echo_mute"`     // Half-duplex echo cancellation
}

// Version is the current application version formatted as 1.0.yymmdd.
const Version = "1.0.261010"

// AppConfig is the root configuration struct.
type AppConfig struct {
	Version     string          `json:"version"`
	Host        string          `json:"host"`
	Port        int             `json:"port"`
	StaticDir   string          `json:"static_dir"`
	Audio       AudioConfig     `json:"audio"`
	KWS         KWSConfig       `json:"kws"`
	Intent      IntentConfig    `json:"intent"`
	Dictation   DictationConfig `json:"dictation"`
	Engine      EngineConfig    `json:"engine"`
	TTS         TTSConfig       `json:"tts"`
}

// DefaultConfig returns the default production-ready configuration.
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Version:   Version,
		Host:      "0.0.0.0",
		Port:      8080,
		StaticDir: "web",
		Audio: AudioConfig{
			SampleRate:            16000,
			Channels:              1,
			ChunkSamples:          480, // 30ms
			NoiseGateThresholdDBFS: -42.0,
			HighPassCutoffHz:      80.0,
			AGCEnabled:            true,
			TargetRMS:             0.12,
			Enhancer: EnhancerConfig{
				Enabled:    false,
				ModelType:  "gtcrn",
				ModelPath:  "models/gtcrn_simple.onnx",
				NumThreads: 1,
				Provider:   "cpu",
			},
		},
		KWS: KWSConfig{
			Enabled:            false,
			Keyword:            "hey voxlab",
			Threshold:          0.40,
			PreRollDurationSec: 1.0,
		},
		Intent: IntentConfig{
			ConfidenceThreshold: 0.80,
			MarginThreshold:     0.12,
			CatalogPath:         "data/commands.json",
		},
		Dictation: DictationConfig{
			SilenceEndpointMs: 800,
			MaxDurationSec:    30.0,
		},
		Engine: EngineConfig{
			Mode:           "simulator",
			ModelDir:       "models",
			SherpaTtsBin:   "sherpa-onnx-offline-tts",
			SherpaKwsBin:   "sherpa-onnx-keyword-spotter",
			SherpaAsrBin:   "sherpa-onnx-online-websocket-server",
			KokoroModelDir: "models/kokoro-multi-lang-v1_0",
			ZipformerDir:   "models/sherpa-onnx-streaming-zipformer-en-2023-06-26",
			NumThreads:     4,
			Provider:       "cpu",
		},
		TTS: TTSConfig{
			DefaultVoice: "af",
			DefaultSpeed: 1.0,
			SampleRate:   24000,
			EchoMute:     true,
		},
	}
}

// LoadConfig loads configuration from a JSON file, falling back to defaults if not found.
func LoadConfig(path string) (*AppConfig, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.Version == "" {
		cfg.Version = Version
	}
	return cfg, nil
}
