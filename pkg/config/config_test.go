package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigRejectsBadInput(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"unknown key": `{"audio":{"sampleRate":48000}}`,
		"zero rate":   `{"audio":{"sample_rate":0}}`,
		"bad port":    `{"port":70000}`,
		"bad thresh":  `{"intent":{"confidence_threshold":1.5}}`,
	} {
		p := filepath.Join(dir, "c.json")
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadConfig(p); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}

	p := filepath.Join(dir, "ok.json")
	_ = os.WriteFile(p, []byte(`{"port":0,"audio":{"sample_rate":48000}}`), 0o644)
	if cfg, err := LoadConfig(p); err != nil || cfg.Audio.SampleRate != 48000 {
		t.Fatalf("valid config rejected: %v", err)
	}
}
