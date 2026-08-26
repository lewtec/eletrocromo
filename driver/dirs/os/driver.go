// Package os is the default dirs driver: env overrides, then OS data/cache/config homes.
package os

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lewtec/eletrocromo/driver"
	"github.com/lewtec/eletrocromo/driver/dirs"
)

const (
	envData   = "ELETROCROMO_DATA_DIR"
	envCache  = "ELETROCROMO_CACHE_DIR"
	envConfig = "ELETROCROMO_CONFIG_DIR"
)

func init() {
	driver.Register[dirs.Driver](factory{})
}

type factory struct{}

func (factory) ID() string                               { return "os" }
func (factory) Priority() int                            { return 0 }
func (factory) CheckCompatibility(context.Context) error { return nil }
func (factory) New(context.Context) (dirs.Driver, error) { return impl{}, nil }

type impl struct{}

func (impl) Resolve(_ context.Context, appID string) (dirs.Dirs, error) {
	out, err := paths(appID, runtime.GOOS, os.Getenv, os.UserHomeDir, os.UserCacheDir, os.UserConfigDir)
	if err != nil {
		return dirs.Dirs{}, err
	}
	for _, dir := range []string{out.Data, out.Cache, out.Config, out.Inbox} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return dirs.Dirs{}, fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	return out, nil
}

func paths(
	appID, goos string,
	getenv func(string) string,
	homeFn func() (string, error),
	cacheFn func() (string, error),
	configFn func() (string, error),
) (dirs.Dirs, error) {
	data, err := pick(getenv(envData), func() (string, error) {
		base, err := dataHome(goos, getenv, homeFn)
		if err != nil {
			return "", err
		}
		return filepath.Join(base, "eletrocromo", "apps", appID, "data"), nil
	})
	if err != nil {
		return dirs.Dirs{}, fmt.Errorf("data dir: %w", err)
	}
	cache, err := pick(getenv(envCache), func() (string, error) {
		base, err := cacheFn()
		if err != nil {
			return "", err
		}
		return filepath.Join(base, "eletrocromo", "apps", appID, "cache"), nil
	})
	if err != nil {
		return dirs.Dirs{}, fmt.Errorf("cache dir: %w", err)
	}
	config, err := pick(getenv(envConfig), func() (string, error) {
		base, err := configFn()
		if err != nil {
			return "", err
		}
		return filepath.Join(base, "eletrocromo", "apps", appID, "config"), nil
	})
	if err != nil {
		return dirs.Dirs{}, fmt.Errorf("config dir: %w", err)
	}

	return dirs.Dirs{
		Data:   data,
		Cache:  cache,
		Config: config,
		Inbox:  filepath.Join(cache, "inbox"),
	}, nil
}

func pick(env string, fallback func() (string, error)) (string, error) {
	if v := strings.TrimSpace(env); v != "" {
		return v, nil
	}
	return fallback()
}

func dataHome(goos string, getenv func(string) string, homeFn func() (string, error)) (string, error) {
	if v := getenv("XDG_DATA_HOME"); v != "" {
		return v, nil
	}
	home, err := homeFn()
	if err != nil {
		return "", err
	}
	switch goos {
	case "windows":
		if v := getenv("LOCALAPPDATA"); v != "" {
			return v, nil
		}
		return filepath.Join(home, "AppData", "Local"), nil
	case "darwin":
		return filepath.Join(home, "Library", "Application Support"), nil
	default:
		return filepath.Join(home, ".local", "share"), nil
	}
}
