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
		// NewTimer + Stop avoids the time.After leak when ctx cancels before d.
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
		}
		return nil
	})
}

// NewBrowserLaunchTask launches Helium for urlStr using the appID profile
// (same reverse-domain identity as App.ID / ProfileDir).
func NewBrowserLaunchTask(urlStr, appID string) Task {
	return FunctionTask(func(ctx context.Context) error {
		u, err := url.Parse(urlStr)
		if err != nil {
			return err
		}
		return LaunchChromium(ctx, u, appID)
	})
}
