package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
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
	low := strings.ToLower(cfg.Engine.KokoroModelDir)
	if strings.Contains(low, "fp16") && cfg.Engine.KokoroModelFile == "" {
		cfg.Engine.KokoroModelFile = "model.fp16.onnx"
	} else if strings.Contains(low, "int8") && cfg.Engine.KokoroModelFile == "" {
		cfg.Engine.KokoroModelFile = "model.int8.onnx"
	}
	if dir, ok := findKokoroModelDir(cfg.Engine.KokoroModelDir); ok {
		cfg.Engine.KokoroModelDir = dir
	}
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
	if _, err := os.Stat(filepath.Join(dir, "model.fp16.onnx")); err == nil {
		return true
	}
	return false
}

// FindInstalledKokoroModels scans the models directory and returns all valid Kokoro model names and variants.
func FindInstalledKokoroModels(modelsRoot string) []string {
	if modelsRoot == "" {
		modelsRoot = "models"
	}
	candidates := []string{
		modelsRoot,
		filepath.Join("..", modelsRoot),
		filepath.Join("..", "..", modelsRoot),
	}
	var root string
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			root = c
			break
		}
	}
	if root == "" {
		root = modelsRoot
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dirPath := filepath.Join(root, e.Name())
		if isValidKokoroDir(dirPath) && strings.Contains(strings.ToLower(e.Name()), "kokoro") {
			hasBase := false
			if _, err := os.Stat(filepath.Join(dirPath, "model.onnx")); err == nil {
				out = append(out, e.Name())
				hasBase = true
			}
			if _, err := os.Stat(filepath.Join(dirPath, "model.fp16.onnx")); err == nil {
				if strings.Contains(strings.ToLower(e.Name()), "fp16") {
					if !hasBase {
						out = append(out, e.Name())
						hasBase = true
					}
				} else {
					out = append(out, e.Name()+" (FP16)")
				}
			}
			if _, err := os.Stat(filepath.Join(dirPath, "model.int8.onnx")); err == nil {
				if strings.Contains(strings.ToLower(e.Name()), "int8") {
					if !hasBase {
						out = append(out, e.Name())
						hasBase = true
					}
				} else {
					out = append(out, e.Name()+" (INT8)")
				}
			}
		}
	}

	// Sort models logically: v1_1 first, then v1_0 (base then FP16), then legacy v0_19
	modelRank := func(m string) int {
		low := strings.ToLower(m)
		switch {
		case strings.Contains(low, "v1_1") && !strings.Contains(low, "int8"):
			return 50
		case strings.Contains(low, "v1_1") && strings.Contains(low, "int8"):
			return 40
		case strings.Contains(low, "v1_0") && !strings.Contains(low, "fp16"):
			return 30
		case strings.Contains(low, "v1_0") && strings.Contains(low, "fp16"):
			return 25
		case strings.Contains(low, "v0_19"):
			return 10
		default:
			return 0
		}
	}

	sort.Slice(out, func(i, j int) bool {
		rI, rJ := modelRank(out[i]), modelRank(out[j])
		if rI != rJ {
			return rI > rJ
		}
		return out[i] > out[j]
	})
	return out
}

