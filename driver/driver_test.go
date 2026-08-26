package driver_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lewtec/eletrocromo/driver"
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
	if err != nil {
		t.Fatal(err)
	}
	if got.Name() != "high" {
		t.Fatalf("Name() = %q; want high", got.Name())
	}
}

func TestGet_NotFound(t *testing.T) {
	driver.Reset()
	t.Cleanup(driver.Reset)

	_, err := driver.Get[probe](t.Context())
	if !errors.Is(err, driver.ErrNotFound) {
		t.Fatalf("err = %v; want ErrNotFound", err)
	}
}

func TestGet_Unavailable(t *testing.T) {
	driver.Reset()
	t.Cleanup(driver.Reset)

	driver.Register[probe](factory{id: "x", compat: errNope})
	_, err := driver.Get[probe](t.Context())
	if !errors.Is(err, driver.ErrUnavailable) {
		t.Fatalf("err = %v; want ErrUnavailable", err)
	}
	if !errors.Is(err, errNope) {
		t.Fatalf("err = %v; want errNope", err)
	}
}

func TestGet_NotInterface(t *testing.T) {
	_, err := driver.Get[int](t.Context())
	if !errors.Is(err, driver.ErrNotInterface) {
		t.Fatalf("err = %v; want ErrNotInterface", err)
	}
}

func TestRegister_DuplicateIDPanics(t *testing.T) {
	driver.Reset()
	t.Cleanup(driver.Reset)

	driver.Register[probe](factory{id: "once"})
	defer func() {
		if recover() == nil {
			t.Fatal("want panic")
		}
	}()
	driver.Register[probe](factory{id: "once"})
}

func TestWithResult(t *testing.T) {
	driver.Reset()
	t.Cleanup(driver.Reset)

	driver.Register[probe](factory{id: "only", name: "ok"})
	name, err := driver.WithResult(t.Context(), func(p probe) (string, error) {
		return p.Name(), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if name != "ok" {
		t.Fatalf("name = %q; want ok", name)
	}
}
