//go:build windows

package engine

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"voxlab/pkg/audio"
)

type cVitsModelConfig struct {
	Model       uintptr
	Lexicon     uintptr
	Tokens      uintptr
	DataDir     uintptr
	NoiseScale  float32
	NoiseScaleW float32
	LengthScale float32
	_pad        uint32
	DictDir     uintptr
}

type cMatchaModelConfig struct {
	AcousticModel uintptr
	Vocoder       uintptr
	Lexicon       uintptr
	Tokens        uintptr
	DataDir       uintptr
	NoiseScale    float32
	LengthScale   float32
	DictDir       uintptr
}

type cKokoroModelConfig struct {
	Model       uintptr
	Voices      uintptr
	Tokens      uintptr
	DataDir     uintptr
	LengthScale float32
	_pad        uint32
	DictDir     uintptr
	Lexicon     uintptr
	Lang        uintptr
}

type cKittenModelConfig struct {
	Model       uintptr
	Voices      uintptr
	Tokens      uintptr
	DataDir     uintptr
	LengthScale float32
	_pad        uint32
}

type cZipvoiceModelConfig struct {
	Tokens        uintptr
	Encoder       uintptr
	Decoder       uintptr
	Vocoder       uintptr
	DataDir       uintptr
	Lexicon       uintptr
	FeatScale     float32
	TShift        float32
	TargetRms     float32
	GuidanceScale float32
}

type cPocketModelConfig struct {
	LmFlow                      uintptr
	LmMain                      uintptr
	Encoder                     uintptr
	Decoder                     uintptr
	TextConditioner             uintptr
	VocabJson                   uintptr
	TokenScoresJson             uintptr
	VoiceEmbeddingCacheCapacity int32
	_pad                        uint32
}

type cSupertonicModelConfig struct {
	DurationPredictor uintptr
	TextEncoder       uintptr
	VectorEstimator   uintptr
	Vocoder           uintptr
	TtsJson           uintptr
	UnicodeIndexer    uintptr
	VoiceStyle        uintptr
}

type cOfflineTtsModelConfig struct {
	Vits       cVitsModelConfig
	NumThreads int32
	Debug      int32
	Provider   uintptr
	Matcha     cMatchaModelConfig
	Kokoro     cKokoroModelConfig
	Kitten     cKittenModelConfig
	Zipvoice   cZipvoiceModelConfig
	Pocket     cPocketModelConfig
	Supertonic cSupertonicModelConfig
}

type cOfflineTtsConfig struct {
	Model           cOfflineTtsModelConfig
	RuleFsts        uintptr
	MaxNumSentences int32
	_pad1           uint32
	RuleFars        uintptr
	SilenceScale    float32
	_pad2           uint32
}

type cGeneratedAudio struct {
	Samples    *float32
	N          int32
	SampleRate int32
}

func toCString(s string, keepAlive *[][]byte) uintptr {
	if s == "" {
		return 0
	}
	b := append([]byte(s), 0)
	*keepAlive = append(*keepAlive, b)
	return uintptr(unsafe.Pointer(&b[0]))
}

type windowsWarmTTS struct {
	dll            *syscall.LazyDLL
	createFn       *syscall.LazyProc
	destroyFn      *syscall.LazyProc
	genFn          *syscall.LazyProc
	genWithCbFn    *syscall.LazyProc
	destroyAudioFn *syscall.LazyProc
	handle         uintptr
	modelKey       string
	mu             sync.Mutex
}

var (
	warmTtsCbMu sync.Mutex
	warmTtsCbFn func(samples []float32, progress float32) error
)

func warmTtsProgressCallback(samplesPtr uintptr, numSamples int32, progressBits uintptr, arg uintptr) uintptr {
	warmTtsCbMu.Lock()
	fn := warmTtsCbFn
	warmTtsCbMu.Unlock()

	if fn != nil && numSamples > 0 && samplesPtr != 0 {
		src := unsafe.Slice((*float32)(unsafe.Pointer(samplesPtr)), int(numSamples))
		chunk := make([]float32, numSamples)
		copy(chunk, src)
		progress := math.Float32frombits(uint32(progressBits))
		if err := fn(chunk, progress); err != nil {
			return 0 // Abort synthesis immediately
		}
	}
	return 1
}

var warmTtsCallbackPtr = syscall.NewCallback(warmTtsProgressCallback)

func (w *windowsWarmTTS) IsWarm() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.handle != 0
}

func (w *windowsWarmTTS) ModelKey() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.modelKey
}

func (w *windowsWarmTTS) Synthesize(text string, sid int, speed float64) (*TTSResult, error) {
	return w.SynthesizeStream(text, sid, speed, nil)
}

