package eletrocromo

import (
	"fmt"
	"net/url"
)

// openSystemBrowser is the former OS-default-browser fallback from LaunchChromium.
// SPEC forbids that path: a missing Chromium-like host must fail closed so the
// auth token URL never opens in full browser chrome (xdg-open, open, start, …).
// Kept as the hook LaunchChromium already calls; return ErrNoChromium.
func openSystemBrowser(_ *url.URL) error {
	return fmt.Errorf("%w: install Helium or another Chromium-based browser", ErrNoChromium)
}
