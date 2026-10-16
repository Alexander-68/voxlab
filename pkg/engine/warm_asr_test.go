package engine

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"voxlab/pkg/audio"
)

func TestGetFreeTCPPort(t *testing.T) {
	port, err := getFreeTCPPort()
	if err != nil {
		t.Fatalf("getFreeTCPPort failed: %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Fatalf("invalid port returned: %d", port)
	}
}

// TestWarmASRStreaming feeds speech + pause + speech through one stream and expects
// the server to endpoint the first utterance on its own and Finish to flush the second.
func TestWarmASRStreaming(t *testing.T) {
	tok, enc, dec, joi, modelsExist := findZipformerModelDir("")
	_, binExists := findSherpaBin("sherpa-onnx-online-websocket-server")
	if !modelsExist || !binExists {
		t.Skip("Sherpa-ONNX streaming Zipformer models or websocket server binary not found")
	}
	wavData, err := os.ReadFile(filepath.Join("..", "..", "models", "sherpa-onnx-streaming-zipformer-en-2023-06-26", "test_wavs", "0.wav"))
	if err != nil {
		t.Skip("test wav 0.wav not available")
	}
	samples, _, err := audio.DecodeWAV(bytes.NewReader(wavData))
	if err != nil {
		t.Fatalf("failed decoding test wav: %v", err)
	}

	warm, err := NewWarmASR(tok, enc, dec, joi, 4, 0.8)
	if err != nil {
		t.Fatalf("NewWarmASR failed: %v", err)
	}
	defer warm.Close()

	st, err := warm.Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	audioIn := append(append(append([]float32{}, samples...), make([]float32, 16000*2)...), samples...)
	var finals []string
	sawPartial := false
	for i := 0; i < len(audioIn); i += 480 {
		if err := st.Send(audioIn[i:min(i+480, len(audioIn))]); err != nil {
			t.Fatalf("Send failed: %v", err)
		}
		time.Sleep(2 * time.Millisecond) // faster than real time, slow enough to read results
		for res := st.Next(); res != nil; res = st.Next() {
			if res.IsFinal {
				finals = append(finals, res.Transcript)
			} else if res.Transcript != "" {
				sawPartial = true
			}
		}
	}
	tail := st.Finish(5 * time.Second)
	finals = append(finals, tail...)
	t.Logf("partials=%v segments=%q", sawPartial, finals)

	if !sawPartial {
		t.Errorf("expected partial hypotheses while streaming")
	}
	if len(finals) != 2 {
		t.Fatalf("expected 2 segments (server endpoint + flush), got %d: %q", len(finals), finals)
	}
	for _, f := range finals {
		if !strings.Contains(f, "yellow lamps") {
			t.Errorf("unexpected transcript %q", f)
		}
	}
}
