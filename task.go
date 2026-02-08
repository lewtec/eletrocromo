package eletrocromo

import (
	"context"
	"net/url"
	"time"
)

// Task represents a unit of work that can be executed in the background.
// Implementations must respect the provided context for cancellation and timeout.
//
// Tasks are designed to be run via App.BackgroundRun, which manages their
// lifecycle using a sync.WaitGroup.
type Task interface {
	// Run executes the task logic. It should return immediately if the task
	// is asynchronous, or block until completion/cancellation if it is synchronous.
	// It must respect context cancellation.
	Run(context.Context) error
}

// FunctionTask is an adapter that allows the use of ordinary functions as Tasks.
// Ideally used for simple, stateless tasks.
type FunctionTask func(context.Context) error

// Run executes the underlying function, passing the context to it.
func (f FunctionTask) Run(ctx context.Context) error {
	return f(ctx)
}

// NewKeepAliveTask creates a task that blocks for a specified duration or until
// the context is cancelled.
//
// This is primarily used to keep the application process alive for a minimum amount
// of time (e.g., to allow the browser to launch and connect) even if no other
// blocking tasks are running.
func NewKeepAliveTask(d time.Duration) Task {
	return FunctionTask(func(ctx context.Context) error {
		select {
		case <-time.After(d):
		case <-ctx.Done():
		}
		return nil
	})
}

// NewBrowserLaunchTask creates a task that launches a Chromium-based browser
// pointing to the specified URL.
//
// It parses the URL string and delegates the actual launching to LaunchChromium.
// Note: The task returns immediately after spawning the browser process; it does
// not wait for the browser window to close.
func NewBrowserLaunchTask(urlStr string) Task {
	return FunctionTask(func(ctx context.Context) error {
		u, err := url.Parse(urlStr)
		if err != nil {
			return err
		}
		return LaunchChromium(u)
	})
}
