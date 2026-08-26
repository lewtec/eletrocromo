// Package notification posts a local OS notification.
//
// Import an implementation (for example driver/notification/discard) so
// Register runs. Notify fails closed when none is registered.
package notification

import (
	"context"

	"github.com/lewtec/eletrocromo/driver"
)

// Notification is a local alert. Keep it small until a host needs more fields.
type Notification struct {
	Title   string
	Message string
}

// Driver posts one notification.
type Driver interface {
	Notify(ctx context.Context, n Notification) error
}

// Notify posts n using the selected driver.
func Notify(ctx context.Context, n Notification) error {
	return driver.With(ctx, func(d Driver) error {
		return d.Notify(ctx, n)
	})
}
