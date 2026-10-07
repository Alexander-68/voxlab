package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"voxlab/pkg/audio"
	"voxlab/pkg/config"
)

// SherpaRunner executes official Sherpa-ONNX binaries or falls back to simulation.
type SherpaRunner struct {
	cfg           *config.AppConfig
	simulator     *SimulatorEngine
	mu            sync.Mutex
	turnAudio     []float32
	speechChunks  int
	silenceChunks int
	lastPartial   time.Time
}

// NewSherpaRunner creates a new SherpaRunner instance.
func NewSherpaRunner(cfg *config.AppConfig) *SherpaRunner {
	return &SherpaRunner{
		cfg:       cfg,
		simulator: NewSimulatorEngine(),
	}
}

func (r *SherpaRunner) Name() string {
	tok, enc, _, _, exists := findZipformerModelDir(r.cfg.Engine.ZipformerDir)
	bin, binExists := findSherpaBin("sherpa-onnx")
	if exists && binExists && tok != "" && enc != "" && bin != "" {
		return "sherpa_onnx_native"
	}
	return "simulator"
}

func (r *SherpaRunner) DetectVAD(chunk []float32) (bool, float32) {
	return r.simulator.DetectVAD(chunk)
}

func (r *SherpaRunner) DetectWakeWord(chunk []float32) (bool, string, float64) {
	return r.simulator.DetectWakeWord(chunk)
}

func (r *SherpaRunner) ProcessASRChunk(chunk []float32, isDictation bool) (*ASRResult, error) {
	tok, enc, dec, joi, modelsExist := findZipformerModelDir(r.cfg.Engine.ZipformerDir)
	binPath, binExists := findSherpaBin("sherpa-onnx")

	if !modelsExist || !binExists {
		return r.simulator.ProcessASRChunk(chunk, isDictation)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	_, dbfs := audio.CalculateRMS(chunk)
	isVoice := dbfs > -38.0

	if isVoice {
		r.speechChunks++
		r.silenceChunks = 0
		r.turnAudio = append(r.turnAudio, chunk...)
	} else {
		if r.speechChunks > 0 {
			r.silenceChunks++
			r.turnAudio = append(r.turnAudio, chunk...) // include trailing silence for natural endpointing
		}
	}

	// Endpointing: when speech occurred and trailing silence hits ~450ms (15 chunks @ 30ms)
	// and we have at least 0.4s of audio
	if r.speechChunks >= 8 && r.silenceChunks >= 15 && len(r.turnAudio) >= 16000*4/10 {
		audioToTranscribe := make([]float32, len(r.turnAudio))
		copy(audioToTranscribe, r.turnAudio)
		r.turnAudio = nil
		r.speechChunks = 0
		r.silenceChunks = 0

		res, err := r.transcribeWAV(binPath, tok, enc, dec, joi, audioToTranscribe)
		if err != nil {
			log.Printf("[SherpaRunner] ASR inference error: %v, falling back to simulator", err)
			return r.simulator.ProcessASRChunk(chunk, isDictation)
		}
		res.IsFinal = true
		return res, nil
	}

	// While speaking, stream partial preview
	if r.speechChunks >= 8 && time.Since(r.lastPartial) >= 400*time.Millisecond {
		r.lastPartial = time.Now()
		return &ASRResult{
			Transcript: "(recognizing speech...)",
			IsFinal:    false,
		}, nil
	}

	return nil, nil
}

func (r *SherpaRunner) transcribeWAV(bin, tok, enc, dec, joi string, samples []float32) (*ASRResult, error) {
	tempWav := filepath.Join(os.TempDir(), fmt.Sprintf("voxlab_asr_%d.wav", time.Now().UnixNano()))
	defer os.Remove(tempWav)

	wavBytes, err := audio.EncodeWAV(samples, 16000)
	if err != nil {
		return nil, fmt.Errorf("failed encoding WAV: %w", err)
	}
	if err := os.WriteFile(tempWav, wavBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed writing temp WAV: %w", err)
	}

	cmd := exec.Command(bin,
		"--tokens="+tok,
		"--encoder="+enc,
		"--decoder="+dec,
		"--joiner="+joi,
		"--num-threads=2",
		tempWav,
	)

	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sherpa-onnx execution error (%w): %s", err, string(outBytes))
	}

	outStr := string(outBytes)
	var text string
	var tokens []string

	lines := strings.Split(outStr, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "{") && strings.Contains(trimmed, `"text"`) {
			var parsed struct {
				Text   string   `json:"text"`
				Tokens []string `json:"tokens"`
			}
			if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
				text = strings.TrimSpace(parsed.Text)
				tokens = parsed.Tokens
				break
			}
		}
	}

	if text == "" {
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "OnlineRecognizer") &&
				!strings.HasPrefix(trimmed, "Start") && !strings.HasPrefix(trimmed, "Recognizer") &&
				!strings.HasPrefix(trimmed, "Number of threads") && !strings.HasPrefix(trimmed, "D:") &&
				!strings.HasPrefix(trimmed, "{") {
				text = trimmed
				break
			}
		}
	}

	log.Printf("[SherpaRunner] Real Neural ASR Result: '%s' (%d tokens)", text, len(tokens))

	return &ASRResult{
		Transcript: text,
		Tokens:     tokens,
		Confidence: 0.95,
	}, nil
}

