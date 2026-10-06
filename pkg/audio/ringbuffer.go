package audio

import (
	"sync"
)

// RingBuffer is a thread-safe circular buffer for float32 audio samples.
type RingBuffer struct {
	mu       sync.RWMutex
	buf      []float32
	capacity int
	writePos int
	isFull   bool
}

// NewRingBuffer creates a new RingBuffer with the specified sample capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 16000 // default 1 second at 16kHz
	}
	return &RingBuffer{
		buf:      make([]float32, capacity),
		capacity: capacity,
	}
}

// Write appends samples to the ring buffer, overwriting oldest samples when full.
func (r *RingBuffer) Write(samples []float32) {
	if len(samples) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, s := range samples {
		r.buf[r.writePos] = s
		r.writePos++
		if r.writePos >= r.capacity {
			r.writePos = 0
			r.isFull = true
		}
	}
}

// ReadSnapshot returns a contiguous snapshot of all samples currently in the buffer,
// ordered chronologically from oldest to newest.
func (r *RingBuffer) ReadSnapshot() []float32 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.isFull && r.writePos == 0 {
		return nil
	}

	if !r.isFull {
		out := make([]float32, r.writePos)
		copy(out, r.buf[:r.writePos])
		return out
	}

	out := make([]float32, r.capacity)
	// If full, oldest sample is at writePos
	copy(out, r.buf[r.writePos:])
	copy(out[r.capacity-r.writePos:], r.buf[:r.writePos])
	return out
}

// Size returns the count of samples currently buffered.
func (r *RingBuffer) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.isFull {
		return r.capacity
	}
	return r.writePos
}

// Clear empties the ring buffer.
func (r *RingBuffer) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writePos = 0
	r.isFull = false
}
