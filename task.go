package eletrocromo

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/lewtec/lewkit/x/driver/webview"
)

// errInvalidURLScheme is returned when a launch URL is not http(s).
var errInvalidURLScheme = errors.New("invalid URL scheme")

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

// NewBrowserLaunchTask opens urlStr in a system web view using the appID profile
// (same reverse-domain identity as App.ID / ProfileDir).
func NewBrowserLaunchTask(urlStr, appID string) Task {
	return FunctionTask(func(ctx context.Context) error {
		// Task requires respecting cancellation; do not open a window
		// after App.Run has already begun shutdown.
		if err := ctx.Err(); err != nil {
			return err
		}
		return launchBrowserURL(ctx, urlStr, appID)
	})
}

func launchBrowserURL(ctx context.Context, urlStr, appID string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("parse app url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: %s", errInvalidURLScheme, u.Scheme)
	}
	profileDir, err := ProfileDir(appID)
	if err != nil {
		return err
	}
	view, err := openDesktopView(ctx, webview.Config{
		Profile: profileDir,
		Handler: urlHandler(u),
	})
	if err != nil {
		return err
	}
	go func() {
		select {
		case <-ctx.Done():
			if err := view.Close(); err != nil {
				log.Printf("close web view: %v", err)
			}
		case <-view.Done():
		}
	}()
	return nil
}
