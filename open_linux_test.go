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
