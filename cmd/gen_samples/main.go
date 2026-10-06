package main

import (
	"log"
	"os"
	"path/filepath"

	"voxlab/pkg/audio"
)

func main() {
	dir := "data/test_samples"
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatal(err)
	}

	// 1. Chime tone
	b1, err := audio.EncodeWAV(audio.GenerateChime(16000, 440, 880, 2.0), 16000)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "chime_alert.wav"), b1, 0644); err != nil {
		log.Fatal(err)
	}

	// 2. Simulated voice harmonic
	b2, err := audio.EncodeWAV(audio.GenerateChime(16000, 220, 440, 3.5), 16000)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "voice_test.wav"), b2, 0644); err != nil {
		log.Fatal(err)
	}

	log.Println("Generated test sample WAVs in data/test_samples/")
}
