package eletrocromo

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// heliumCandidates are local Helium binary names (workspaced registry bin: helium).
var heliumCandidates = []string{
	"helium",
}

// ErrNoChromium is returned when Helium cannot be resolved (local PATH or
// workspaced ensure of registry helium-browser).
var ErrNoChromium = errors.New("no Helium browser host found")

// lookPath is exec.LookPath; tests may override.
var lookPath = exec.LookPath

// GetChromium returns a local Helium binary path if present on PATH.
// It does not download or call workspaced — see ResolveBrowserHost.
// Name kept for compatibility; only Helium is supported as the app window host.
func GetChromium() (string, error) {
	for _, ch := range heliumCandidates {
		path, err := lookPath(ch)
		if errors.Is(err, exec.ErrNotFound) {
			continue
		}
		if err != nil {
			continue
		}
		return path, nil
	}
	return "", ErrNoChromium
}

// ResolveBrowserHost finds Helium for --app launch.
//
// Order (SPEC): local Helium → ensure via workspaced (tool which helium-browser
// helium, bootstrapping workspaced if needed) → error.
// Never opens Chrome/Edge/system browser.
//
// Set ELETROCROMO_NO_ENSURE=1 to skip network ensure (tests/CI).
func ResolveBrowserHost(ctx context.Context) (string, error) {
	if path, err := GetChromium(); err == nil {
		return path, nil
	}
	if ensureDisabled() {
		return "", fmt.Errorf("%w: install Helium, or allow ensure (unset ELETROCROMO_NO_ENSURE)", ErrNoChromium)
	}
	path, err := ensureHeliumBrowser(ctx)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrNoChromium, err)
	}
	return path, nil
}

func ensureDisabled() bool {
	v := strings.TrimSpace(os.Getenv("ELETROCROMO_NO_ENSURE"))
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

// LaunchChromium opens the URL in Helium app mode (--app).
// Host resolution follows ResolveBrowserHost (uses ctx for ensure only).
// Only http(s) schemes are allowed.
func LaunchChromium(ctx context.Context, u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: %s", u.Scheme)
	}
	bin, err := ResolveBrowserHost(ctx)
	if err != nil {
		return err
	}
	// Start without binding the browser lifetime to ctx yet; window-owned
	// process wait is a later SPEC item. Ensure still respects ctx.
	cmd := exec.Command(bin, "--app", u.String())
	return cmd.Start()
}
