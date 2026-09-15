package eletrocromo

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestBackgroundRun_TracksWaitGroup ensures WaitGroup.Add runs before the
// task goroutine is scheduled, so Wait cannot return early (the race that
// previously needed time.Sleep in App.Run).
func TestBackgroundRun_TracksWaitGroup(t *testing.T) {
	app := &App{Context: t.Context()}
	started := make(chan struct{})
	release := make(chan struct{})

	err := app.BackgroundRun(FunctionTask(func(ctx context.Context) error {
		close(started)
		<-release
		return nil
	}))
	if err != nil {
		t.Fatalf("BackgroundRun: %v", err)
	}

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("task did not start")
	}

	done := make(chan struct{})
	go func() {
		app.WaitGroup.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("WaitGroup.Wait returned before task finished")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("WaitGroup.Wait did not return after task finished")
	}
}

func TestBackgroundRun_UsesAppContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	app := &App{Context: ctx}

	var sawCancel atomic.Bool
	_ = app.BackgroundRun(FunctionTask(func(taskCtx context.Context) error {
		<-taskCtx.Done()
		sawCancel.Store(true)
		return nil
	}))

	cancel()
	app.WaitGroup.Wait()
	if !sawCancel.Load() {
		t.Fatal("task did not observe context cancellation")
	}
}

// Nil Context must not panic inside task.Run; same default as App.Run.
func TestBackgroundRun_NilContext_UsesBackground(t *testing.T) {
	app := &App{} // Context intentionally unset
	var gotNonNil atomic.Bool
	err := app.BackgroundRun(FunctionTask(func(taskCtx context.Context) error {
		if taskCtx != nil {
			gotNonNil.Store(true)
		}
		return nil
	}))
	if err != nil {
		t.Fatalf("BackgroundRun: %v", err)
	}
	app.WaitGroup.Wait()
	if !gotNonNil.Load() {
		t.Fatal("task received nil context; expected context.Background()")
	}
}
