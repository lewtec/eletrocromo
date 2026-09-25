package eletrocromo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAppID = "br.tec.lew.test"

var ErrTestBoom = errors.New("boom")

func TestFunctionTask_Run(t *testing.T) {
	task := FunctionTask(func(context.Context) error {
		return ErrTestBoom
	})
	require.ErrorIs(t, task.Run(t.Context()), ErrTestBoom)
}

func TestNewKeepAliveTask_Completes(t *testing.T) {
	task := NewKeepAliveTask(5 * time.Millisecond)
	start := time.Now()
	require.NoError(t, task.Run(t.Context()))
	assert.GreaterOrEqual(t, time.Since(start), 5*time.Millisecond)
}

func TestNewKeepAliveTask_CancelsOnContext(t *testing.T) {
	// Long duration; cancellation must return promptly without waiting for d.
	task := NewKeepAliveTask(time.Hour)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	start := time.Now()
	require.NoError(t, task.Run(ctx))
	assert.Less(t, time.Since(start), time.Second)
}

func TestNewBrowserLaunchTask_InvalidURL(t *testing.T) {
	task := NewBrowserLaunchTask("://bad", testAppID)
	require.Error(t, task.Run(t.Context()))
}

func TestNewBrowserLaunchTask_RejectsNonHTTPScheme(t *testing.T) {
	task := NewBrowserLaunchTask("file:///etc/passwd", testAppID)
	require.ErrorIs(t, task.Run(t.Context()), errInvalidURLScheme)
}

func TestNewBrowserLaunchTask_RespectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	task := NewBrowserLaunchTask("http://127.0.0.1:9/", testAppID)
	require.ErrorIs(t, task.Run(ctx), context.Canceled)
}
