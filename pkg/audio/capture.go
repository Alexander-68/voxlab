package audio

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// AudioSource defines the interface for streaming audio into VoxLab.
type AudioSource interface {
	Name() string
	Start(ctx context.Context, out chan<- []float32) error
	Stop() error
}

// WebSocketSource receives audio chunks from the Web UI mic via WebSocket.
type WebSocketSource struct {
	mu      sync.Mutex
	outChan chan<- []float32
	active  bool
}

// NewWebSocketSource creates a new WebSocketSource.
func NewWebSocketSource() *WebSocketSource {
	return &WebSocketSource{}
}

func (s *WebSocketSource) Name() string { return "web_ui_mic" }

func (s *WebSocketSource) Start(ctx context.Context, out chan<- []float32) error {
	s.mu.Lock()
	s.outChan = out
	s.active = true
	s.mu.Unlock()
	return nil
}

func (s *WebSocketSource) PushChunk(samples []float32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active && s.outChan != nil {
		select {
		case s.outChan <- samples:
		default:
			// Buffer full: drop this (newest) chunk to maintain real-time latency
		}
	}
}

func (s *WebSocketSource) Stop() error {
	s.mu.Lock()
	s.active = false
	s.outChan = nil
	s.mu.Unlock()
	return nil
}

// chunkInterval is the real-time duration of one chunk, kept at ns precision (ms truncation made
// e.g. 160 @ 44.1 kHz play ~20% fast) and never <= 0 (time.NewTicker panics).
func chunkInterval(chunkSamples, sampleRate int) time.Duration {
	if chunkSamples <= 0 || sampleRate <= 0 {
		return 20 * time.Millisecond
	}
	return time.Duration(chunkSamples) * time.Second / time.Duration(sampleRate)
}

// WavFileSource streams audio from a WAV file at real-time rate for test bench evaluation.
type WavFileSource struct {
	filePath     string
	sampleRate   int
	chunkSamples int
	cancel       context.CancelFunc
	done         chan struct{} // closed when the streaming goroutine exits
	running      bool
	mu           sync.Mutex
	loop         bool
}

// NewWavFileSource creates a new WavFileSource.
func NewWavFileSource(filePath string, sampleRate int, chunkSamples int, loop bool) *WavFileSource {
	return &WavFileSource{
		filePath:     filePath,
		sampleRate:   sampleRate,
		chunkSamples: chunkSamples,
		loop:         loop,
	}
}

func (s *WavFileSource) Name() string { return "wav_file_injection" }

func (s *WavFileSource) Start(ctx context.Context, out chan<- []float32) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("wav source already running")
	}

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		s.mu.Unlock()
		return err
	}

	samples, wavRate, err := DecodeWAV(bytes.NewReader(data))
	if err != nil {
		s.mu.Unlock()
		return err
	}

	// Resample if needed
	if wavRate != s.sampleRate && wavRate > 0 {
		samples = Resample(samples, wavRate, s.sampleRate)
	}
	if len(samples) == 0 {
		s.mu.Unlock()
		return errors.New("wav file contains no audio samples")
	}

	subCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	s.cancel = cancel
	s.done = done
	s.running = true
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
			cancel()
			close(done)
		}()

		chunkDuration := chunkInterval(s.chunkSamples, s.sampleRate)
		ticker := time.NewTicker(chunkDuration)
		defer ticker.Stop()

		idx := 0
		for {
			select {
			case <-subCtx.Done():
				return
			case <-ticker.C:
				if idx >= len(samples) {
					if s.loop {
						idx = 0 // loop
					} else {
						return // End of file
					}
				}
				end := idx + s.chunkSamples
				if end > len(samples) {
					end = len(samples)
				}
				chunk := samples[idx:end]
				idx = end

				// Block (bounded by the ticker cadence) rather than drop: lost chunks corrupt evaluation.
				select {
				case out <- chunk:
				case <-subCtx.Done():
					return
				}
			}
		}
	}()

	return nil
}

// Stop cancels streaming and waits for the goroutine to exit, so a following Start cannot overlap it.
func (s *WavFileSource) Stop() error {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	done := s.done
	s.done = nil
	s.mu.Unlock()
	if done != nil {
		<-done
	}
	return nil
}

// HostNativeSource captures live microphone audio directly from host OS hardware (ALSA on Linux, DirectShow on Windows).
type HostNativeSource struct {
	sampleRate   int
	chunkSamples int
	cancel       context.CancelFunc
	done         chan struct{} // closed when the capture goroutine exits
	running      bool
	mu           sync.Mutex
	deviceName   string
}

// NewHostNativeSource creates a new host native audio capture source.
func NewHostNativeSource(sampleRate, chunkSamples int) *HostNativeSource {
	return &HostNativeSource{
		sampleRate:   sampleRate,
		chunkSamples: chunkSamples,
	}
}

func (s *HostNativeSource) Name() string { return "host_native_mic" }

// DeviceName returns the active device name detected.
func (s *HostNativeSource) DeviceName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deviceName
}