// findKokoroModelDir automatically discovers the Kokoro model directory.
// Priority: explicitly configured valid dir -> latest multi-lang v1.1 -> int8 v1.1 -> v1.0 -> legacy v0.19.
func findKokoroModelDir(configured string) (string, bool) {
	if configured != "" {
		clean := strings.TrimSpace(configured)
		clean = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(clean, "(FP16)", ""), "(fp16)", ""))
		clean = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(clean, "(INT8)", ""), "(int8)", ""))
		prefixes := []string{
			"",
			"..",
			filepath.Join("..", ".."),
			"models",
			filepath.Join("..", "models"),
			filepath.Join("..", "..", "models"),
		}
		for _, prefix := range prefixes {
			p := filepath.Join(prefix, clean)
			if isValidKokoroDir(p) {
				return p, true
			}
		}
		return configured, false
	}

	candidates := []string{
		filepath.Join("models", "kokoro-multi-lang-v1_1"),
		filepath.Join("..", "models", "kokoro-multi-lang-v1_1"),
		filepath.Join("..", "..", "models", "kokoro-multi-lang-v1_1"),
		filepath.Join("models", "kokoro-int8-multi-lang-v1_1"),
		filepath.Join("..", "models", "kokoro-int8-multi-lang-v1_1"),
		filepath.Join("..", "..", "models", "kokoro-int8-multi-lang-v1_1"),
		filepath.Join("models", "kokoro-multi-lang-v1_0"),
		filepath.Join("..", "models", "kokoro-multi-lang-v1_0"),
		filepath.Join("..", "..", "models", "kokoro-multi-lang-v1_0"),
		filepath.Join("models", "kokoro-int8-multi-lang-v1_0"),
		filepath.Join("..", "models", "kokoro-int8-multi-lang-v1_0"),
		filepath.Join("..", "..", "models", "kokoro-int8-multi-lang-v1_0"),
		filepath.Join("models", "kokoro-en-v0_19"),
		filepath.Join("..", "models", "kokoro-en-v0_19"),
		filepath.Join("..", "..", "models", "kokoro-en-v0_19"),
	}

	for _, c := range candidates {
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
		filepath.Join("..", "bin", name),
		filepath.Join("..", "bin", name+".exe"),
		filepath.Join("..", "..", "bin", name),
		filepath.Join("..", "..", "bin", name+".exe"),
		filepath.Join("models", "bin", name),
		filepath.Join("models", "bin", name+".exe"),
		filepath.Join("..", "models", "bin", name),
		filepath.Join("..", "models", "bin", name+".exe"),
		filepath.Join("..", "..", "models", "bin", name),
		filepath.Join("..", "..", "models", "bin", name+".exe"),
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

	var modelFile string
	if r.cfg.Engine.KokoroModelFile != "" {
		candidate := filepath.Join(modelDir, r.cfg.Engine.KokoroModelFile)
		if isRegularFile(candidate) {
			modelFile = candidate
		}
	}
	if modelFile == "" {
		modelFile = filepath.Join(modelDir, "model.onnx")
		if _, err := os.Stat(modelFile); os.IsNotExist(err) {
			modelFile = filepath.Join(modelDir, "model.int8.onnx")
			if _, err := os.Stat(modelFile); os.IsNotExist(err) {
				modelFile = filepath.Join(modelDir, "model.fp16.onnx")
			}
		}
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

	var ruleFsts []string
	for _, fstName := range []string{"date-zh.fst", "number-zh.fst", "phone-zh.fst"} {
		fstPath := filepath.Join(modelDir, fstName)
		if _, err := os.Stat(fstPath); err == nil {
			ruleFsts = append(ruleFsts, fstPath)
		}
	}
	if len(ruleFsts) > 0 {
		args = append(args, fmt.Sprintf("--tts-rule-fsts=%s", strings.Join(ruleFsts, ",")))
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
	version := detectKokoroVersion(base)

	// Map requested speaker voice to Kokoro speaker ID
	sid := mapVoiceToSID(req.Voice, version)
	cleanText := normalizeTTSText(req.Text)
	args = append(args, fmt.Sprintf("--sid=%d", sid))
	args = append(args, cleanText)

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
	modelFile := r.cfg.Engine.KokoroModelFile
	if modelFile == "" {
		if _, err := os.Stat(filepath.Join(modelDir, "model.onnx")); err == nil {
			modelFile = "model.onnx"
		} else if _, err := os.Stat(filepath.Join(modelDir, "model.int8.onnx")); err == nil {
			modelFile = "model.int8.onnx"
		} else if _, err := os.Stat(filepath.Join(modelDir, "model.fp16.onnx")); err == nil {
			modelFile = "model.fp16.onnx"
		}
	}

	variant := "FP32"
	if strings.Contains(strings.ToLower(modelFile), "fp16") {
		variant = "FP16"
	} else if strings.Contains(strings.ToLower(modelFile), "int8") || strings.Contains(strings.ToLower(base), "int8") {
		variant = "INT8"
	}

	return fmt.Sprintf("%s (%s)", base, variant), true
}

// InstalledTTSModels returns all installed Kokoro models found in the models directory.
func (r *SherpaRunner) InstalledTTSModels() []string {
	modelsRoot := r.cfg.Engine.ModelDir
	if modelsRoot == "" {
		modelsRoot = "models"
	}
	return FindInstalledKokoroModels(modelsRoot)
}

// SetTTSModel switches the active Kokoro model directory and variant to the requested model.
func (r *SherpaRunner) SetTTSModel(modelName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	modelsRoot := r.cfg.Engine.ModelDir
	if modelsRoot == "" {
		modelsRoot = "models"
	}

	clean := strings.TrimSpace(modelName)
	isFP16 := strings.Contains(strings.ToLower(clean), "fp16")
	isINT8 := strings.Contains(strings.ToLower(clean), "int8")
	baseName := clean
	baseName = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(baseName, "(FP16)", ""), "(fp16)", ""))
	baseName = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(baseName, "(INT8)", ""), "(int8)", ""))

	targetDir := filepath.Join(modelsRoot, baseName)
	if !isValidKokoroDir(targetDir) {
		if dir, ok := findKokoroModelDir(baseName); ok {
			targetDir = dir
		} else if isValidKokoroDir(baseName) {
			targetDir = baseName
		} else {
			return fmt.Errorf("model directory not found or invalid: %s", targetDir)
		}
	}

	r.cfg.Engine.KokoroModelDir = targetDir
	if isFP16 {
		fp16Path := filepath.Join(targetDir, "model.fp16.onnx")
		if _, err := os.Stat(fp16Path); err == nil {
			r.cfg.Engine.KokoroModelFile = "model.fp16.onnx"
		} else {
			return fmt.Errorf("fp16 model not found in directory: %s", fp16Path)
		}
	} else if isINT8 {
		int8Path := filepath.Join(targetDir, "model.int8.onnx")
		if _, err := os.Stat(int8Path); err == nil {
			r.cfg.Engine.KokoroModelFile = "model.int8.onnx"
		} else {
			return fmt.Errorf("int8 model not found in directory: %s", int8Path)
		}
	} else {
		if _, err := os.Stat(filepath.Join(targetDir, "model.onnx")); err == nil {
			r.cfg.Engine.KokoroModelFile = "model.onnx"
		} else if _, err := os.Stat(filepath.Join(targetDir, "model.int8.onnx")); err == nil {
			r.cfg.Engine.KokoroModelFile = "model.int8.onnx"
		} else if _, err := os.Stat(filepath.Join(targetDir, "model.fp16.onnx")); err == nil {
			r.cfg.Engine.KokoroModelFile = "model.fp16.onnx"
		} else {
			r.cfg.Engine.KokoroModelFile = ""
		}
	}

	log.Printf("[SherpaRunner] Switched active Kokoro model to: %s (file: %s)", targetDir, r.cfg.Engine.KokoroModelFile)
	return nil
}