func (r *SherpaRunner) ResetASR() {
	r.mu.Lock()
	r.turnAudio = nil
	r.speechChunks = 0
	r.silenceChunks = 0
	r.mu.Unlock()
	r.simulator.ResetASR()
}

// findZipformerModelDir searches for streaming Zipformer model components.
func findZipformerModelDir(configured string) (tokens, encoder, decoder, joiner string, exists bool) {
	candidates := []string{
		configured,
		filepath.Join("models", "sherpa-onnx-streaming-zipformer-en-2023-06-26"),
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			tok := filepath.Join(c, "tokens.txt")
			enc := filepath.Join(c, "encoder-epoch-99-avg-1-chunk-16-left-128.int8.onnx")
			if _, err := os.Stat(enc); os.IsNotExist(err) {
				enc = filepath.Join(c, "encoder-epoch-99-avg-1-chunk-16-left-128.onnx")
			}
			dec := filepath.Join(c, "decoder-epoch-99-avg-1-chunk-16-left-128.int8.onnx")
			if _, err := os.Stat(dec); os.IsNotExist(err) {
				dec = filepath.Join(c, "decoder-epoch-99-avg-1-chunk-16-left-128.onnx")
			}
			joi := filepath.Join(c, "joiner-epoch-99-avg-1-chunk-16-left-128.int8.onnx")
			if _, err := os.Stat(joi); os.IsNotExist(err) {
				joi = filepath.Join(c, "joiner-epoch-99-avg-1-chunk-16-left-128.onnx")
			}
			if isRegularFile(tok) && isRegularFile(enc) && isRegularFile(dec) && isRegularFile(joi) {
				return tok, enc, dec, joi, true
			}
		}
	}
	return "", "", "", "", false
}

func isRegularFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

// isValidKokoroDir checks if a directory contains a valid Kokoro ONNX model weight.
func isValidKokoroDir(dir string) bool {
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, "model.onnx")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, "model.int8.onnx")); err == nil {
		return true
	}
	return false
}

// findKokoroModelDir automatically discovers the latest available Kokoro model directory.
// Priority order: latest multi-lang v1.1 -> int8 v1.1 -> v1.0 -> explicitly configured custom path -> legacy v0.19.
func findKokoroModelDir(configured string) (string, bool) {
	// If user explicitly configured a custom path (not default v1.1 and not legacy v0.19), try it first
	if configured != "" &&
		!strings.Contains(configured, "kokoro-en-v0_19") &&
		!strings.Contains(configured, "kokoro-multi-lang-v1_1") &&
		!strings.Contains(configured, "kokoro-int8-multi-lang-v1_1") {
		if isValidKokoroDir(configured) {
			return configured, true
		}
	}

	candidates := []string{
		filepath.Join("models", "kokoro-multi-lang-v1_1"),
		filepath.Join("models", "kokoro-int8-multi-lang-v1_1"),
		filepath.Join("models", "kokoro-multi-lang-v1_0"),
		filepath.Join("models", "kokoro-int8-multi-lang-v1_0"),
		configured,
		filepath.Join("models", "kokoro-en-v0_19"),
	}

	for _, c := range candidates {
		if c == "" {
			continue
		}
		if isValidKokoroDir(c) {
			return c, true
		}
	}
	return configured, false
}

