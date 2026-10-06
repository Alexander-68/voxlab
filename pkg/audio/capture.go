package audio

import (
	"bytes"
	"context"
	"errors"
	"os"
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
}

// NewWavFileSource creates a new WavFileSource.
func NewWavFileSource(filePath string, sampleRate int, chunkSamples int) *WavFileSource {
	return &WavFileSource{
		filePath:     filePath,
		sampleRate:   sampleRate,
		chunkSamples: chunkSamples,
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

	// Simple resample if needed
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
					return // End of file
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

// HostSimulatedSource simulates host audio capture when physical microphone capture is unavailable or requested.
type HostSimulatedSource struct {
	sampleRate   int
	chunkSamples int
	cancel       context.CancelFunc
	running      bool
	mu           sync.Mutex
}

// NewHostSimulatedSource creates a new host simulated audio source.
func NewHostSimulatedSource(sampleRate, chunkSamples int) *HostSimulatedSource {
	return &HostSimulatedSource{
		sampleRate:   sampleRate,
		chunkSamples: chunkSamples,
	}
}

func (s *HostSimulatedSource) Name() string { return "host_native_mic" }

func (s *HostSimulatedSource) Start(ctx context.Context, out chan<- []float32) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("host mic already running")
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

		for {
			select {
			case <-subCtx.Done():
				return
			case <-ticker.C:
				// Generate clean ambient background noise (~ -55 dBFS)
				chunk := make([]float32, s.chunkSamples)
				for i := range chunk {
					chunk[i] = float32(0.001 * (float64(i%7) - 3.0))
				}
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

func (s *HostSimulatedSource) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.running = false
	return nil
}