func (w *windowsWarmTTS) SynthesizeStream(text string, sid int, speed float64, onChunk func(chunk TTSChunk) error) (*TTSResult, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.handle == 0 {
		return nil, fmt.Errorf("warm TTS engine handle is not initialized")
	}

	if speed <= 0 {
		speed = 1.0
	}

	startTime := time.Now()
	var keep [][]byte
	textPtr := toCString(text, &keep)
	speedBits := math.Float32bits(float32(speed))

	// If streaming callback requested and SherpaOnnxOfflineTtsGenerateWithCallback procedure is available
	if onChunk != nil && w.genWithCbFn != nil && w.genWithCbFn.Find() == nil {
		var (
			chunkIdx   int
			sampleRate = 24000
			cbErr      error
		)

		warmTtsCbMu.Lock()
		warmTtsCbFn = func(samples []float32, progress float32) error {
			chunkIdx++
			dur := float64(len(samples)) / float64(sampleRate)
			lat := time.Since(startTime).Milliseconds()
			wav, err := audio.EncodeWAV(samples, sampleRate)
			if err != nil {
				return err
			}
			c := TTSChunk{
				Index:        chunkIdx - 1,
				IsLast:       false,
				AudioSamples: samples,
				WAVBytes:     wav,
				SampleRate:   sampleRate,
				DurationSec:  dur,
				LatencyMs:    lat,
			}
			if err := onChunk(c); err != nil {
				cbErr = err
				return err
			}
			return nil
		}
		warmTtsCbMu.Unlock()

		audioPtr, _, callErr := w.genWithCbFn.Call(
			w.handle,
			textPtr,
			uintptr(sid),
			uintptr(speedBits),
			warmTtsCallbackPtr,
			0,
		)

		warmTtsCbMu.Lock()
		warmTtsCbFn = nil
		warmTtsCbMu.Unlock()

		if cbErr != nil {
			if audioPtr != 0 {
				w.destroyAudioFn.Call(audioPtr)
			}
			return nil, fmt.Errorf("streaming synthesis aborted: %w", cbErr)
		}

		if audioPtr == 0 {
			return nil, fmt.Errorf("sherpa-onnx c-api streaming synthesis failed: %v", callErr)
		}

		audioStruct := (*cGeneratedAudio)(unsafe.Pointer(audioPtr))
		if audioStruct == nil || audioStruct.N <= 0 || audioStruct.Samples == nil {
			w.destroyAudioFn.Call(audioPtr)
			return nil, fmt.Errorf("synthesis returned empty audio waveform")
		}

		sampleRate = int(audioStruct.SampleRate)
		samples := make([]float32, audioStruct.N)
		src := unsafe.Slice(audioStruct.Samples, audioStruct.N)
		copy(samples, src)

		w.destroyAudioFn.Call(audioPtr)

		latency := time.Since(startTime).Milliseconds()
		// If native engine did not fire callbacks (single sentence), dispatch as chunk 0
		if chunkIdx == 0 {
			wav, _ := audio.EncodeWAV(samples, sampleRate)
			_ = onChunk(TTSChunk{
				Index:        0,
				IsLast:       true,
				AudioSamples: samples,
				WAVBytes:     wav,
				SampleRate:   sampleRate,
				DurationSec:  float64(len(samples)) / float64(sampleRate),
				LatencyMs:    latency,
			})
		}

		return float32SliceToTTSResult(samples, sampleRate, latency)
	}

	// Standard non-streaming path
	audioPtr, _, err := w.genFn.Call(
		w.handle,
		textPtr,
		uintptr(sid),
		uintptr(speedBits),
	)

	latency := time.Since(startTime).Milliseconds()
	if audioPtr == 0 {
		return nil, fmt.Errorf("sherpa-onnx c-api synthesis failed: %v", err)
	}

	audioStruct := (*cGeneratedAudio)(unsafe.Pointer(audioPtr))
	if audioStruct == nil || audioStruct.N <= 0 || audioStruct.Samples == nil {
		w.destroyAudioFn.Call(audioPtr)
		return nil, fmt.Errorf("synthesis returned empty audio waveform")
	}

	samples := make([]float32, audioStruct.N)
	src := unsafe.Slice(audioStruct.Samples, audioStruct.N)
	copy(samples, src)
	sampleRate := int(audioStruct.SampleRate)

	w.destroyAudioFn.Call(audioPtr)

	res, err := float32SliceToTTSResult(samples, sampleRate, latency)
	if err == nil && onChunk != nil {
		_ = onChunk(TTSChunk{
			Index:        0,
			IsLast:       true,
			AudioSamples: samples,
			WAVBytes:     res.WAVBytes,
			SampleRate:   sampleRate,
			DurationSec:  res.DurationSec,
			LatencyMs:    latency,
		})
	}
	return res, err
}

