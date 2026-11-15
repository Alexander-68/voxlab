package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"voxlab/pkg/audio"
	"voxlab/pkg/config"
	"voxlab/pkg/engine"
	"voxlab/pkg/evaluation/corpus"
)

func main() {
	dir := flag.String("out", filepath.Join("data", "test_samples"), "output directory (the server reads data/test_samples relative to its working directory)")
	flag.Parse()
	if err := os.MkdirAll(*dir, 0755); err != nil {
		log.Fatal(err)
	}

	// 1. Chime tone
	b1, err := audio.EncodeWAV(audio.GenerateChime(16000, 440, 880, 2.0), 16000)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(*dir, "chime_alert.wav"), b1, 0644); err != nil {
		log.Fatal(err)
	}

	// 2. Spoken wake word + command, injected as mic input by the wav_file_injection source.
	cfg := config.DefaultConfig()
	if !engine.TTSInstalled(cfg) {
		log.Fatalf("voice_test.wav needs real speech: install a Kokoro model under %s", cfg.Engine.ModelDir)
	}
	cfg.KWS.Enabled = false
	runner := engine.NewSherpaRunner(cfg)
	defer runner.Close()
	voicePath := filepath.Join(*dir, "voice_test.wav")
	// LoadOrSynthesizeClip keeps existing clips, so the old one must go.
	if err := os.Remove(voicePath); err != nil && !os.IsNotExist(err) {
		log.Fatalf("cannot replace %s: %v", voicePath, err)
	}
	if _, err := corpus.LoadOrSynthesizeClip(voicePath, runner, engine.TTSRequest{
		Text: "Hey VoxLab. Start recording.", Voice: cfg.TTS.DefaultVoice, Speed: 1.0, SampleRate: 24000,
	}); err != nil {
		runner.Close()
		log.Fatal(err)
	}

	log.Printf("Generated test sample WAVs in %s", *dir)
}
