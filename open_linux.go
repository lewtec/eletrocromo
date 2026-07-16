//go:build linux || freebsd || openbsd || netbsd || dragonfly || solaris || illumos || aix

package eletrocromo

import (
	"fmt"
	"net/url"
	"os/exec"
)

// openSystemBrowser opens u with the desktop "open URL" helper.
// Accepts loopback addresses that httptest uses (127.0.0.1 / ::1).
func openSystemBrowser(u *url.URL) error {
	raw := u.String()
	// Prefer portal/X helpers; fall back to a few common browsers by name.
	for _, candidate := range [][]string{
		{"xdg-open", raw},
		{"sensible-browser", raw},
		{"gio", "open", raw},
		{"wslview", raw},
	} {
		cmd := exec.Command(candidate[0], candidate[1:]...)
		if err := cmd.Start(); err != nil {
			continue
		}
		// Reap so a finished helper does not leave a zombie.
		go func() { _ = cmd.Wait() }()
		return nil
	}
	return fmt.Errorf("open system browser: no helper worked for %s", raw)
}
