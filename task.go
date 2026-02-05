package eletrocromo

import (
	"context"
	"net/url"
	"time"
)

type Task interface {
	Run(context.Context) error
}

type FunctionTask func(context.Context) error

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
