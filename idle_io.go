package eletrocromo

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// downloadIdleTimeout aborts bootstrap transfers that stall with no bytes.
// Total transfer time is unlimited so large Helium/workspaced downloads can finish.
var downloadIdleTimeout = 2 * time.Minute

// idleTimeoutReader closes the underlying ReadCloser if no Read completes
// within idle. Closing unblocks a stuck Read on HTTP response bodies.
type idleTimeoutReader struct {
	r    io.ReadCloser
	idle time.Duration

	mu sync.Mutex
	t  *time.Timer
}

func newIdleTimeoutReader(r io.ReadCloser, idle time.Duration) *idleTimeoutReader {
	if idle <= 0 {
		idle = downloadIdleTimeout
	}
	ir := &idleTimeoutReader{r: r, idle: idle}
	ir.arm()
	return ir
}

func (r *idleTimeoutReader) arm() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.t == nil {
		r.t = time.AfterFunc(r.idle, func() { _ = r.r.Close() })
		return
	}
	r.t.Reset(r.idle)
}

func (r *idleTimeoutReader) stopTimer() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.t != nil {
		r.t.Stop()
	}
}

func (r *idleTimeoutReader) Read(p []byte) (int, error) {
	r.arm()
	n, err := r.r.Read(p)
	if n > 0 {
		r.arm()
	}
	if err != nil {
		r.stopTimer()
		// Map close-from-idle to a clearer error when possible.
		if err != io.EOF {
			return n, fmt.Errorf("download idle timeout or read error after %v: %w", r.idle, err)
		}
	}
	return n, err
}

func (r *idleTimeoutReader) Close() error {
	r.stopTimer()
	return r.r.Close()
}
