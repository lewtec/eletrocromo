package eletrocromo

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCommandOutput_PreservesContextCanceled ensures cancel is detectable
// through the returned error chain. CommandContext may report the child as
// "signal: killed"; we join ctx.Err() so errors.Is(..., context.Canceled) works
// for ResolveBrowserHost / App.Run shutdown.
func TestCommandOutput_PreservesContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		_, err := commandOutput(ctx, "sleep", "30")
		done <- err
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error after cancel")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want errors.Is(..., context.Canceled), got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("commandOutput did not return after cancel")
	}
}

func TestCommandOutput_AlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := commandOutput(ctx, "true")
	if err == nil {
		t.Fatal("expected error when context already canceled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestCommandOutput_PreservesExitError(t *testing.T) {
	// false exits 1 with empty stderr on most systems.
	_, err := commandOutput(t.Context(), "false")
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("want *exec.ExitError in chain, got %v", err)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatalf("active context must not look canceled: %v", err)
	}
}

func TestCommandOutput_IncludesStderrAndWraps(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fail.sh")
	body := "#!/bin/sh\necho boom-on-stderr >&2\nexit 2\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := commandOutput(t.Context(), script)
	if err == nil {
		t.Fatal("expected error")
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("want ExitError wrap, got %v", err)
	}
	if !strings.Contains(err.Error(), "boom-on-stderr") {
		t.Fatalf("stderr missing from error: %q", err.Error())
	}
}
