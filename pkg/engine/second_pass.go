package engine

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// WarmOffline is a resident sherpa-onnx-offline-websocket-server holding a non-streaming
// model (Parakeet TDT). It re-transcribes endpointed segments for a punctuated, more
// accurate final text while the streaming model keeps providing live partials.
type WarmOffline struct {
	cmd  *exec.Cmd
	port int
}

// StartWarmOffline spawns the offline server without waiting for the model to load.
func StartWarmOffline(tok, enc, dec, joi string, numThreads int) (*WarmOffline, error) {
	serverBin, ok := findSherpaBin("sherpa-onnx-offline-websocket-server")
	if !ok {
		return nil, fmt.Errorf("sherpa-onnx-offline-websocket-server binary not found")
	}
	if numThreads <= 0 {
		numThreads = 4
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
		fmt.Sprintf("--port=%d", port),
	)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start sherpa-onnx offline websocket server: %w", err)
	}
	log.Printf("[SherpaRunner] Starting warm second-pass ASR server on port %d (%d threads, %s)", port, numThreads, enc)
	return &WarmOffline{cmd: cmd, port: port}, nil
}

// WaitReady polls until the server accepts connections; on timeout it kills the process.
func (w *WarmOffline) WaitReady() error {
	if waitForPort(w.port) {
		return nil
	}
	_ = w.Close()
	return fmt.Errorf("sherpa-onnx-offline-websocket-server failed to bind port %d within timeout", w.port)
}

// Close kills the server process.
func (w *WarmOffline) Close() error {
	if w.cmd.Process != nil {
		return w.cmd.Process.Kill()
	}
	return nil
}

// Transcribe decodes one complete utterance. Wire format of the sherpa offline server:
// int32 sample rate, int32 payload byte count, float32 samples; reply is one JSON (or text) message.
func (w *WarmOffline) Transcribe(samples []float32, sampleRate int) (string, error) {
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	conn, _, err := dialer.Dial(fmt.Sprintf("ws://127.0.0.1:%d", w.port), nil)
	if err != nil {
		return "", fmt.Errorf("failed connecting to second-pass websocket: %w", err)
	}
	defer conn.Close()

	buf := make([]byte, 8+len(samples)*4)
	binary.LittleEndian.PutUint32(buf[0:], uint32(sampleRate))
	binary.LittleEndian.PutUint32(buf[4:], uint32(len(samples)*4))
	for i, v := range samples {
		binary.LittleEndian.PutUint32(buf[8+i*4:], math.Float32bits(v))
	}
	const payload = 10240 // same framing as the reference client
	for len(buf) > 0 {
		n := min(payload, len(buf))
		if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
			return "", err
		}
		buf = buf[n:]
	}

	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return "", err
	}
	_ = conn.WriteMessage(websocket.TextMessage, []byte("Done"))

	var res struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(msg, &res) != nil {
		res.Text = string(msg)
	}
	return normalizeCase(strings.TrimSpace(res.Text)), nil
}
