// Package hostfile queues share.Out onto Cache/share.jsonl for the native host.
package hostfile

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/lewtec/eletrocromo/driver"
	"github.com/lewtec/eletrocromo/driver/share"
)

var errNotPackaged = errors.New("not a packaged host")

func init() {
	driver.Register[share.Driver](factory{})
}

type factory struct{}

func (factory) ID() string    { return "hostfile" }
func (factory) Priority() int { return 10 }
func (factory) CheckCompatibility(context.Context) error {
	if strings.TrimSpace(os.Getenv("ELETROCROMO_CACHE_DIR")) != "" ||
		os.Getenv("ELETROCROMO_NO_UI") != "" {
		return nil
	}
	return errNotPackaged
}
func (factory) New(context.Context) (share.Driver, error) { return impl{}, nil }

type impl struct{}

func (impl) Out(_ context.Context, item share.Item) error {
	cache := strings.TrimSpace(os.Getenv("ELETROCROMO_CACHE_DIR"))
	if cache == "" {
		return errNotPackaged
	}
	return share.AppendFile(share.FilePath(cache), item)
}
