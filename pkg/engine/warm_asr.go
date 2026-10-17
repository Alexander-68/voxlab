package engine

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/gorilla/websocket"
)

// WarmASR is a resident sherpa-onnx-online-websocket-server process. Models load once;
// each listening session opens one ASRStream and feeds audio to it in real time.
type WarmASR struct {
	cmd  *exec.Cmd
	port int
}

// NewWarmASR starts the streaming ASR server and waits until it accepts connections.
// endpointSilenceSec is the trailing pause after speech that closes a segment (sherpa rule2).
func NewWarmASR(tok, enc, dec, joi string, numThreads int, endpointSilenceSec float64) (*WarmASR, error) {
	w, err := StartWarmASR(tok, enc, dec, joi, numThreads, endpointSilenceSec)
	if err != nil {
		return nil, err
	}
	if err := w.WaitReady(); err != nil {
		return nil, err
	}
	return w, nil
}

// StartWarmASR spawns the server process without waiting for the models to load,
// so the caller can Close (kill) it even mid-startup.
func StartWarmASR(tok, enc, dec, joi string, numThreads int, endpointSilenceSec float64) (*WarmASR, error) {
	serverBin, ok := findSherpaBin("sherpa-onnx-online-websocket-server")
	if !ok {
		return nil, fmt.Errorf("sherpa-onnx-online-websocket-server binary not found")
	}
	if numThreads <= 0 {
		numThreads = 4
	}
	if endpointSilenceSec <= 0 {
		endpointSilenceSec = 0.8
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
		fmt.Sprintf("--num-threads=%d", numThreads),
		fmt.Sprintf("--rule2-min-trailing-silence=%.2f", endpointSilenceSec),
		fmt.Sprintf("--port=%d", port),
	)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start sherpa-onnx websocket server: %w", err)
	}
	log.Printf("[SherpaRunner] Starting warm streaming ASR server on port %d (%d threads, endpoint pause %.2fs)",
		port, numThreads, endpointSilenceSec)
	return &WarmASR{cmd: cmd, port: port}, nil
}

// WaitReady polls until the server accepts connections; on timeout it kills the process.
func (w *WarmASR) WaitReady() error {
	if waitForPort(w.port) {
		return nil
	}
	_ = w.Close()
	return fmt.Errorf("sherpa-onnx-online-websocket-server failed to bind port %d within timeout", w.port)
}

// waitForPort polls until a local server accepts connections (up to 30s for large models to load).
func waitForPort(port int) bool {
	for i := 0; i < 300; i++ {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// findTransducerModel locates tokens.txt and encoder/decoder/joiner ONNX files in dir,
// preferring int8 variants. Works for Zipformer, NeMo/Nemotron and Parakeet exports.
func findTransducerModel(dir string) (tok, enc, dec, joi string, ok bool) {
	if dir == "" {
		return "", "", "", "", false
	}
	pick := func(part string) string {
		for _, pat := range []string{part + "*.int8.onnx", part + "*.onnx"} {
			if m, _ := filepath.Glob(filepath.Join(dir, pat)); len(m) > 0 {
				return m[0]
			}
		}
		return ""
	}
	tok = filepath.Join(dir, "tokens.txt")
	enc, dec, joi = pick("encoder"), pick("decoder"), pick("joiner")
	ok = isRegularFile(tok) && enc != "" && dec != "" && joi != ""
	return tok, enc, dec, joi, ok
}

// normalizeCase lowercases all-caps output (LibriSpeech Zipformer) and keeps the casing
// of models that produce real capitalization (Nemotron, Parakeet). Punctuation-only output
// (punctuating models emit a lone "." on trailing silence) counts as no speech.
func normalizeCase(text string) string {
	if !strings.ContainsFunc(text, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) {
		return ""
	}
	if strings.ToUpper(text) == text {
		return strings.ToLower(text)
	}
	return text
}

func getFreeTCPPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// Close kills the server process.
func (w *WarmASR) Close() error {
	if w.cmd.Process != nil {
		return w.cmd.Process.Kill()
	}
	return nil
}

// Open starts a new recognition stream.
func (w *WarmASR) Open() (*ASRStream, error) {
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	conn, _, err := dialer.Dial(fmt.Sprintf("ws://127.0.0.1:%d", w.port), nil)
	if err != nil {
		return nil, fmt.Errorf("failed connecting to warm ASR websocket: %w", err)
	}
	s := &ASRStream{conn: conn, done: make(chan struct{})}
	go s.readLoop()
	return s, nil
}

// ASRStream is one live recognition session. The server decodes audio as it arrives,
// endpoints segments itself, and pushes hypotheses back; readLoop collects them.
// Send/Finish must be called from a single goroutine; Close may be called from any.
type ASRStream struct {
	conn *websocket.Conn
	done chan struct{}

	mu             sync.Mutex
	finals         []ASRResult // endpointed segments not yet taken
	partial        string      // hypothesis of the segment in progress
	partialChanged bool
}

func (s *ASRStream) readLoop() {
	defer close(s.done)
	for {
		_, msg, err := s.conn.ReadMessage()
		if err != nil || string(msg) == "Done!" {
			return
		}
		var res struct {
			Text       string    `json:"text"`
			IsFinal    bool      `json:"is_final"`
			StartTime  float64   `json:"start_time"` // segment start, seconds since stream start
			Timestamps []float64 `json:"timestamps"` // token times relative to start_time
		}
		if json.Unmarshal(msg, &res) != nil {
			continue
		}
		text := normalizeCase(strings.TrimSpace(res.Text))
		s.mu.Lock()
		if res.IsFinal && text != "" {
			seg := ASRResult{IsFinal: true, Transcript: text, Confidence: 0.95}
			if n := len(res.Timestamps); n > 0 {
				seg.StartSec = res.StartTime + res.Timestamps[0]
				seg.EndSec = res.StartTime + res.Timestamps[n-1]
			}
			s.finals = append(s.finals, seg)
			text = ""
		}
		if text != s.partial {
			s.partial = text
			s.partialChanged = true
		}
		s.mu.Unlock()
	}
}

// Send streams float32 samples to the recognizer.
func (s *ASRStream) Send(samples []float32) error {
	buf := make([]byte, len(samples)*4)
	for i, v := range samples {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return s.conn.WriteMessage(websocket.BinaryMessage, buf)
}

// Next returns the oldest endpointed segment as a final result, otherwise the current
// partial hypothesis if it changed since the last call (empty text = hypothesis dropped), otherwise nil.
func (s *ASRStream) Next() *ASRResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.finals) > 0 {
		seg := s.finals[0]
		s.finals = s.finals[1:]
		return &seg
	}
	if s.partialChanged {
		s.partialChanged = false
		return &ASRResult{Transcript: s.partial}
	}
	return nil
}

// Finish tells the server no more audio is coming, waits for it to decode the tail,
// closes the stream and returns all segments not yet taken (including the in-progress one).
func (s *ASRStream) Finish(timeout time.Duration) []string {
	if s.conn.WriteMessage(websocket.TextMessage, []byte("Done")) == nil {
		select {
		case <-s.done:
		case <-time.After(timeout):
			log.Printf("[SherpaRunner] Timed out waiting for final ASR result")
		}
	}
	s.Close()
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, seg := range s.finals {
		out = append(out, seg.Transcript)
	}
	if s.partial != "" {
		out = append(out, s.partial)
	}
	s.finals, s.partial = nil, ""
	return out
}

// Close drops the stream without waiting for results.
func (s *ASRStream) Close() {
	_ = s.conn.Close()
}
