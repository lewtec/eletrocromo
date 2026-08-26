// Package discard is a no-op notification driver (tests, headless).
package discard

import (
	"context"

	"github.com/lewtec/eletrocromo/driver"
	"github.com/lewtec/eletrocromo/driver/notification"
)

func init() {
	driver.Register[notification.Driver](factory{})
}

type factory struct{}

func (factory) ID() string                               { return "discard" }
func (factory) Priority() int                            { return 0 }
func (factory) CheckCompatibility(context.Context) error { return nil }
func (factory) New(context.Context) (notification.Driver, error) {
	return impl{}, nil
}

type impl struct{}

func (impl) Notify(context.Context, notification.Notification) error { return nil }
