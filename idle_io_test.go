package eletrocromo

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"
)

type stallReader struct {
	// first Read returns data; subsequent block until closed
	once  bool
	block chan struct{}
}

func (s *stallReader) Read(p []byte) (int, error) {
	if !s.once {
		s.once = true
		return copy(p, []byte("hi")), nil
	}
	<-s.block
	return 0, io.ErrClosedPipe
}

func (s *stallReader) Close() error {
	select {
	case <-s.block:
	default:
		close(s.block)
	}
	return nil
}

func TestIdleTimeoutReader_ProgressResets(t *testing.T) {
	// Streaming reader that always returns quickly.
	r := newIdleTimeoutReader(io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("x"), 100))), time.Second)
	buf, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(buf) != 100 {
		t.Fatalf("len=%d", len(buf))
	}
}

func TestIdleTimeoutReader_Stalls(t *testing.T) {
	s := &stallReader{block: make(chan struct{})}
	r := newIdleTimeoutReader(s, 50*time.Millisecond)
	// First read succeeds.
	buf := make([]byte, 8)
	n, err := r.Read(buf)
	if err != nil || n != 2 {
		t.Fatalf("first read n=%d err=%v", n, err)
	}
	// Second read blocks until idle close.
	_, err = r.Read(buf)
	if err == nil {
		t.Fatal("expected idle error")
	}
	if errors.Is(err, io.EOF) {
		t.Fatalf("unexpected EOF: %v", err)
	}
}
