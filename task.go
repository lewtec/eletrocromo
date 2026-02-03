package eletrocromo

import "context"

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
