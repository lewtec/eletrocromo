//go:build linux || freebsd || openbsd || netbsd || dragonfly || solaris || illumos || aix

package eletrocromo

import (
	"fmt"
	"net/url"
	"os/exec"
)

// systemBrowserCmds returns argv candidates for openSystemBrowser, tried in order.
// Desktop helpers first; then non-Chromium browsers GetChromium will not find
// (Firefox). Chromium-likes are already handled by LaunchChromium before this
// fallback runs.
func systemBrowserCmds(raw string) [][]string {
	return [][]string{
		{"xdg-open", raw},
		{"sensible-browser", raw},
		{"gio", "open", raw},
		{"wslview", raw},
		{"firefox", raw},
		{"firefox-esr", raw},
	}
}

// openSystemBrowser opens u with the desktop "open URL" helper.
// Accepts loopback addresses that httptest uses (127.0.0.1 / ::1).
func openSystemBrowser(u *url.URL) error {
	raw := u.String()
	var lastErr error
	for _, candidate := range systemBrowserCmds(raw) {
		cmd := exec.Command(candidate[0], candidate[1:]...)
		if err := cmd.Start(); err != nil {
			lastErr = err
			continue
		}
		// Reap so a finished helper does not leave a zombie.
		go func() { _ = cmd.Wait() }()
		return nil
	}
	if lastErr != nil {
		return fmt.Errorf("open system browser: no helper worked for %s: %w", raw, lastErr)
	}
	return fmt.Errorf("open system browser: no helper worked for %s", raw)
}
