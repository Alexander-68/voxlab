package engine

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const daemonTailBytes = 2048

// tailWriter keeps the last daemonTailBytes written to it.
type tailWriter struct {
	mu  sync.Mutex
	buf []byte
}

func (t *tailWriter) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if len(t.buf) > daemonTailBytes {
		t.buf = append(t.buf[:0], t.buf[len(t.buf)-daemonTailBytes:]...)
	}
	return len(p), nil
}

func (t *tailWriter) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return strings.TrimSpace(string(t.buf))
}

// daemon is a resident helper process. It reaps the process, and if the process dies on its own
// it logs the end of its stderr, which is the only place sherpa-onnx explains a failure.
type daemon struct {
	cmd     *exec.Cmd
	closing atomic.Bool
	done    chan struct{} // closed once the process is reaped
}

func startDaemon(name string, cmd *exec.Cmd) (*daemon, error) {
	stderr := &tailWriter{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start %s: %w", name, err)
	}
	if err := killWithParent(cmd); err != nil {
		log.Printf("[SherpaRunner] %s may outlive a crash of this process: %v", name, err)
	}
	d := &daemon{cmd: cmd, done: make(chan struct{})}
	go func() {
		defer close(d.done)
		err := cmd.Wait()
		if !d.closing.Load() {
			log.Printf("[SherpaRunner] %s exited unexpectedly (%v): %s", name, err, stderr.String())
		}
	}()
	return d, nil
}

// Close kills the process and waits (bounded) until it is reaped.
func (d *daemon) Close() error {
	d.closing.Store(true)
	err := d.cmd.Process.Kill()
	if errors.Is(err, os.ErrProcessDone) {
		err = nil
	}
	select {
	case <-d.done:
	case <-time.After(5 * time.Second):
	}
	return err
}
