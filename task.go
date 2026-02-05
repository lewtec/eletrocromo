package eletrocromo

import (
	"context"
	"net/url"
	"time"
)

// Task represents a unit of work that can be executed in the background.
// Implementations must respect the provided context for cancellation and timeout.
type Task interface {
	Run(context.Context) error
}

// FunctionTask is an adapter that allows the use of ordinary functions as Tasks.
type FunctionTask func(context.Context) error

// Run executes the underlying function, passing the context to it.
func (f FunctionTask) Run(ctx context.Context) error {
	return f(ctx)
}

func NewKeepAliveTask(d time.Duration) Task {
	return FunctionTask(func(ctx context.Context) error {
		select {
		case <-time.After(d):
		case <-ctx.Done():
		}
		return nil
	})
}

func NewBrowserLaunchTask(urlStr string) Task {
	return FunctionTask(func(ctx context.Context) error {
		u, err := url.Parse(urlStr)
		if err != nil {
			return err
		}
		return LaunchChromium(u)
	})
}
