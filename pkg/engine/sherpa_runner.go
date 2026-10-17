package engine

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"voxlab/pkg/audio"
	"voxlab/pkg/config"
)

// SherpaRunner executes official Sherpa-ONNX binaries or falls back to simulation.
type SherpaRunner struct {
	cfg       *config.AppConfig
	simulator *SimulatorEngine
	warmTTS   WarmTTS
	mu        sync.Mutex
	asrMu     sync.Mutex // guards the ASR fields below; separate from mu so TTS never blocks ASR
	warmASR   *WarmASR   // resident ASR server, set as soon as its process is spawned
	asrReady  bool       // warmASR finished loading models
	asrModels bool       // streaming models + server binary are installed
	asrClosed bool       // Close was called; late-starting servers must be killed
	stream    *ASRStream
	streamDir string // active first-pass (streaming) model directory

	offline      *WarmOffline // resident second-pass (Parakeet) server
	offlineReady bool
	secondPass   bool
	segAudio     []float32 // audio sent to the current stream not yet assigned to a segment
	segBase      int       // stream sample index of segAudio[0]

	// OnRefine reports each second-pass correction (set by the server before audio flows).
	OnRefine func(isDictation bool, firstPass, refined string, latencyMs int64)
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
	runner := &SherpaRunner{
		cfg:       cfg,
		simulator: NewSimulatorEngine(),
	}
	fallbacks := []string{cfg.Engine.StreamingModelDir}
	fallbacks = append(fallbacks, InstalledStreamingModels(cfg.Engine.ModelDir)...)
	fallbacks = append(fallbacks, filepath.Join("models", "sherpa-onnx-streaming-zipformer-en-2023-06-26"))
	runner.streamDir = resolveModelDir(fallbacks...)
	_, binExists := findSherpaBin("sherpa-onnx-online-websocket-server")
	runner.asrModels = runner.streamDir != "" && binExists
	runner.secondPass = cfg.Engine.SecondPass

	// Eagerly warm up ASR in background so first spoken utterance has zero cold start latency
	runner.asrMu.Lock()
	runner.startWarmASRLocked()
	runner.startOfflineLocked()
	runner.asrMu.Unlock()
	return runner
}

// startWarmASRLocked spawns the resident ASR server and loads it in the background (caller holds asrMu).
func (r *SherpaRunner) startWarmASRLocked() {
	if !r.asrModels || r.asrClosed || r.warmASR != nil {
		return
	}
	tok, enc, dec, joi, _ := findTransducerModel(r.streamDir)
	warm, err := StartWarmASR(tok, enc, dec, joi, r.cfg.Engine.NumThreads, float64(r.cfg.Dictation.SilenceEndpointMs)/1000)
	if err != nil {
		log.Printf("[SherpaRunner] Warm ASR start failed: %v", err)
		return
	}
	r.warmASR = warm
	go func() {
		err := warm.WaitReady()
		r.asrMu.Lock()
		defer r.asrMu.Unlock()
		if r.warmASR != warm { // closed or replaced meanwhile
			return
		}
		if err != nil {
			log.Printf("[SherpaRunner] Warm ASR initialization failed: %v", err)
			r.warmASR = nil
			return
		}
		r.asrReady = true
	}()
}

// startOfflineLocked spawns the second-pass server in the background when enabled (caller holds asrMu).
func (r *SherpaRunner) startOfflineLocked() {
	if !r.secondPass || r.asrClosed || r.offline != nil {
		return
	}
	dir := resolveModelDir(r.cfg.Engine.SecondPassModelDir)
	if dir == "" {
		log.Printf("[SherpaRunner] Second-pass model not found (%s); using streaming results only", r.cfg.Engine.SecondPassModelDir)
		return
	}
	tok, enc, dec, joi, _ := findTransducerModel(dir)
	off, err := StartWarmOffline(tok, enc, dec, joi, r.cfg.Engine.NumThreads)
	if err != nil {
		log.Printf("[SherpaRunner] Second-pass ASR start failed: %v", err)
		return
	}
	r.offline = off
	go func() {
		err := off.WaitReady()
		r.asrMu.Lock()
		defer r.asrMu.Unlock()
		if r.offline != off {
			return
		}
		if err != nil {
			log.Printf("[SherpaRunner] Second-pass ASR initialization failed: %v", err)
			r.offline = nil
			return
		}
		r.offlineReady = true
	}()
}

