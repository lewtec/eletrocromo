// Package dirs resolves per-app storage locations.
//
// Inbox is always Cache/inbox so the OS may reclaim it with cache.
// Import an implementation (driver/dirs/os) so Register runs.
package dirs

import (
	"context"

	"github.com/lewtec/eletrocromo"
	"github.com/lewtec/eletrocromo/driver"
)

// Dirs is the app-private tree. Inbox is a subdirectory of Cache.
type Dirs struct {
	Data   string
	Cache  string
	Config string
	Inbox  string
}

// Driver resolves directories for one app id.
type Driver interface {
	Resolve(ctx context.Context, appID string) (Dirs, error)
}

// Resolve picks a dirs driver and returns locations for appID.
func Resolve(ctx context.Context, appID string) (Dirs, error) {
	if err := eletrocromo.ValidateAppID(appID); err != nil {
		return Dirs{}, err
	}
	return driver.WithResult(ctx, func(d Driver) (Dirs, error) {
		return d.Resolve(ctx, appID)
	})
}