// findSherpaBin searches for a sherpa-onnx binary on PATH or local folders.
func findSherpaBin(name string) (string, bool) {
	// 1. Direct LookPath
	if path, err := exec.LookPath(name); err == nil {
		return path, true
	}

	// 2. Local bin/ or models/bin/ directories
	candidates := []string{
		filepath.Join("bin", name),
		filepath.Join("bin", name+".exe"),
		filepath.Join("models", "bin", name),
		filepath.Join("models", "bin", name+".exe"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, true
		}
	}

	// 3. Search within models subfolders for unpacked binaries
	matches, _ := filepath.Glob(filepath.Join("models", "*", "bin", name+"*"))
	if len(matches) > 0 {
		return matches[0], true
	}
	matchesBin, _ := filepath.Glob(filepath.Join("bin", "*", "bin", name+"*"))
	if len(matchesBin) > 0 {
		return matchesBin[0], true
	}

	return name, false
}

// Synthesize runs Kokoro TTS via sherpa-onnx-offline-tts binary if available.
func (r *SherpaRunner) Synthesize(req TTSRequest) (*TTSResult, error) {
	modelDir, modelExists := findKokoroModelDir(r.cfg.Engine.KokoroModelDir)
	ttsBin, binExists := findSherpaBin(r.cfg.Engine.SherpaTtsBin)

	if !binExists || !modelExists {
		// Gracefully fall back to simulator if real model/binary is not yet downloaded
		return r.simulator.Synthesize(req)
	}

	startTime := time.Now()
	tempWav := filepath.Join(os.TempDir(), fmt.Sprintf("voxlab_tts_%d.wav", time.Now().UnixNano()))
	defer os.Remove(tempWav)

	modelFile := filepath.Join(modelDir, "model.onnx")
	if _, err := os.Stat(modelFile); os.IsNotExist(err) {
		modelFile = filepath.Join(modelDir, "model.int8.onnx")
	}
	voicesPath := filepath.Join(modelDir, "voices.bin")
	tokensPath := filepath.Join(modelDir, "tokens.txt")
	dataDirPath := filepath.Join(modelDir, "espeak-ng-data")

	args := []string{
		fmt.Sprintf("--kokoro-model=%s", modelFile),
		fmt.Sprintf("--kokoro-voices=%s", voicesPath),
		fmt.Sprintf("--kokoro-tokens=%s", tokensPath),
		fmt.Sprintf("--kokoro-data-dir=%s", dataDirPath),
		fmt.Sprintf("--output-filename=%s", tempWav),
	}

	var lexicons []string
	for _, lexName := range []string{"lexicon-us-en.txt", "lexicon-gb-en.txt", "lexicon-zh.txt"} {
		lexPath := filepath.Join(modelDir, lexName)
		if _, err := os.Stat(lexPath); err == nil {
			lexicons = append(lexicons, lexPath)
		}
	}
	if len(lexicons) > 0 {
		args = append(args, fmt.Sprintf("--kokoro-lexicon=%s", strings.Join(lexicons, ",")))
	}

	// Speech speed parameter (0.5x to 2.0x)
	speed := req.Speed
	if speed <= 0 {
		speed = 1.0
	}
	if speed < 0.5 {
		speed = 0.5
	} else if speed > 2.0 {
		speed = 2.0
	}
	args = append(args, fmt.Sprintf("--speed=%.2f", speed))

	base := filepath.Base(modelDir)
	isV019 := strings.Contains(strings.ToLower(base), "v0_19") || strings.Contains(strings.ToLower(base), "v0.19")

	// Map requested speaker voice to Kokoro speaker ID
	sid := mapVoiceToSID(req.Voice, isV019)
	args = append(args, fmt.Sprintf("--sid=%d", sid))
	args = append(args, req.Text)

	log.Printf("[SherpaRunner] Synthesizing speech: voice='%s' -> sid=%d, speed=%.2f", req.Voice, sid, speed)

	cmd := exec.Command(ttsBin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("[SherpaRunner] Execution error (%v), falling back to simulator: %s", err, stderr.String())
		return r.simulator.Synthesize(req)
	}

	wavData, err := os.ReadFile(tempWav)
	if err != nil {
		return nil, fmt.Errorf("failed reading output wav: %w", err)
	}

	samples, sRate, err := audio.DecodeWAV(bytes.NewReader(wavData))
	if err != nil {
		return nil, fmt.Errorf("failed decoding output wav: %w", err)
	}

	duration := float64(len(samples)) / float64(sRate)
	latency := time.Since(startTime).Milliseconds()

	return &TTSResult{
		AudioSamples: samples,
		WAVBytes:     wavData,
		SampleRate:   sRate,
		DurationSec:  duration,
		LatencyMs:    latency,
	}, nil
}