func (w *windowsWarmTTS) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.handle != 0 {
		w.destroyFn.Call(w.handle)
		w.handle = 0
	}
	return nil
}

// newPlatformWarmTTS loads sherpa-onnx-c-api.dll and initializes a persistent in-memory TTS engine.
func newPlatformWarmTTS(
	dllDir string,
	modelFile string,
	voicesPath string,
	tokensPath string,
	dataDirPath string,
	lexPath string,
	ruleFsts string,
	numThreads int,
	provider string,
) (WarmTTS, error) {
	const dllName = "sherpa-onnx-c-api.dll"
	absDllDir, err := filepath.Abs(dllDir)
	if err != nil {
		absDllDir = dllDir
	}
	absDllPath := filepath.Join(absDllDir, dllName)
	if _, err := os.Stat(absDllPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("c-api dll not found at %s", absDllPath)
	}

	// Ensure Windows searches dllDir before System32 so that the co-located onnxruntime.dll
	// is loaded rather than any incompatible system-installed onnxruntime.dll
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setDllDir := kernel32.NewProc("SetDllDirectoryW")
	setDllDir.Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(absDllDir))))

	dll := syscall.NewLazyDLL(absDllPath)
	if err := dll.Load(); err != nil {
		return nil, fmt.Errorf("failed to load dll %s: %w", absDllPath, err)
	}

	createFn := dll.NewProc("SherpaOnnxCreateOfflineTts")
	if err := createFn.Find(); err != nil {
		return nil, fmt.Errorf("symbol SherpaOnnxCreateOfflineTts not found in %s: %w", absDllPath, err)
	}
	destroyFn := dll.NewProc("SherpaOnnxDestroyOfflineTts")
	genFn := dll.NewProc("SherpaOnnxOfflineTtsGenerate")
	genWithCbFn := dll.NewProc("SherpaOnnxOfflineTtsGenerateWithCallback")
	destroyAudioFn := dll.NewProc("SherpaOnnxDestroyOfflineTtsGeneratedAudio")

	if numThreads <= 0 {
		numThreads = 4
	}
	if provider == "" {
		provider = "cpu"
	}

	absModelFile, _ := filepath.Abs(modelFile)
	absVoicesPath, _ := filepath.Abs(voicesPath)
	absTokensPath, _ := filepath.Abs(tokensPath)
	absDataDirPath, _ := filepath.Abs(dataDirPath)
	var absLexPath string
	if lexPath != "" {
		parts := strings.Split(lexPath, ",")
		for i, p := range parts {
			if a, err := filepath.Abs(strings.TrimSpace(p)); err == nil {
				parts[i] = a
			}
		}
		absLexPath = strings.Join(parts, ",")
	}

	var keep [][]byte
	var cfg cOfflineTtsConfig
	cfg.Model.NumThreads = int32(numThreads)
	cfg.Model.Provider = toCString(provider, &keep)
	cfg.Model.Kokoro.Model = toCString(absModelFile, &keep)
	cfg.Model.Kokoro.Voices = toCString(absVoicesPath, &keep)
	cfg.Model.Kokoro.Tokens = toCString(absTokensPath, &keep)
	cfg.Model.Kokoro.DataDir = toCString(absDataDirPath, &keep)
	cfg.Model.Kokoro.Lexicon = toCString(absLexPath, &keep)
	cfg.Model.Kokoro.LengthScale = 1.0
	if ruleFsts != "" {
		parts := strings.Split(ruleFsts, ",")
		for i, p := range parts {
			if a, err := filepath.Abs(strings.TrimSpace(p)); err == nil {
				parts[i] = a
			}
		}
		cfg.RuleFsts = toCString(strings.Join(parts, ","), &keep)
	}
	cfg.MaxNumSentences = 1
	cfg.SilenceScale = 0.2

	t0 := time.Now()
	ttsPtr, _, callErr := createFn.Call(uintptr(unsafe.Pointer(&cfg)))
	if ttsPtr == 0 {
		return nil, fmt.Errorf("SherpaOnnxCreateOfflineTts returned NULL: %v", callErr)
	}

	key := fmt.Sprintf("%s|t%d|%s", modelFile, numThreads, provider)
	log.Printf("[WarmTTS] Preloaded Kokoro TTS into resident RAM in %v (file: %s, threads: %d, provider: %s)",
		time.Since(t0), filepath.Base(modelFile), numThreads, provider)

	return &windowsWarmTTS{
		dll:            dll,
		createFn:       createFn,
		destroyFn:      destroyFn,
		genFn:          genFn,
		genWithCbFn:    genWithCbFn,
		destroyAudioFn: destroyAudioFn,
		handle:         ttsPtr,
		modelKey:       key,
	}, nil
}