func (r *SherpaRunner) stopOfflineLocked() {
	if r.offline != nil {
		_ = r.offline.Close()
		r.offline, r.offlineReady = nil, false
	}
}

// SetSecondPass toggles Parakeet re-transcription of endpointed segments; disabling frees its RAM.
func (r *SherpaRunner) SetSecondPass(enabled bool) {
	r.asrMu.Lock()
	defer r.asrMu.Unlock()
	r.secondPass = enabled
	r.cfg.Engine.SecondPass = enabled
	if enabled {
		r.startOfflineLocked()
	} else {
		r.stopOfflineLocked()
	}
}

// SetStreamingModel restarts the resident first-pass server with another installed model directory.
func (r *SherpaRunner) SetStreamingModel(name string) error {
	dir := ""
	for _, m := range InstalledStreamingModels(r.cfg.Engine.ModelDir) {
		if filepath.Base(m) == name {
			dir = m
		}
	}
	if dir == "" {
		return fmt.Errorf("streaming model %q not installed", name)
	}
	r.asrMu.Lock()
	defer r.asrMu.Unlock()
	if r.stream != nil {
		r.stream.Close()
		r.stream = nil
	}
	if r.warmASR != nil {
		_ = r.warmASR.Close()
		r.warmASR, r.asrReady = nil, false
	}
	r.streamDir = dir
	r.cfg.Engine.StreamingModelDir = dir
	_, binExists := findSherpaBin("sherpa-onnx-online-websocket-server")
	r.asrModels = binExists
	r.startWarmASRLocked()
	return nil
}

// ASRModelInfo reports the active streaming model, installed alternatives and second-pass state.
func (r *SherpaRunner) ASRModelInfo() map[string]interface{} {
	r.asrMu.Lock()
	defer r.asrMu.Unlock()
	var names []string
	for _, m := range InstalledStreamingModels(r.cfg.Engine.ModelDir) {
		names = append(names, filepath.Base(m))
	}
	return map[string]interface{}{
		"asr_model":          filepath.Base(r.streamDir),
		"asr_models":         names,
		"second_pass":        r.secondPass,
		"second_pass_model":  filepath.Base(r.cfg.Engine.SecondPassModelDir),
		"second_pass_loaded": r.offlineReady,
	}
}

// cutSegmentLocked removes the audio of an endpointed segment from segAudio and returns it.
// Token timestamps bound the speech; pads cover onsets before the first token and the last word's tail.
func (r *SherpaRunner) cutSegmentLocked(res *ASRResult) []float32 {
	sr := float64(r.cfg.Audio.SampleRate)
	from, to := 0, len(r.segAudio)
	if res.EndSec > 0 {
		from = max(0, min(to, int((res.StartSec-0.6)*sr)-r.segBase))
		to = max(from, min(to, int((res.EndSec+0.5)*sr)-r.segBase))
	}
	seg := r.segAudio[from:to]
	r.segAudio = append([]float32(nil), r.segAudio[to:]...)
	r.segBase += to
	return seg
}

// refineLocked replaces res.Transcript with the second-pass result for seg, if available.
func (r *SherpaRunner) refineLocked(res *ASRResult, seg []float32, isDictation bool) {
	if !r.secondPass || !r.offlineReady || len(seg) == 0 {
		return
	}
	t0 := time.Now()
	text, err := r.offline.Transcribe(seg, r.cfg.Audio.SampleRate)
	if err != nil {
		log.Printf("[SherpaRunner] Second pass failed, keeping streaming result: %v", err)
		return
	}
	if text == "" {
		return // streaming model heard words the offline model did not; keep them rather than lose speech
	}
	ms := time.Since(t0).Milliseconds()
	first := res.Transcript
	res.Transcript = text
	if r.OnRefine != nil {
		r.OnRefine(isDictation, first, text, ms)
	}
}

