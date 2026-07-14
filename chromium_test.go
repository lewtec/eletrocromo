package eletrocromo

import (
	"net/url"
	"strings"
	"testing"
)

func TestLaunchChromium_RejectsNonHTTPSchemes(t *testing.T) {
	cases := []string{
		"file:///etc/passwd",
		"javascript:alert(1)",
		"ftp://example.com/",
		"data:text/html,hi",
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			u, err := url.Parse(raw)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			err = LaunchChromium(u)
			if err == nil {
				t.Fatal("expected error for non-http(s) scheme")
			}
			if !strings.Contains(err.Error(), "invalid URL scheme") {
				t.Fatalf("expected scheme error, got %v", err)
			}
		})
	}
}

func TestGetChromium_NoPanic(t *testing.T) {
	path, err := GetChromium()
	if err == nil {
		if path == "" {
			t.Fatal("empty path with nil error")
		}
		return
	}
	if err != ErrNoChromium {
		t.Fatalf("unexpected error: %v", err)
	}
}
