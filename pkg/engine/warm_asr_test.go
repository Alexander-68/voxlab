package engine

import (
	"bytes"
	"os"
	"path/filepath"
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

func TestWarmASRTranscription(t *testing.T) {
	tok, enc, dec, joi, modelsExist := findZipformerModelDir("")
	_, binExists := findSherpaBin("sherpa-onnx-online-websocket-server")

	if !modelsExist || !binExists {
		t.Skip("Sherpa-ONNX streaming Zipformer models or websocket server binary not found, skipping live warm ASR test")
	}

	testWavCandidates := []string{
		filepath.Join("models", "sherpa-onnx-streaming-zipformer-en-2023-06-26", "test_wavs", "0.wav"),
		filepath.Join("..", "..", "models", "sherpa-onnx-streaming-zipformer-en-2023-06-26", "test_wavs", "0.wav"),
	}
	var wavData []byte
	var err error
	for _, cand := range testWavCandidates {
		if data, readErr := os.ReadFile(cand); readErr == nil {
			wavData = data
			break
		}
	}
	if len(wavData) == 0 {
		t.Skip("test wav 0.wav not available, skipping")
	}

	samples, sampleRate, err := audio.DecodeWAV(bytes.NewReader(wavData))
	if err != nil {
		t.Fatalf("failed decoding test wav: %v", err)
	}
	if sampleRate != 16000 {
		t.Fatalf("expected 16kHz test wav, got %d", sampleRate)
	}

	server, err := NewWarmASR(tok, enc, dec, joi)
	if err != nil {
		t.Fatalf("NewWarmASR failed: %v", err)
	}
	defer server.Close()

	if !server.IsWarm() {
		t.Fatalf("expected server to be warm and ready")
	}

	res, err := server.Transcribe(samples)
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}
	if res == nil || res.Transcript == "" {
		t.Fatalf("expected non-empty transcript from warm ASR, got empty")
	}

	t.Logf("Warm ASR Recognized Text: %q", res.Transcript)

	// Measure warm subsequent inference latency (zero cold starts)
	start2 := time.Now()
	res2, err := server.Transcribe(samples)
	if err != nil {
		t.Fatalf("2nd Transcribe failed: %v", err)
	}
	dur2 := time.Since(start2)
	t.Logf("2nd Transcribe took %v (text: %q)", dur2, res2.Transcript)
	if dur2 > 1500*time.Millisecond {
		t.Errorf("expected warm inference to be under 1.5s, took %v", dur2)
	}
}
