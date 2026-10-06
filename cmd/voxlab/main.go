package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"voxlab/pkg/config"
	"voxlab/pkg/server"
)

func main() {
	portFlag := flag.Int("port", 8080, "HTTP and WebSocket server port")
	hostFlag := flag.String("host", "0.0.0.0", "Host address to bind")
	configFlag := flag.String("config", "config.json", "Path to config file")
	engineFlag := flag.String("engine", "simulator", "Speech engine: 'simulator' or 'sherpa'")
	modelsFlag := flag.String("models", "models", "Directory containing Sherpa-ONNX model files")
	uiFlag := flag.String("ui", "web", "Directory containing static web UI assets")
	flag.Parse()

	log.Printf("=====================================================")
	log.Printf(" VoxLab - The Voice Pipeline Test Bench for Web Runtimes")
	log.Printf("=====================================================")

	cfg, err := config.LoadConfig(*configFlag)
	if err != nil {
		log.Printf("[Notice] Could not load %s, using defaults: %v", *configFlag, err)
		cfg = config.DefaultConfig()
	}

	// CLI flags override config file
	if *portFlag != 8080 || cfg.Port == 0 {
		cfg.Port = *portFlag
	}
	if *hostFlag != "0.0.0.0" || cfg.Host == "" {
		cfg.Host = *hostFlag
	}
	if *engineFlag != "simulator" {
		cfg.Engine.Mode = *engineFlag
	}
	if *modelsFlag != "models" {
		cfg.Engine.ModelDir = *modelsFlag
	}
	if *uiFlag != "web" {
		cfg.StaticDir = *uiFlag
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		log.Fatalf("[Fatal] Failed to initialize VoxLab server: %v", err)
	}

	// Trap termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("[Server] Stopped: %v", err)
		}
	}()

	fmt.Printf("\n VoxLab is active!\n Web UI:   http://localhost:%d\n Engine:   %s\n Press Ctrl+C to stop.\n\n", cfg.Port, cfg.Engine.Mode)

	<-sigChan
	log.Println("[VoxLab] Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Stop(ctx); err != nil {
		log.Printf("[Error] Shutdown error: %v", err)
	}
	log.Println("[VoxLab] Exited cleanly.")
}