func (r *SherpaRunner) Name() string {
	if r.asrModels {
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

// ProcessASRChunk streams a chunk to the live recognizer and returns the next pending
// result: an endpointed segment (final), a changed partial hypothesis, or nil.
// Endpointing is done by sherpa itself on decoded blank frames, so it is independent of
// gain/noise-gate settings.
func (r *SherpaRunner) ProcessASRChunk(chunk []float32, isDictation bool) (*ASRResult, error) {
	if !r.asrModels {
		return r.simulator.ProcessASRChunk(chunk, isDictation)
	}

	r.asrMu.Lock()
	defer r.asrMu.Unlock()

	if r.stream == nil {
		if !r.asrReady {
			r.startWarmASRLocked()
			return nil, nil // still loading; audio before the model is ready is dropped
		}
		st, err := r.warmASR.Open()
		if err != nil {
			// Server likely died: restart it in the background.
			_ = r.warmASR.Close()
			r.warmASR, r.asrReady = nil, false
			r.startWarmASRLocked()
			return nil, err
		}
		r.stream = st
		r.segAudio, r.segBase = nil, 0
	}

	if err := r.stream.Send(chunk); err != nil {
		r.stream.Close()
		r.stream = nil
		return nil, err
	}
	r.segAudio = append(r.segAudio, chunk...)

	res := r.stream.Next()
	if res != nil && res.IsFinal {
		r.refineLocked(res, r.cutSegmentLocked(res), isDictation)
	}
	if res != nil && res.IsFinal && !isDictation {
		// A command turn ends at its first endpoint; the next turn gets a fresh stream.
		r.stream.Close()
		r.stream = nil
	}
	return res, nil
}

// ASRReady reports whether the resident ASR server has loaded its models.
func (r *SherpaRunner) ASRReady() bool {
	r.asrMu.Lock()
	defer r.asrMu.Unlock()
	return r.asrReady
}

// FlushASR finalizes the active stream, returning any speech not yet endpointed.
func (r *SherpaRunner) FlushASR() string {
	r.asrMu.Lock()
	defer r.asrMu.Unlock()
	if r.stream == nil {
		return ""
	}
	parts := r.stream.Finish(3 * time.Second)
	r.stream = nil
	res := &ASRResult{IsFinal: true, Transcript: strings.Join(parts, " ")}
	if res.Transcript != "" {
		r.refineLocked(res, r.segAudio, true)
	}
	r.segAudio, r.segBase = nil, 0
	return res.Transcript
}

func (r *SherpaRunner) ResetASR() {
	r.asrMu.Lock()
	if r.stream != nil {
		r.stream.Close()
		r.stream = nil
	}
	r.asrMu.Unlock()
	r.simulator.ResetASR()
}

// resolveModelDir returns the first candidate directory (also tried relative to parent dirs,
// so tests running inside pkg/ find the repo models) that holds a complete transducer model.
func resolveModelDir(candidates ...string) string {
	for _, c := range candidates {
		if c == "" {
			continue
		}
		for _, base := range []string{"", "..", filepath.Join("..", "..")} {
			dir := filepath.Join(base, c)
			if _, _, _, _, ok := findTransducerModel(dir); ok {
				return dir
			}
		}
	}
	return ""
}

// InstalledStreamingModels lists streaming transducer model directories under modelsRoot.
func InstalledStreamingModels(modelsRoot string) []string {
	matches, _ := filepath.Glob(filepath.Join(modelsRoot, "*streaming*"))
	var out []string
	for _, m := range matches {
		if _, _, _, _, ok := findTransducerModel(m); ok {
			out = append(out, m)
		}
	}
	return out
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

	// Sort models logically: v1_0 first (default), then v1_1, then legacy v0_19
	modelRank := func(m string) int {
		low := strings.ToLower(m)
		switch {
		case strings.Contains(low, "v1_0") && !strings.Contains(low, "fp16") && !strings.Contains(low, "int8"):
			return 50
		case strings.Contains(low, "v1_0") && strings.Contains(low, "fp16"):
			return 45
		case strings.Contains(low, "v1_1") && !strings.Contains(low, "int8"):
			return 40
		case strings.Contains(low, "v1_1") && strings.Contains(low, "int8"):
			return 30
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
// Priority: explicitly configured valid dir -> default multi-lang v1.0 -> int8 v1.0 -> v1.1 -> legacy v0.19.
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
		filepath.Join("models", "kokoro-multi-lang-v1_0"),
		filepath.Join("..", "models", "kokoro-multi-lang-v1_0"),
		filepath.Join("..", "..", "models", "kokoro-multi-lang-v1_0"),
		filepath.Join("models", "kokoro-int8-multi-lang-v1_0"),
		filepath.Join("..", "models", "kokoro-int8-multi-lang-v1_0"),
		filepath.Join("..", "..", "models", "kokoro-int8-multi-lang-v1_0"),
		filepath.Join("models", "kokoro-multi-lang-v1_1"),
		filepath.Join("..", "models", "kokoro-multi-lang-v1_1"),
		filepath.Join("..", "..", "models", "kokoro-multi-lang-v1_1"),
		filepath.Join("models", "kokoro-int8-multi-lang-v1_1"),
		filepath.Join("..", "models", "kokoro-int8-multi-lang-v1_1"),
		filepath.Join("..", "..", "models", "kokoro-int8-multi-lang-v1_1"),
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

// resolveKokoroModelFile finds the active model weights file (.onnx, .fp16.onnx, .int8.onnx) in modelDir.
func (r *SherpaRunner) resolveKokoroModelFile(modelDir string) string {
	if modelDir == "" {
		return ""
	}
	if r.cfg.Engine.KokoroModelFile != "" {
		candidate := filepath.Join(modelDir, r.cfg.Engine.KokoroModelFile)
		if isRegularFile(candidate) {
			return candidate
		}
	}
	for _, name := range []string{"model.onnx", "model.fp16.onnx", "model.int8.onnx"} {
		p := filepath.Join(modelDir, name)
		if isRegularFile(p) {
			return p
		}
	}
	return ""
}

// getWarmTTS returns or initializes the resident in-memory TTS engine if the C-API DLL is available.
func (r *SherpaRunner) getWarmTTS(modelDir string, modelFile string) WarmTTS {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.getWarmTTSLocked(modelDir, modelFile)
}

func (r *SherpaRunner) getWarmTTSLocked(modelDir string, modelFile string) WarmTTS {

	numThreads := r.cfg.Engine.NumThreads
	if numThreads <= 0 {
		numThreads = 4
	}
	provider := r.cfg.Engine.Provider
	if provider == "" {
		provider = "cpu"
	}

	key := fmt.Sprintf("%s|t%d|%s", modelFile, numThreads, provider)
	if r.warmTTS != nil && r.warmTTS.IsWarm() && r.warmTTS.ModelKey() == key {
		return r.warmTTS
	}

	if r.warmTTS != nil {
		r.warmTTS.Close()
		r.warmTTS = nil
	}

	// Search for directory containing sherpa-onnx-c-api.dll
	binCandidates := []string{"bin", filepath.Join("..", "bin"), filepath.Join("..", "..", "bin")}
	var dllDir string
	for _, c := range binCandidates {
		if isRegularFile(filepath.Join(c, "sherpa-onnx-c-api.dll")) {
			dllDir = c
			break
		}
	}
	if dllDir == "" {
		return nil
	}

	voicesPath := filepath.Join(modelDir, "voices.bin")
	tokensPath := filepath.Join(modelDir, "tokens.txt")
	dataDirPath := filepath.Join(modelDir, "espeak-ng-data")

	var lexicons []string
	for _, lexName := range []string{"lexicon-us-en.txt", "lexicon-gb-en.txt", "lexicon-zh.txt"} {
		lp := filepath.Join(modelDir, lexName)
		if isRegularFile(lp) {
			absL, _ := filepath.Abs(lp)
			lexicons = append(lexicons, absL)
		}
	}
	lexPath := strings.Join(lexicons, ",")

	var ruleFsts []string
	for _, fstName := range []string{"date-zh.fst", "number-zh.fst", "phone-zh.fst"} {
		fp := filepath.Join(modelDir, fstName)
		if isRegularFile(fp) {
			absF, _ := filepath.Abs(fp)
			ruleFsts = append(ruleFsts, absF)
		}
	}
	ruleFstsStr := strings.Join(ruleFsts, ",")

	warm, err := newPlatformWarmTTS(dllDir, modelFile, voicesPath, tokensPath, dataDirPath, lexPath, ruleFstsStr, numThreads, provider)
	if err != nil {
		log.Printf("[SherpaRunner] In-process warm TTS init notice: %v (will use CLI fallback)", err)
		return nil
	}
	r.warmTTS = warm
	return warm
}

// WarmUp pre-loads the active Kokoro model into RAM and synthesizes "Hello" to keep the model always warm.
func (r *SherpaRunner) WarmUp() {
	r.mu.Lock()
	modelDir, modelExists := findKokoroModelDir(r.cfg.Engine.KokoroModelDir)
	var modelFile string
	if modelExists {
		modelFile = r.resolveKokoroModelFile(modelDir)
	}
	r.mu.Unlock()

	if !modelExists || !isRegularFile(modelFile) {
		return
	}
	warm := r.getWarmTTS(modelDir, modelFile)
	if warm == nil {
		return
	}
	version := detectKokoroVersion(filepath.Base(modelDir))
	sid := mapVoiceToSID("default", version)
	t0 := time.Now()
	_, err := warm.SynthesizeStream("Hello", sid, 1.0, nil)
	if err == nil {
		log.Printf("[SherpaRunner] Model warmed up with 'Hello' in %v (file: %s)", time.Since(t0), filepath.Base(modelFile))
	} else {
		log.Printf("[SherpaRunner] Model warmup notice: %v", err)
	}
}

// Synthesize runs Kokoro TTS via in-process warm engine if available, or falls back to sherpa-onnx-offline-tts binary / simulator.
func (r *SherpaRunner) Synthesize(req TTSRequest) (*TTSResult, error) {
	return r.SynthesizeStream(req, nil)
}

// SynthesizeStream runs streaming Kokoro TTS via in-process warm engine callback or fallback pipelines.
func (r *SherpaRunner) SynthesizeStream(req TTSRequest, onChunk func(chunk TTSChunk) error) (*TTSResult, error) {
	modelDir, modelExists := findKokoroModelDir(r.cfg.Engine.KokoroModelDir)
	ttsBin, binExists := findSherpaBin(r.cfg.Engine.SherpaTtsBin)

	if !modelExists && !binExists {
		// Gracefully fall back to simulator if real model/binary is not yet downloaded
		return r.simulator.SynthesizeStream(req, onChunk)
	}

	modelFile := r.resolveKokoroModelFile(modelDir)
	base := filepath.Base(modelDir)
	version := detectKokoroVersion(base)
	sid := mapVoiceToSID(req.Voice, version)
	cleanText := normalizeTTSText(req.Text)
	if onChunk != nil {
		cleanText = optimizeStreamingClauses(cleanText)
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

	// 1. Try In-Memory Warm Engine (sub-second latency with chunk-by-chunk streaming)
	if isRegularFile(modelFile) {
		if warm := r.getWarmTTS(modelDir, modelFile); warm != nil {
			log.Printf("[SherpaRunner] Synthesizing speech (WARM in-memory, streaming=%v): voice='%s' -> sid=%d, speed=%.2f",
				onChunk != nil, req.Voice, sid, speed)
			result, err := warm.SynthesizeStream(cleanText, sid, speed, onChunk)
			if err == nil {
				log.Printf("[SherpaRunner] Warm TTS generation complete: %.2fs audio in %dms (RTF=%.3f)",
					result.DurationSec, result.LatencyMs, float64(result.LatencyMs)/(result.DurationSec*1000.0))
				return result, nil
			}
			log.Printf("[SherpaRunner] Warm in-process TTS failed (%v), falling back to CLI process...", err)
		}
	}

	// 2. Fall back to CLI process execution
	if !binExists || !modelExists {
		return r.simulator.SynthesizeStream(req, onChunk)
	}

	startTime := time.Now()
	tempWav := filepath.Join(os.TempDir(), fmt.Sprintf("voxlab_tts_%d.wav", time.Now().UnixNano()))
	defer os.Remove(tempWav)

	voicesPath := filepath.Join(modelDir, "voices.bin")
	tokensPath := filepath.Join(modelDir, "tokens.txt")
	dataDirPath := filepath.Join(modelDir, "espeak-ng-data")

	numThreads := r.cfg.Engine.NumThreads
	if numThreads <= 0 {
		numThreads = 4
	}
	provider := r.cfg.Engine.Provider
	if provider == "" {
		provider = "cpu"
	}

	args := []string{
		fmt.Sprintf("--kokoro-model=%s", modelFile),
		fmt.Sprintf("--kokoro-voices=%s", voicesPath),
		fmt.Sprintf("--kokoro-tokens=%s", tokensPath),
		fmt.Sprintf("--kokoro-data-dir=%s", dataDirPath),
		fmt.Sprintf("--output-filename=%s", tempWav),
		fmt.Sprintf("--num-threads=%d", numThreads),
		fmt.Sprintf("--provider=%s", provider),
		"--tts-silence-scale=0.5",
	}

	var lexicons []string
	for _, lexName := range []string{"lexicon-us-en.txt", "lexicon-gb-en.txt", "lexicon-zh.txt"} {
		lexPath := filepath.Join(modelDir, lexName)
		if isRegularFile(lexPath) {
			lexicons = append(lexicons, lexPath)
		}
	}
	if len(lexicons) > 0 {
		args = append(args, fmt.Sprintf("--kokoro-lexicon=%s", strings.Join(lexicons, ",")))
	}

	var ruleFsts []string
	for _, fstName := range []string{"date-zh.fst", "number-zh.fst", "phone-zh.fst"} {
		fstPath := filepath.Join(modelDir, fstName)
		if isRegularFile(fstPath) {
			ruleFsts = append(ruleFsts, fstPath)
		}
	}
	if len(ruleFsts) > 0 {
		args = append(args, fmt.Sprintf("--tts-rule-fsts=%s", strings.Join(ruleFsts, ",")))
	}

	args = append(args, fmt.Sprintf("--speed=%.2f", speed))
	args = append(args, fmt.Sprintf("--sid=%d", sid))
	args = append(args, cleanText)

	log.Printf("[SherpaRunner] Synthesizing speech (CLI process, %d threads, %s provider): voice='%s' -> sid=%d, speed=%.2f",
		numThreads, provider, req.Voice, sid, speed)

	cmd := exec.Command(ttsBin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("[SherpaRunner] Execution error (%v), falling back to simulator: %s", err, stderr.String())
		return r.simulator.SynthesizeStream(req, onChunk)
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

	if onChunk != nil {
		_ = onChunk(TTSChunk{
			Index:        0,
			IsLast:       true,
			AudioSamples: samples,
			WAVBytes:     wavData,
			SampleRate:   sRate,
			DurationSec:  duration,
			LatencyMs:    latency,
		})
	}

	return &TTSResult{
		AudioSamples: samples,
		WAVBytes:     wavData,
		SampleRate:   sRate,
		DurationSec:  duration,
		LatencyMs:    latency,
	}, nil
}

// Close closes any active warm in-process engines and frees allocated native resources.
func (r *SherpaRunner) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.warmTTS != nil {
		r.warmTTS.Close()
		r.warmTTS = nil
	}
	r.asrMu.Lock()
	defer r.asrMu.Unlock()
	r.asrClosed = true
	if r.stream != nil {
		r.stream.Close()
		r.stream = nil
	}
	if r.warmASR != nil {
		_ = r.warmASR.Close()
		r.warmASR, r.asrReady = nil, false
	}
	r.stopOfflineLocked()
	return nil
}

// TTSModelInfo discovers and returns the current active Kokoro model name and neural status.
func (r *SherpaRunner) TTSModelInfo() (string, bool) {
	modelDir, modelExists := findKokoroModelDir(r.cfg.Engine.KokoroModelDir)
	_, binExists := findSherpaBin(r.cfg.Engine.SherpaTtsBin)

	if !binExists && !modelExists {
		return "Simulator (Fallback: Model Not Found)", false
	}

	base := filepath.Base(modelDir)
	modelFile := r.cfg.Engine.KokoroModelFile
	if modelFile == "" {
		if isRegularFile(filepath.Join(modelDir, "model.onnx")) {
			modelFile = "model.onnx"
		} else if isRegularFile(filepath.Join(modelDir, "model.fp16.onnx")) {
			modelFile = "model.fp16.onnx"
		} else if isRegularFile(filepath.Join(modelDir, "model.int8.onnx")) {
			modelFile = "model.int8.onnx"
		}
	}

	variant := "FP32"
	if strings.Contains(strings.ToLower(modelFile), "fp16") {
		variant = "FP16"
	} else if strings.Contains(strings.ToLower(modelFile), "int8") || strings.Contains(strings.ToLower(base), "int8") {
		variant = "INT8"
	}

	status := fmt.Sprintf("%s (%s)", base, variant)
	return status, true
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

	if r.warmTTS != nil {
		r.warmTTS.Close()
		r.warmTTS = nil
	}

	log.Printf("[SherpaRunner] Switched active Kokoro model to: %s (file: %s)", targetDir, r.cfg.Engine.KokoroModelFile)

	// Keep model always warm: synthesize "Hello" after model switch
	go r.WarmUp()

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

var (
	reTrailingDots = regexp.MustCompile(`\.{2,}\s*$`)
	reMultiDot     = regexp.MustCompile(`\.{2,}`)
)

// normalizeTTSText converts full-width and non-ASCII punctuation to standard ASCII equivalents for TTS,
// and maps ellipses and pause markers to symbols that Kokoro executes as natural pauses.
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
		"……", "; ",
		"…", "; ",
		"[pause]", "; ",
		"<pause>", "; ",
		"(pause)", "; ",
		" -- ", "; ",
		"--", "; ",
		" — ", "; ",
		"—", "; ",
	)
	text = strings.TrimSpace(r.Replace(text))

	// If text ends with multiple dots (e.g. "Loading..."), normalize to a single period
	text = reTrailingDots.ReplaceAllString(text, ".")

	// Inside text, convert runs of 2+ dots ("..." or "..") into semicolon pauses ("; ")
	// because Kokoro has no token for "..." and espeak-ng drops them, causing words to stick together.
	text = reMultiDot.ReplaceAllString(text, "; ")

	// Clean up any double semicolons, leading space before punctuation, or double spaces
	for strings.Contains(text, "; ;") || strings.Contains(text, ";;") {
		text = strings.ReplaceAll(text, "; ;", "; ")
		text = strings.ReplaceAll(text, ";;", "; ")
	}
	text = strings.ReplaceAll(text, " ;", ";")
	text = strings.ReplaceAll(text, " :", ":")
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}

	return strings.TrimSpace(text)
}

// optimizeStreamingClauses converts semicolons, colons, and multi-word comma clauses
// into capitalized sentence-level boundaries (. ) so Sherpa-ONNX streaming callback
// fires after the first clause instead of waiting for the full utterance.
func optimizeStreamingClauses(text string) string {
	text = strings.ReplaceAll(text, "; ", ". ")
	text = strings.ReplaceAll(text, ";", ". ")
	text = strings.ReplaceAll(text, ": ", ". ")

	// If text contains commas, split into clauses if both sides contain >= 3 words
	if strings.Contains(text, ",") {
		parts := strings.Split(text, ",")
		var b strings.Builder
		for i, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed == "" {
				continue
			}
			if i > 0 {
				prevWords := len(strings.Fields(parts[i-1]))
				currWords := len(strings.Fields(p))
				if prevWords >= 3 && currWords >= 3 {
					b.WriteString(". ")
				} else {
					b.WriteString(", ")
				}
			}
			b.WriteString(trimmed)
		}
		text = b.String()
	}

	// Ensure that every sentence segment after a period has its first letter capitalized
	// because Sherpa-ONNX's sentence segmenter only treats uppercase followers as sentence boundaries
	// (ignoring lowercase to prevent splitting on abbreviations like e.g. or i.e.).
	if strings.Contains(text, ". ") {
		parts := strings.Split(text, ". ")
		var b strings.Builder
		for i, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			if i > 0 {
				b.WriteString(". ")
			}
			r, sz := utf8.DecodeRuneInString(trimmed)
			b.WriteString(string(unicode.ToUpper(r)) + trimmed[sz:])
		}
		text = b.String()
	}

	return text
}