// detectWindowsDirectShowMic searches for available DirectShow microphones.
func detectWindowsDirectShowMic() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg", "-list_devices", "true", "-f", "dshow", "-i", "dummy")
	out, _ := cmd.CombinedOutput()
	outputStr := string(out)

	// Search for: "Microphone ..." (audio)
	re := regexp.MustCompile(`"([^"]+)"\s+\(audio\)`)
	matches := re.FindAllStringSubmatch(outputStr, -1)
	if len(matches) > 0 {
		return matches[0][1] // Return first detected microphone
	}
	return ""
}

func (s *HostNativeSource) Start(ctx context.Context, out chan<- []float32) error {
	// Device detection runs ffmpeg; keep it outside s.mu so a slow probe cannot block Stop/DeviceName.
	mic := ""
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("ffmpeg"); err == nil {
			mic = detectWindowsDirectShowMic()
		}
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("host mic already running")
	}

	subCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	s.cancel = cancel
	s.done = done
	s.running = true
	rate := strconv.Itoa(s.sampleRate)

	// Check if ffmpeg or arecord is available
	var captureCmd *exec.Cmd
	devName := "simulated"

	if runtime.GOOS == "windows" {
		if mic != "" {
			devName = mic
			captureCmd = exec.CommandContext(subCtx, "ffmpeg",
				"-f", "dshow",
				"-audio_buffer_size", "20",
				"-fflags", "nobuffer",
				"-flags", "low_delay",
				"-i", "audio="+mic,
				"-ar", rate,
				"-ac", "1",
				"-f", "s16le",
				"-flush_packets", "1",
				"pipe:1",
			)
		}
	} else {
		// Linux: try arecord or ffmpeg
		if _, err := exec.LookPath("arecord"); err == nil {
			devName = "default"
			captureCmd = exec.CommandContext(subCtx, "arecord",
				"-q",
				"-r", rate,
				"-c", "1",
				"-f", "S16_LE",
				"-t", "raw",
			)
		} else if _, err := exec.LookPath("ffmpeg"); err == nil {
			devName = "pulse/alsa"
			captureCmd = exec.CommandContext(subCtx, "ffmpeg",
				"-f", "pulse",
				"-fflags", "nobuffer",
				"-flags", "low_delay",
				"-i", "default",
				"-ar", rate,
				"-ac", "1",
				"-f", "s16le",
				"-flush_packets", "1",
				"pipe:1",
			)
		}
	}

	s.deviceName = devName
	s.mu.Unlock()

	log.Printf("[Audio] Starting Host Native Mic: %s", devName)

	if captureCmd != nil {
		stdout, err := captureCmd.StdoutPipe()
		if err != nil {
			log.Printf("[Audio] Pipe error: %v, falling back to simulation", err)
			go s.runSimulatedLoop(subCtx, cancel, done, out)
			return nil
		}

		if err := captureCmd.Start(); err != nil {
			log.Printf("[Audio] Host capture process error: %v, falling back to simulation", err)
			go s.runSimulatedLoop(subCtx, cancel, done, out)
			return nil
		}

		go func() {
			defer func() {
				_ = captureCmd.Process.Kill()
				_ = captureCmd.Wait() // reap the child and release its pipes
				s.mu.Lock()
				s.running = false
				s.mu.Unlock()
				cancel()
				close(done)
			}()

			bytesPerChunk := s.chunkSamples * 2 // 16-bit = 2 bytes per sample
			chunkBytes := make([]byte, bytesPerChunk)

			for {
				select {
				case <-subCtx.Done():
					return
				default:
					_, err := io.ReadFull(stdout, chunkBytes)
					if err != nil {
						return
					}
					samples := BytesToFloat32PCM(chunkBytes)
					select {
					case out <- samples:
					case <-subCtx.Done():
						return
					}
				}
			}
		}()
		return nil
	}

	// Fallback to simulated audio if no native capture tool
	go s.runSimulatedLoop(subCtx, cancel, done, out)
	return nil
}

func (s *HostNativeSource) runSimulatedLoop(ctx context.Context, cancel context.CancelFunc, done chan struct{}, out chan<- []float32) {
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		cancel()
		close(done)
	}()

	chunkDuration := chunkInterval(s.chunkSamples, s.sampleRate)
	ticker := time.NewTicker(chunkDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Ambient noise simulation
			chunk := make([]float32, s.chunkSamples)
			for i := range chunk {
				chunk[i] = float32(0.001 * (float64(i%7) - 3.0))
			}
			select {
			case out <- chunk:
			case <-ctx.Done():
				return
			default:
			}
		}
	}
}

// Stop cancels capture (CommandContext kills the child) and waits for the goroutine to exit,
// so a following Start cannot overlap the old run or have its running flag cleared by it.
func (s *HostNativeSource) Stop() error {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	done := s.done
	s.done = nil
	s.mu.Unlock()
	if done != nil {
		<-done
	}
	return nil
}