// TTSModelInfo discovers and returns the current active Kokoro model name and neural status.
func (r *SherpaRunner) TTSModelInfo() (string, bool) {
	modelDir, modelExists := findKokoroModelDir(r.cfg.Engine.KokoroModelDir)
	_, binExists := findSherpaBin(r.cfg.Engine.SherpaTtsBin)

	if !binExists || !modelExists {
		return "Simulator (Fallback: Model Not Found)", false
	}

	base := filepath.Base(modelDir)
	modelFile := filepath.Join(modelDir, "model.onnx")
	variant := "FP32"
	if _, err := os.Stat(modelFile); os.IsNotExist(err) {
		variant = "INT8"
	}

	return fmt.Sprintf("%s (%s)", base, variant), true
}

// mapVoiceToSID maps a voice identifier or name to the correct Kokoro speaker ID.
func mapVoiceToSID(voice string, isV019 bool) int {
	v := strings.ToLower(strings.TrimSpace(voice))

	if isV019 {
		if sid, ok := KokoroV019VoiceToSID[v]; ok {
			return sid
		}
		// Substring heuristics for legacy v0.19
		if strings.Contains(v, "adam") {
			return 5
		}
		if strings.Contains(v, "michael") {
			return 6
		}
		if strings.Contains(v, "george") || strings.Contains(v, "fenrir") {
			return 9
		}
		if strings.Contains(v, "lewis") || strings.Contains(v, "puck") {
			return 10
		}
		if strings.HasPrefix(v, "am_") || strings.HasPrefix(v, "bm_") || strings.Contains(v, "male") {
			return 5
		}
		if strings.Contains(v, "bella") {
			return 1
		}
		if strings.Contains(v, "nicole") || strings.Contains(v, "alloy") {
			return 2
		}
		if strings.Contains(v, "sarah") {
			return 3
		}
		if strings.Contains(v, "sky") || strings.Contains(v, "aoede") {
			return 4
		}
		if strings.Contains(v, "emma") {
			return 7
		}
		if strings.Contains(v, "isabella") {
			return 8
		}
		return 0
	}

	// v1.1 Model (Default)
	if sid, ok := KokoroV11VoiceToSID[v]; ok {
		return sid
	}

	// Heuristics for v1.1
	if v == "af" || strings.Contains(v, "maple") {
		return 0
	}
	if strings.Contains(v, "sol") {
		return 1
	}
	if strings.Contains(v, "vale") {
		return 2
	}
	if strings.HasPrefix(v, "zm_") || strings.Contains(v, "male") {
		return 58 // Default male in v1.1 (zm_009)
	}
	if strings.HasPrefix(v, "zf_") || strings.Contains(v, "female") {
		return 0 // Default female in v1.1 (af_maple)
	}

	return 0
}