// detectKokoroVersion determines the version family ("v0_19", "v1_0", "v1_1") of a Kokoro model.
func detectKokoroVersion(nameOrPath string) string {
	low := strings.ToLower(nameOrPath)
	if strings.Contains(low, "v0_19") || strings.Contains(low, "v0.19") {
		return "v0_19"
	}
	if strings.Contains(low, "v1_0") || strings.Contains(low, "v1.0") {
		return "v1_0"
	}
	return "v1_1"
}

// mapVoiceToSID maps a voice identifier or name to the correct Kokoro speaker ID.
func mapVoiceToSID(voice string, version string) int {
	v := strings.ToLower(strings.TrimSpace(voice))

	// Direct numeric SID support (e.g. "53")
	if n, err := strconv.Atoi(v); err == nil && n >= 0 {
		return n
	}

	switch version {
	case "v0_19":
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

	case "v1_0":
		if sid, ok := KokoroV10VoiceToSID[v]; ok {
			return sid
		}
		// Heuristics for v1.0
		if v == "af" || strings.Contains(v, "heart") {
			return 3 // Default flagship female in v1.0
		}
		if strings.Contains(v, "alloy") {
			return 0
		}
		if strings.Contains(v, "aoede") {
			return 1
		}
		if strings.Contains(v, "bella") {
			return 2
		}
		if strings.Contains(v, "jessica") {
			return 4
		}
		if strings.Contains(v, "kore") {
			return 5
		}
		if strings.Contains(v, "nicole") {
			return 6
		}
		if strings.Contains(v, "nova") {
			return 7
		}
		if strings.Contains(v, "river") {
			return 8
		}
		if strings.Contains(v, "sarah") {
			return 9
		}
		if strings.Contains(v, "sky") {
			return 10
		}
		if strings.Contains(v, "adam") {
			return 11
		}
		if strings.Contains(v, "echo") {
			return 12
		}
		if strings.Contains(v, "eric") {
			return 13
		}
		if strings.Contains(v, "fenrir") {
			return 14
		}
		if strings.Contains(v, "liam") {
			return 15
		}
		if strings.Contains(v, "michael") {
			return 16
		}
		if strings.Contains(v, "onyx") {
			return 17
		}
		if strings.Contains(v, "puck") {
			return 18
		}
		if strings.Contains(v, "alice") {
			return 20
		}
		if strings.Contains(v, "emma") {
			return 21
		}
		if strings.Contains(v, "isabella") {
			return 22
		}
		if strings.Contains(v, "lily") {
			return 23
		}
		if strings.Contains(v, "daniel") {
			return 24
		}
		if strings.Contains(v, "fable") {
			return 25
		}
		if strings.Contains(v, "george") {
			return 26
		}
		if strings.Contains(v, "lewis") {
			return 27
		}
		if strings.Contains(v, "dora") {
			return 28
		}
		if strings.Contains(v, "alex") {
			return 29
		}
		if strings.Contains(v, "siwis") {
			return 30
		}
		if strings.Contains(v, "sara") {
			return 35
		}
		if strings.Contains(v, "nicola") {
			return 36
		}
		if strings.Contains(v, "kumo") {
			return 41
		}
		if strings.Contains(v, "xiaobei") {
			return 45
		}
		if strings.Contains(v, "xiaoni") {
			return 46
		}
		if strings.Contains(v, "xiaoxiao") {
			return 47
		}
		if strings.Contains(v, "xiaoyi") {
			return 48
		}
		if strings.Contains(v, "yunjian") {
			return 49
		}
		if strings.Contains(v, "yunxi") {
			return 50
		}
		if strings.Contains(v, "yunxia") {
			return 51
		}
		if strings.Contains(v, "yunyang") {
			return 52
		}
		if strings.Contains(v, "santa") {
			return 53
		}
		if strings.HasPrefix(v, "am_") || strings.HasPrefix(v, "bm_") || strings.HasPrefix(v, "zm_") || strings.Contains(v, "male") {
			return 11 // Default male in v1.0 (am_adam)
		}
		if strings.HasPrefix(v, "af_") || strings.HasPrefix(v, "bf_") || strings.HasPrefix(v, "zf_") || strings.Contains(v, "female") {
			return 3 // Default female in v1.0 (af_heart)
		}
		return 3

	default: // "v1_1"
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
}

// normalizeTTSText converts full-width and non-ASCII punctuation to standard ASCII equivalents for TTS.
func normalizeTTSText(text string) string {
	r := strings.NewReplacer(
		"，", ", ",
		"。", ". ",
		"！", "! ",
		"？", "? ",
		"；", "; ",
		"：", ": ",
		"、", ", ",
		"（", " (",
		"）", ") ",
		"“", "\"",
		"”", "\"",
		"‘", "'",
		"’", "'",
		"—", "-",
		"…", "...",
	)
	return strings.TrimSpace(r.Replace(text))
}
