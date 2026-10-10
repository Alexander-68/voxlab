//go:build windows

package audio

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"

	"voxlab/pkg/config"
)

type cGtcrnModelConfig struct {
	Model uintptr
}

type cDpdfNetModelConfig struct {
	Model              uintptr
	AttenuationLimitDb float32
	_pad               uint32
}

type cDenoiserModelConfig struct {
	Gtcrn      cGtcrnModelConfig
	NumThreads int32
	Debug      int32
	Provider   uintptr
	Dpdfnet    cDpdfNetModelConfig
}

type cOnlineSpeechDenoiserConfig struct {
	Model cDenoiserModelConfig
}

type cDenoisedAudio struct {
	Samples    *float32
	N          int32
	SampleRate int32
}

func toCStringEnhancer(s string, keep *[][]byte) uintptr {
	if s == "" {
		return 0
	}
	b := append([]byte(s), 0)
	*keep = append(*keep, b)
	return uintptr(unsafe.Pointer(&b[0]))
}

type windowsSpeechEnhancer struct {
	mu             sync.Mutex
	dll            *syscall.LazyDLL
	createFn       *syscall.LazyProc
	destroyFn      *syscall.LazyProc
	getSampleRate  *syscall.LazyProc
	getFrameShift  *syscall.LazyProc
	runFn          *syscall.LazyProc
	flushFn        *syscall.LazyProc
	resetFn        *syscall.LazyProc
	destroyAudioFn *syscall.LazyProc

	dllDir     string
	modelDir   string
	numThreads int
	provider   string

	handle     uintptr
	modelType  string
	modelPath  string
	sampleRate int
	frameShift int
	fifo       []float32
	enabled    bool
}

func newPlatformSpeechEnhancer(dllDir string, modelDir string, cfg config.EnhancerConfig) (SpeechEnhancer, error) {
	const dllName = "sherpa-onnx-c-api.dll"
	absDllDir, err := filepath.Abs(dllDir)
	if err != nil {
		absDllDir = dllDir
	}
	absDllPath := filepath.Join(absDllDir, dllName)
	if _, err := os.Stat(absDllPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("c-api dll not found at %s", absDllPath)
	}

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setDllDir := kernel32.NewProc("SetDllDirectoryW")
	setDllDir.Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(absDllDir))))

	dll := syscall.NewLazyDLL(absDllPath)
	if err := dll.Load(); err != nil {
		return nil, fmt.Errorf("failed to load dll %s: %w", absDllPath, err)
	}

	createFn := dll.NewProc("SherpaOnnxCreateOnlineSpeechDenoiser")
	if err := createFn.Find(); err != nil {
		return nil, fmt.Errorf("symbol SherpaOnnxCreateOnlineSpeechDenoiser not found: %w", err)
	}
	destroyFn := dll.NewProc("SherpaOnnxDestroyOnlineSpeechDenoiser")
	getSampleRate := dll.NewProc("SherpaOnnxOnlineSpeechDenoiserGetSampleRate")
	getFrameShift := dll.NewProc("SherpaOnnxOnlineSpeechDenoiserGetFrameShiftInSamples")
	runFn := dll.NewProc("SherpaOnnxOnlineSpeechDenoiserRun")
	flushFn := dll.NewProc("SherpaOnnxOnlineSpeechDenoiserFlush")
	resetFn := dll.NewProc("SherpaOnnxOnlineSpeechDenoiserReset")
	destroyAudioFn := dll.NewProc("SherpaOnnxDestroyDenoisedAudio")

	modelType := cfg.ModelType
	if modelType == "" {
		modelType = "gtcrn"
	}
	numThreads := cfg.NumThreads
	if numThreads <= 0 {
		numThreads = 1
	}
	provider := cfg.Provider
	if provider == "" {
		provider = "cpu"
	}

	enhancer := &windowsSpeechEnhancer{
		dll:            dll,
		createFn:       createFn,
		destroyFn:      destroyFn,
		getSampleRate:  getSampleRate,
		getFrameShift:  getFrameShift,
		runFn:          runFn,
		flushFn:        flushFn,
		resetFn:        resetFn,
		destroyAudioFn: destroyAudioFn,
		dllDir:         absDllDir,
		modelDir:       modelDir,
		numThreads:     numThreads,
		provider:       provider,
		modelType:      modelType,
		enabled:        cfg.Enabled,
	}

	// Attempt to initialize model
	if err := enhancer.initModel(modelType, cfg.ModelPath); err != nil {
		// Log warning, but keep enhancer initialized so user can toggle later
		log.Printf("[SpeechEnhancer] Initial model load failed (%v); will stay standby", err)
	}

	return enhancer, nil
}

func (w *windowsSpeechEnhancer) resolveModelPath(modelType string, preferredPath string) (string, error) {
	if preferredPath != "" {
		if _, err := os.Stat(preferredPath); err == nil {
			return filepath.Abs(preferredPath)
		}
	}

	candidates := []string{}
	switch modelType {
	case "dpdfnet":
		candidates = []string{
			filepath.Join(w.modelDir, "dpdfnet_baseline.onnx"),
			filepath.Join(w.modelDir, "dpdfnet4.onnx"),
			filepath.Join(w.modelDir, "dpdfnet2.onnx"),
			filepath.Join("models", "dpdfnet_baseline.onnx"),
		}
	case "gtcrn":
		fallthrough
	default:
		candidates = []string{
			filepath.Join(w.modelDir, "gtcrn_simple.onnx"),
			filepath.Join("models", "gtcrn_simple.onnx"),
		}
	}

	for _, cand := range candidates {
		if _, err := os.Stat(cand); err == nil {
			return filepath.Abs(cand)
		}
	}

	return "", fmt.Errorf("no model file found for %s in %s", modelType, w.modelDir)
}

