//go:build !linux && !freebsd && !openbsd && !netbsd && !dragonfly && !solaris && !illumos && !aix && !darwin && !windows

package eletrocromo

import (
	"fmt"
	"net/url"
	"runtime"
)

func openSystemBrowser(u *url.URL) error {
	return fmt.Errorf("open system browser: unsupported OS %s", runtime.GOOS)
}
