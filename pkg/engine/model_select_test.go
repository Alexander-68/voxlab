package engine

import (
	"os"
	"path/filepath"
	"testing"

	"voxlab/pkg/config"
)

func installKokoro(t *testing.T, dir, file string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, file), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestSetTTSModelOnlyAcceptsInstalledModels(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	installKokoro(t, filepath.Join(outside, "kokoro-evil"), "model.onnx")

	cfg := config.DefaultConfig()
	cfg.Engine.ModelDir = root
	before := cfg.Engine.KokoroModelDir
	r := &SherpaRunner{cfg: cfg}

	for _, name := range []string{
		filepath.Join(outside, "kokoro-evil"),
		filepath.Join("..", filepath.Base(outside), "kokoro-evil"),
		`\\host\share\kokoro-evil`,
		"not-installed",
	} {
		if err := r.SetTTSModel(name); err == nil {
			t.Errorf("accepted %q", name)
		}
	}
	if cfg.Engine.KokoroModelDir != before {
		t.Errorf("rejected model changed the config to %q", cfg.Engine.KokoroModelDir)
	}
}

// Download scripts default to int8/v1_1 while the config names v1_0.
func TestUnavailableConfiguredModelFallsBackToDiscovery(t *testing.T) {
	t.Chdir(t.TempDir())
	want := filepath.Join("models", "kokoro-int8-multi-lang-v1_1")
	installKokoro(t, want, "model.int8.onnx")

	if dir, ok := findKokoroModelDir("models/kokoro-multi-lang-v1_0"); !ok || dir != want {
		t.Fatalf("got %q, %v, want %q", dir, ok, want)
	}
}

func TestMissingModelIsNotReportedAsNeural(t *testing.T) {
	t.Chdir(t.TempDir())
	cfg := config.DefaultConfig()
	r := &SherpaRunner{cfg: cfg}
	if name, neural := r.TTSModelInfo(); neural {
		t.Fatalf("reported %q as neural with no model installed", name)
	}
}
