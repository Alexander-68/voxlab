package audio

import (
	"context"
	"testing"
	"time"
)

func TestHostNativeCapture(t *testing.T) {
	source := NewHostNativeSource(16000, 480)
	ch := make(chan []float32, 100)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := source.Start(ctx, ch)
	if err != nil {
		t.Fatalf("failed to start source: %v", err)
	}
	defer source.Stop()

	t.Logf("Detected device: %s", source.DeviceName())

	var chunkCount int
	var nonZeroCount int
	timer := time.NewTimer(1500 * time.Millisecond)
	defer timer.Stop()

loop:
	for {
		select {
		case chunk, ok := <-ch:
			if !ok {
				break loop
			}
			chunkCount++
			rms, _ := CalculateRMS(chunk)
			if rms > 1e-4 {
				nonZeroCount++
			}
		case <-timer.C:
			break loop
		}
	}

	t.Logf("Received %d chunks (non-zero: %d) in 1.5s", chunkCount, nonZeroCount)
	if chunkCount < 20 {
		t.Errorf("expected at least 20 chunks in 1.5s, got %d", chunkCount)
	}
}
