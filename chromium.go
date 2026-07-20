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

// chromiumLikes are secondary Chromium-family hosts used only when Helium is
// not already on PATH. Platform init hooks may prepend absolute paths.
var chromiumLikes = []string{
	"C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe", // we hate it but we can count it's there
	"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
	"/Applications/Chromium.app/Contents/MacOS/Chromium",
	"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
	"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
	"/Applications/Vivaldi.app/Contents/MacOS/Vivaldi",
	"/Applications/Opera.app/Contents/MacOS/Opera",
	"msedge",
	"brave",
	"vivaldi",
	"opera",
	"chromium",
	"chrome",
	"google-chrome",
	"google-chrome-stable",
	"chromium-browser",
}

// heliumCandidates are preferred app-window hosts (registry binary name: helium).
var heliumCandidates = []string{
	"helium",
}

// ErrNoChromium is returned when no Chromium-like --app host can be resolved
// (local Helium, secondary discovers, or workspaced ensure of helium-browser).
var ErrNoChromium = errors.New("no app-window browser host found")

// lookPath is exec.LookPath; tests may override.
var lookPath = exec.LookPath

// GetChromium searches for a local Chromium-based browser that supports --app.
// Helium is preferred; other chromiumLikes are secondary. It does not download
// or call workspaced — see ResolveBrowserHost.
func GetChromium() (string, error) {
	candidates := make([]string, 0, len(heliumCandidates)+len(chromiumLikes))
	candidates = append(candidates, heliumCandidates...)
	candidates = append(candidates, chromiumLikes...)
	for _, ch := range candidates {
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

// ResolveBrowserHost finds a binary suitable for --app launch.
//
// Order (SPEC): local Helium → secondary Chromium-likes → ensure Helium via
// workspaced (tool which helium-browser helium, bootstrapping workspaced if
// needed) → error. Never opens the system default browser.
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

// LaunchChromium opens the specified URL in app mode (--app).
// Host resolution follows ResolveBrowserHost (uses ctx for ensure only).
// Only http(s) schemes are allowed. Never falls back to the system browser.
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
