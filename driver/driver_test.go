package driver_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lewtec/eletrocromo/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errMissing = errors.New("missing")
	errNope    = errors.New("nope")
)

type probe interface {
	Name() string
}

type factory struct {
	id       string
	priority int
	compat   error
	newErr   error
	name     string
}

func (f factory) ID() string { return f.id }
func (f factory) Priority() int {
	return f.priority
}
func (f factory) CheckCompatibility(context.Context) error { return f.compat }
func (f factory) New(context.Context) (probe, error) {
	if f.newErr != nil {
		return nil, f.newErr
	}
	return impl{name: f.name}, nil
}

type impl struct{ name string }

func (i impl) Name() string { return i.name }

func TestGet_PicksHighestPriorityCompatible(t *testing.T) {
	driver.Reset()
	t.Cleanup(driver.Reset)

	driver.Register[probe](factory{id: "low", priority: 1, name: "low"})
	driver.Register[probe](factory{
		id: "broken", priority: 10, compat: errMissing, name: "broken",
	})
	driver.Register[probe](factory{id: "high", priority: 5, name: "high"})

	got, err := driver.Get[probe](t.Context())
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "high", got.Name())
}

func TestGet_NotFound(t *testing.T) {
	driver.Reset()
	t.Cleanup(driver.Reset)

	_, err := driver.Get[probe](t.Context())
	require.ErrorIs(t, err, driver.ErrNotFound)
}

func TestGet_Unavailable(t *testing.T) {
	driver.Reset()
	t.Cleanup(driver.Reset)

	driver.Register[probe](factory{id: "x", compat: errNope})
	_, err := driver.Get[probe](t.Context())
	assert.ErrorIs(t, err, driver.ErrUnavailable)
	assert.ErrorIs(t, err, errNope)
}

func TestGet_NotInterface(t *testing.T) {
	_, err := driver.Get[int](t.Context())
	require.ErrorIs(t, err, driver.ErrNotInterface)
}

func TestRegister_DuplicateIDPanics(t *testing.T) {
	driver.Reset()
	t.Cleanup(driver.Reset)

	driver.Register[probe](factory{id: "once"})
	require.Panics(t, func() {
		driver.Register[probe](factory{id: "once"})
	})
}

func TestWithResult(t *testing.T) {
	driver.Reset()
	t.Cleanup(driver.Reset)

	driver.Register[probe](factory{id: "only", name: "ok"})
	name, err := driver.WithResult(t.Context(), func(p probe) (string, error) {
		return p.Name(), nil
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", name)
}
