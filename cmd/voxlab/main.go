package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"voxlab/pkg/config"
	"voxlab/pkg/engine"
	"voxlab/pkg/server"
)

func main() {
	portFlag := flag.Int("port", 8080, "HTTP and WebSocket server port")
	hostFlag := flag.String("host", "127.0.0.1", "Host address to bind (0.0.0.0 exposes the mic and controls to the LAN, unauthenticated)")
	configFlag := flag.String("config", "config.json", "Path to config file")
	engineFlag := flag.String("engine", "auto", "Speech engine: 'auto', 'simulator' or 'sherpa'")
	modelsFlag := flag.String("models", "models", "Directory containing Sherpa-ONNX model files")
	uiFlag := flag.String("ui", "web", "Directory containing static web UI assets")
	threadsFlag := flag.Int("threads", 4, "Number of threads for neural network inference")
	providerFlag := flag.String("provider", "cpu", "Execution provider: 'cpu', 'directml', 'cuda'")
	flag.Parse()

	log.Printf("=====================================================")
	log.Printf(" VoxLab v%s - Voice Pipeline Test Bench for Web Runtimes", config.Version)
	log.Printf("=====================================================")

	cfg, err := config.LoadConfig(*configFlag)
	if err != nil {
		log.Printf("[Notice] Could not load %s, using defaults: %v", *configFlag, err)
		cfg = config.DefaultConfig()
	}

	// CLI flags override config file
	if cfg.Port == 0 {
		cfg.Port = *portFlag
	}
	if cfg.Host == "" {
		cfg.Host = *hostFlag
	}
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "host":
			cfg.Host = *hostFlag
		case "port":
			cfg.Port = *portFlag
		case "engine":
			cfg.Engine.Mode = *engineFlag
		case "models":
			cfg.Engine.ModelDir = *modelsFlag
		case "ui":
			cfg.StaticDir = *uiFlag
		case "threads":
			cfg.Engine.NumThreads = *threadsFlag
		case "provider":
			cfg.Engine.Provider = *providerFlag
		}
	})

	switch installed := engine.ASRInstalled(cfg); {
	case cfg.Engine.Mode == "auto" && installed:
		log.Printf("[VoxLab] Detected installed Sherpa-ONNX binaries and streaming ASR model: Auto-activating 'sherpa' neural engine!")
		cfg.Engine.Mode = "sherpa"
	case cfg.Engine.Mode == "auto":
		cfg.Engine.Mode = "simulator"
	case cfg.Engine.Mode == "simulator" && installed:
		log.Printf("[Notice] Engine mode 'simulator' is set explicitly although Sherpa-ONNX is installed; use -engine auto, or set engine.mode to \"auto\" in %s, to enable it", *configFlag)
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		log.Fatalf("[Fatal] Failed to initialize VoxLab server: %v", err)
	}

	// Trap termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ln, err := srv.Listen()
	if err != nil {
		_ = srv.Stop(context.Background()) // log.Fatal skips defers, and the ASR daemons would outlive us
		log.Fatalf("[Fatal] %v", err)
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	fmt.Printf("\n VoxLab v%s is active!\n Web UI:   http://localhost:%d\n Engine:   %s\n Press Ctrl+C to stop.\n\n", config.Version, cfg.Port, cfg.Engine.Mode)

	failed := false
	select {
	case <-sigChan:
	case err := <-serveErr:
		log.Printf("[Server] Stopped: %v", err)
		failed = !errors.Is(err, http.ErrServerClosed)
	}
	log.Println("[VoxLab] Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	if err := srv.Stop(ctx); err != nil {
		log.Printf("[Error] Shutdown error: %v", err)
	}
	cancel()
	if failed {
		os.Exit(1)
	}
	log.Println("[VoxLab] Exited cleanly.")
}
