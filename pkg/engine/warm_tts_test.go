package engine

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestWarmTTSSentenceSplitting(t *testing.T) {
	dllDir := filepath.Join("..", "..", "bin")
	modelDir := filepath.Join("..", "..", "models", "kokoro-multi-lang-v1_0")
	modelFile := filepath.Join(modelDir, "model.onnx")
	voicesPath := filepath.Join(modelDir, "voices.bin")
	tokensPath := filepath.Join(modelDir, "tokens.txt")
	dataDirPath := filepath.Join(modelDir, "espeak-ng-data")
	lexPath := filepath.Join(modelDir, "lexicon-us-en.txt")

	warm, err := newPlatformWarmTTS(dllDir, modelFile, voicesPath, tokensPath, dataDirPath, lexPath, "", 4, "cpu")
	if err != nil {
		t.Skipf("Warm TTS not available: %v", err)
		return
	}
	defer warm.Close()

	testCases := []string{
		"Settings updated successfully, voice pipeline operational.",
		"Settings updated successfully; voice pipeline operational.",
		"Settings updated successfully. Voice pipeline operational.",
	}

	for _, text := range testCases {
		optimized := optimizeStreamingClauses(text)
		t.Logf("Testing text: %q (optimized: %q)", text, optimized)
		var chunks []int64
		t0 := time.Now()
		res, err := warm.SynthesizeStream(optimized, 3, 1.0, func(chunk TTSChunk) error {
			lat := time.Since(t0).Milliseconds()
			chunks = append(chunks, lat)
			fmt.Printf("   [Chunk %d] at +%dms (Dur: %.2fs)\n", chunk.Index, lat, chunk.DurationSec)
			return nil
		})
		if err != nil {
			t.Fatalf("SynthesizeStream failed: %v", err)
		}
		total := time.Since(t0).Milliseconds()
		fmt.Printf("   -> Total: %dms (audio: %.2fs, total chunks: %d)\n\n", total, res.DurationSec, len(chunks))
	}
}

func TestOptimizeStreamingClauses(t *testing.T) {
	in1 := "Settings updated successfully, voice pipeline operational."
	out1 := optimizeStreamingClauses(in1)
	if out1 != "Settings updated successfully. Voice pipeline operational." {
		t.Errorf("expected clause split on comma, got %q", out1)
	}

	in2 := "Light 1, 2, 3"
	out2 := optimizeStreamingClauses(in2)
	if out2 != "Light 1, 2, 3" {
		t.Errorf("expected no split for short list, got %q", out2)
	}

	in3 := "Command acknowledged; starting background execution."
	out3 := optimizeStreamingClauses(in3)
	if out3 != "Command acknowledged. Starting background execution." {
		t.Errorf("expected semicolon split, got %q", out3)
	}
}
