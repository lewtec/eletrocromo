// Package driver is a small Register/Get registry for platform capabilities.
//
// Each capability lives in its own package under driver/<name> (interface +
// facade). Concrete implementations live in driver/<name>/<impl> and call
// Register from init. There is no prelude: import the impl you want.
package driver

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sync"
)

// Registry sentinels (errors.Is).
var (
	ErrNotInterface = errors.New("driver type is not an interface")
	ErrNotFound     = errors.New("no driver registered")
	ErrUnavailable  = errors.New("no compatible driver")
)

// Factory constructs one implementation of capability T.
type Factory[T any] interface {
	ID() string
	// Priority prefers a factory when several pass CheckCompatibility.
	// Higher wins; ties break on ID.
	Priority() int
	CheckCompatibility(context.Context) error
	New(context.Context) (T, error)
}

var (
	mu      sync.RWMutex
	drivers = map[reflect.Type]map[string]any{}
)

// Register adds a factory for T. T must be an interface. Duplicate IDs panic
// (programmer error at init).
func Register[T any](factory Factory[T]) {
	t := reflect.TypeFor[T]()
	if t.Kind() != reflect.Interface {
		panic("driver " + t.String() + " is not an interface")
	}
	id := factory.ID()
	if id == "" {
		panic("driver for " + t.String() + " registered with empty id")
	}

	mu.Lock()
	defer mu.Unlock()
	byID := drivers[t]
	if byID == nil {
		byID = make(map[string]any)
		drivers[t] = byID
	}
	if _, ok := byID[id]; ok {
		panic("driver id " + id + " already registered for " + t.String())
	}
	byID[id] = factory
}

// Get returns the highest-priority compatible implementation of T.
func Get[T any](ctx context.Context) (T, error) {
	var zero T
	t := reflect.TypeFor[T]()
	if t.Kind() != reflect.Interface {
		return zero, ErrNotInterface
	}

	mu.RLock()
	byID := drivers[t]
	factories := make([]Factory[T], 0, len(byID))
	for _, entry := range byID {
		factories = append(factories, entry.(Factory[T]))
	}
	mu.RUnlock()

	if len(factories) == 0 {
		return zero, fmt.Errorf("%w for %s", ErrNotFound, t)
	}

	slices.SortFunc(factories, func(a, b Factory[T]) int {
		if a.Priority() != b.Priority() {
			return b.Priority() - a.Priority()
		}
		if a.ID() < b.ID() {
			return -1
		}
		if a.ID() > b.ID() {
			return 1
		}
		return 0
	})

	var skipped []error
	for _, factory := range factories {
		if err := factory.CheckCompatibility(ctx); err != nil {
			skipped = append(skipped, fmt.Errorf("%s: %w", factory.ID(), err))
			continue
		}
		inst, err := factory.New(ctx)
		if err != nil {
			skipped = append(skipped, fmt.Errorf("%s: %w", factory.ID(), err))
			continue
		}
		return inst, nil
	}
	return zero, fmt.Errorf("%w for %s: %w", ErrUnavailable, t, errors.Join(skipped...))
}

// With loads T and runs fn.
func With[T any](ctx context.Context, fn func(T) error) error {
	d, err := Get[T](ctx)
	if err != nil {
		return err
	}
	return fn(d)
}

// WithResult is With for functions that return a value.
func WithResult[T, R any](ctx context.Context, fn func(T) (R, error)) (R, error) {
	var zero R
	d, err := Get[T](ctx)
	if err != nil {
		return zero, err
	}
	return fn(d)
}

func reset() {
	mu.Lock()
	defer mu.Unlock()
	clear(drivers)
}
