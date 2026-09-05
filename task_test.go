package eletrocromo

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFunctionTask_Run(t *testing.T) {
	want := errors.New("boom")
	task := FunctionTask(func(ctx context.Context) error {
		if ctx == nil {
			t.Fatal("nil context")
		}
		return want
	})
	if err := task.Run(t.Context()); !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func TestNewKeepAliveTask_Completes(t *testing.T) {
	task := NewKeepAliveTask(5 * time.Millisecond)
	start := time.Now()
	if err := task.Run(t.Context()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 5*time.Millisecond {
		t.Fatalf("returned too early: %v", elapsed)
	}
}

func TestNewKeepAliveTask_CancelsOnContext(t *testing.T) {
	// Long duration; cancellation must return promptly without waiting for d.
	task := NewKeepAliveTask(time.Hour)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	start := time.Now()
	if err := task.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("did not observe cancel promptly: %v", elapsed)
	}
}

func TestNewBrowserLaunchTask_InvalidURL(t *testing.T) {
	task := NewBrowserLaunchTask("://bad", testAppID)
	err := task.Run(t.Context())
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestNewBrowserLaunchTask_RejectsNonHTTPScheme(t *testing.T) {
	task := NewBrowserLaunchTask("file:///etc/passwd", testAppID)
	err := task.Run(t.Context())
	if err == nil {
		t.Fatal("expected scheme error")
	}
}

// Cancelled context must short-circuit before LaunchChromium (no browser spawn).
func TestNewBrowserLaunchTask_RespectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	task := NewBrowserLaunchTask("http://127.0.0.1:9/", testAppID)
	err := task.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}
