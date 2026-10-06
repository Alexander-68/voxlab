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
			// Buffer full, drop oldest chunk to maintain real-time latency
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

// WavFileSource streams audio from a WAV file at real-time rate for test bench evaluation.
type WavFileSource struct {
	filePath     string
	sampleRate   int
	chunkSamples int
	cancel       context.CancelFunc
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
		resampled := make([]float32, int(float64(len(samples))*float64(s.sampleRate)/float64(wavRate)))
		ratio := float64(len(samples)) / float64(len(resampled))
		for i := range resampled {
			srcIdx := int(float64(i) * ratio)
			if srcIdx >= len(samples) {
				srcIdx = len(samples) - 1
			}
			resampled[i] = samples[srcIdx]
		}
		samples = resampled
	}

	subCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.running = true
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
		}()

		chunkDuration := time.Duration(float64(s.chunkSamples)/float64(s.sampleRate)*1000) * time.Millisecond
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

				select {
				case out <- chunk:
				case <-subCtx.Done():
					return
				default:
				}
			}
		}
	}()

	return nil
}

func (s *WavFileSource) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.running = false
	return nil
}

// HostNativeSource captures live microphone audio directly from host OS hardware (ALSA on Linux, DirectShow on Windows).
type HostNativeSource struct {
	sampleRate   int
	chunkSamples int
	cancel       context.CancelFunc
	running      bool
	mu           sync.Mutex
	deviceName   string
	cmd          *exec.Cmd
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
	cmd := exec.Command("ffmpeg", "-list_devices", "true", "-f", "dshow", "-i", "dummy")
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
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("host mic already running")
	}

	subCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.running = true

	// Check if ffmpeg or arecord is available
	var captureCmd *exec.Cmd
	devName := "simulated"

	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("ffmpeg"); err == nil {
			mic := detectWindowsDirectShowMic()
			if mic != "" {
				devName = mic
				captureCmd = exec.CommandContext(subCtx, "ffmpeg",
					"-f", "dshow",
					"-audio_buffer_size", "20",
					"-fflags", "nobuffer",
					"-flags", "low_delay",
					"-i", "audio="+mic,
					"-ar", "16000",
					"-ac", "1",
					"-f", "s16le",
					"-flush_packets", "1",
					"pipe:1",
				)
			}
		}
	} else {
		// Linux: try arecord or ffmpeg
		if _, err := exec.LookPath("arecord"); err == nil {
			devName = "default"
			captureCmd = exec.CommandContext(subCtx, "arecord",
				"-q",
				"-r", "16000",
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
				"-ar", "16000",
				"-ac", "1",
				"-f", "s16le",
				"-flush_packets", "1",
				"pipe:1",
			)
		}
	}

	s.deviceName = devName
	s.cmd = captureCmd
	s.mu.Unlock()

	log.Printf("[Audio] Starting Host Native Mic: %s", devName)

	if captureCmd != nil {
		stdout, err := captureCmd.StdoutPipe()
		if err != nil {
			log.Printf("[Audio] Pipe error: %v, falling back to simulation", err)
			s.runSimulatedLoop(subCtx, out)
			return nil
		}

		if err := captureCmd.Start(); err != nil {
			log.Printf("[Audio] Host capture process error: %v, falling back to simulation", err)
			s.runSimulatedLoop(subCtx, out)
			return nil
		}

		go func() {
			defer func() {
				if captureCmd.Process != nil {
					_ = captureCmd.Process.Kill()
				}
				s.mu.Lock()
				s.running = false
				s.mu.Unlock()
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
	go s.runSimulatedLoop(subCtx, out)
	return nil
}

func (s *HostNativeSource) runSimulatedLoop(ctx context.Context, out chan<- []float32) {
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	chunkDuration := time.Duration(float64(s.chunkSamples)/float64(s.sampleRate)*1000) * time.Millisecond
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

func (s *HostNativeSource) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		s.cmd = nil
	}
	s.running = false
	return nil
}