func (w *windowsSpeechEnhancer) initModel(modelType string, preferredPath string) error {
	mPath, err := w.resolveModelPath(modelType, preferredPath)
	if err != nil {
		return err
	}

	var keep [][]byte
	var cfg cOnlineSpeechDenoiserConfig
	cfg.Model.NumThreads = int32(w.numThreads)
	cfg.Model.Provider = toCStringEnhancer(w.provider, &keep)

	switch modelType {
	case "dpdfnet":
		cfg.Model.Dpdfnet.Model = toCStringEnhancer(mPath, &keep)
	case "gtcrn":
		fallthrough
	default:
		modelType = "gtcrn"
		cfg.Model.Gtcrn.Model = toCStringEnhancer(mPath, &keep)
	}

	handle, _, callErr := w.createFn.Call(uintptr(unsafe.Pointer(&cfg)))
	if handle == 0 {
		return fmt.Errorf("failed to create denoiser for %s: %v", mPath, callErr)
	}

	if w.handle != 0 {
		w.destroyFn.Call(w.handle)
	}

	w.handle = handle
	w.modelType = modelType
	w.modelPath = mPath

	sr, _, _ := w.getSampleRate.Call(handle)
	fs, _, _ := w.getFrameShift.Call(handle)
	w.sampleRate = int(sr)
	w.frameShift = int(fs)
	if w.sampleRate <= 0 {
		w.sampleRate = 16000
	}
	if w.frameShift <= 0 {
		w.frameShift = 256
	}

	w.primeWithSilence()

	log.Printf("[SpeechEnhancer] Loaded %s (%s, SampleRate: %d Hz, FrameShift: %d samples, Threads: %d)",
		w.nameLocked(), filepath.Base(mPath), w.sampleRate, w.frameShift, w.numThreads)
	return nil
}

func (w *windowsSpeechEnhancer) primeWithSilence() {
	if w.handle == 0 {
		return
	}
	w.fifo = w.fifo[:0]

	// Feed 2 * frameShift of silence so network receptive field is warmed
	primeSamples := make([]float32, 2*w.frameShift)
	outPtr, _, _ := w.runFn.Call(
		w.handle,
		uintptr(unsafe.Pointer(&primeSamples[0])),
		uintptr(len(primeSamples)),
		uintptr(w.sampleRate),
	)
	if outPtr != 0 {
		audio := (*cDenoisedAudio)(unsafe.Pointer(outPtr))
		if audio.N > 0 && audio.Samples != nil {
			src := unsafe.Slice(audio.Samples, audio.N)
			w.fifo = append(w.fifo, src...)
		}
		w.destroyAudioFn.Call(outPtr)
	}
}

func (w *windowsSpeechEnhancer) nameLocked() string {
	switch w.modelType {
	case "gtcrn":
		return "GTCRN Neural Denoiser"
	case "dpdfnet":
		return "DPDFNet Deep Filter Denoiser"
	default:
		return "Speech Enhancer"
	}
}

func (w *windowsSpeechEnhancer) Name() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.nameLocked()
}

func (w *windowsSpeechEnhancer) ModelType() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.modelType
}

func (w *windowsSpeechEnhancer) ModelPath() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.modelPath
}

func (w *windowsSpeechEnhancer) IsEnabled() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.enabled && w.handle != 0
}

func (w *windowsSpeechEnhancer) SetEnabled(enabled bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.enabled = enabled
	if enabled {
		if w.handle == 0 {
			_ = w.initModel(w.modelType, "")
		} else {
			w.resetFn.Call(w.handle)
			w.primeWithSilence()
		}
	} else {
		w.fifo = w.fifo[:0]
	}
}

func (w *windowsSpeechEnhancer) SwitchModel(modelType string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if modelType == w.modelType && w.handle != 0 {
		return nil
	}
	return w.initModel(modelType, "")
}

func (w *windowsSpeechEnhancer) ProcessChunk(chunk []float32) []float32 {
	if len(chunk) == 0 {
		return chunk
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.enabled || w.handle == 0 {
		return chunk
	}

	outPtr, _, _ := w.runFn.Call(
		w.handle,
		uintptr(unsafe.Pointer(&chunk[0])),
		uintptr(len(chunk)),
		uintptr(w.sampleRate),
	)
	if outPtr != 0 {
		audio := (*cDenoisedAudio)(unsafe.Pointer(outPtr))
		if audio.N > 0 && audio.Samples != nil {
			src := unsafe.Slice(audio.Samples, audio.N)
			w.fifo = append(w.fifo, src...)
		}
		w.destroyAudioFn.Call(outPtr)
	}

	needed := len(chunk)
	if len(w.fifo) >= needed {
		out := make([]float32, needed)
		copy(out, w.fifo[:needed])
		w.fifo = w.fifo[needed:]
		return out
	}

	// If FIFO doesn't yet have enough samples (warming phase), pad with available samples
	// and fallback to original chunk elements to guarantee no sample starvation
	out := make([]float32, needed)
	have := len(w.fifo)
	if have > 0 {
		copy(out[:have], w.fifo)
		w.fifo = w.fifo[:0]
		copy(out[have:], chunk[have:])
	} else {
		copy(out, chunk)
	}
	return out
}

func (w *windowsSpeechEnhancer) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.handle != 0 {
		w.resetFn.Call(w.handle)
		w.primeWithSilence()
	}
}

func (w *windowsSpeechEnhancer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.handle != 0 {
		w.destroyFn.Call(w.handle)
		w.handle = 0
	}
	return nil
}
