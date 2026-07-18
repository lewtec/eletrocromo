//go:build linux || freebsd || openbsd || netbsd || dragonfly || solaris || illumos || aix

package eletrocromo

import (
	"net/url"
	"strings"
	"testing"
)

// Regression: the old gopen fallback rejected loopback URLs that httptest uses.
// openSystemBrowser must accept them (it may still fail if no helper is installed).
func TestOpenSystemBrowser_AcceptsLoopbackURL(t *testing.T) {
	u, err := url.Parse("http://127.0.0.1:9/?token=test")
	if err != nil {
		t.Fatal(err)
	}
	err = openSystemBrowser(u)
	if err != nil {
		// Must not be a URL-shape rejection (gopen returned ErrInvalidURL).
		if strings.Contains(err.Error(), "invalid URL") {
			t.Fatalf("loopback URL rejected as invalid: %v", err)
		}
		t.Logf("openSystemBrowser returned (ok if no desktop helper): %v", err)
	}
}

func TestOpenSystemBrowser_AcceptsLocalhostURL(t *testing.T) {
	u, err := url.Parse("http://localhost:9/")
	if err != nil {
		t.Fatal(err)
	}
	err = openSystemBrowser(u)
	if err != nil && strings.Contains(err.Error(), "invalid URL") {
		t.Fatalf("localhost URL rejected as invalid: %v", err)
	}
}

// GetChromium only discovers Chromium-likes; the system-open fallback must
// still try Firefox when desktop helpers are missing.
func TestSystemBrowserCmds_IncludesFirefox(t *testing.T) {
	cmds := systemBrowserCmds("http://127.0.0.1:9/")
	wantHeads := []string{"xdg-open", "sensible-browser", "gio", "wslview", "firefox", "firefox-esr"}
	if len(cmds) < len(wantHeads) {
		t.Fatalf("got %d candidates, want at least %d", len(cmds), len(wantHeads))
	}
	have := make(map[string]struct{}, len(cmds))
	for _, c := range cmds {
		if len(c) == 0 {
			t.Fatal("empty candidate argv")
		}
		have[c[0]] = struct{}{}
	}
	for _, name := range wantHeads {
		if _, ok := have[name]; !ok {
			t.Errorf("systemBrowserCmds missing %q", name)
		}
	}
}
