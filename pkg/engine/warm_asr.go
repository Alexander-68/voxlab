package engine

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WarmASR defines the interface for an in-memory speech recognizer daemon.
type WarmASR interface {
	Transcribe(samples []float32) (*ASRResult, error)
	IsWarm() bool
	Close() error
}

type warmASRServer struct {
	cmd    *exec.Cmd
	port   int
	mu     sync.Mutex
	ready  bool
	closed bool
}

// NewWarmASR creates and initializes a persistent, warm Sherpa-ONNX streaming ASR server.
func NewWarmASR(tok, enc, dec, joi string) (WarmASR, error) {
	serverBin, ok := findSherpaBin("sherpa-onnx-online-websocket-server")
	if !ok {
		return nil, fmt.Errorf("sherpa-onnx-online-websocket-server binary not found")
	}

	port, err := getFreeTCPPort()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate free port: %w", err)
	}

	cmd := exec.Command(serverBin,
		"--tokens="+tok,
		"--encoder="+enc,
		"--decoder="+dec,
		"--joiner="+joi,
		"--num-threads=2",
		fmt.Sprintf("--port=%d", port),
	)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start sherpa-onnx websocket server: %w", err)
	}

	server := &warmASRServer{
		cmd:  cmd,
		port: port,
	}

	// Poll port until ready (up to 6 seconds for ONNX models to load into RAM once)
	ready := false
	for i := 0; i < 60; i++ {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if !ready {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("sherpa-onnx-online-websocket-server failed to bind port %d within timeout", port)
	}

	server.ready = true
	log.Printf("[SherpaRunner] Warm streaming ASR server initialized on port %d (zero cold starts)", port)
	return server, nil
}

func getFreeTCPPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func (s *warmASRServer) IsWarm() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ready && !s.closed
}

func (s *warmASRServer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.ready = false
	if s.cmd != nil && s.cmd.Process != nil {
		return s.cmd.Process.Kill()
	}
	return nil
}

func (s *warmASRServer) Transcribe(samples []float32) (*ASRResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.ready || s.closed {
		return nil, fmt.Errorf("warm ASR server is not running")
	}

	wsURL := fmt.Sprintf("ws://127.0.0.1:%d", s.port)
	dialer := websocket.Dialer{
		HandshakeTimeout: 2 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed connecting to warm ASR websocket: %w", err)
	}
	defer conn.Close()

	// Convert float32 samples to IEEE 754 float32 little endian bytes
	buf := make([]byte, len(samples)*4)
	for i, v := range samples {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}

	// Stream audio chunks in 8000-sample messages (0.5s chunks)
	chunkBytes := 8000 * 4
	for offset := 0; offset < len(buf); offset += chunkBytes {
		end := offset + chunkBytes
		if end > len(buf) {
			end = len(buf)
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, buf[offset:end]); err != nil {
			return nil, fmt.Errorf("failed writing audio chunk: %w", err)
		}
	}

	// Tell server audio stream is finished
	if err := conn.WriteMessage(websocket.TextMessage, []byte("Done")); err != nil {
		return nil, fmt.Errorf("failed writing Done message: %w", err)
	}

	var finalTranscript string
	var finalTokens []string

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if msgType == websocket.TextMessage {
			text := string(msg)
			if text == "Done!" {
				break
			}
			var res struct {
				Text    string   `json:"text"`
				Tokens  []string `json:"tokens"`
				IsFinal bool     `json:"is_final"`
			}
			if err := json.Unmarshal(msg, &res); err == nil && res.Text != "" {
				finalTranscript = res.Text
				finalTokens = res.Tokens
			}
		}
	}

	if finalTranscript == "" {
		return &ASRResult{
			Transcript: "",
			Tokens:     nil,
			IsFinal:    true,
		}, nil
	}

	return &ASRResult{
		Transcript: finalTranscript,
		Tokens:     finalTokens,
		IsFinal:    true,
	}, nil
}
