package eletrocromo

import (
	"testing"
	"time"
)

func TestAppRun_NoPanicOnNilContext(t *testing.T) {
	// Create an app with no Context (nil)
	app := &App{
		Handler: nil, // Handler can be nil, ServeHTTP handles it
	}

	// Capture panic if it happens
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("App.Run panicked: %v", r)
		}
	}()

	// Run needs to finish. It has a hardcoded 5 second sleep.
	// We accept this for the test to verify the fix for the nil context panic.
	// In a real refactor, we would make the duration configurable.

	// We run it in a separate goroutine to avoid blocking indefinitely if it hangs,
	// though here we expect it to take ~5 seconds.
	done := make(chan error)
	go func() {
		done <- app.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("App.Run returned error: %v", err)
		}
	case <-time.After(6 * time.Second):
		t.Error("App.Run timed out (longer than expected 5s)")
	}
}
