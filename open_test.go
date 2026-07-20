package eletrocromo

import (
	"errors"
	"net/url"
	"testing"
)

func TestOpenSystemBrowser_FailsClosed(t *testing.T) {
	for _, raw := range []string{
		"http://127.0.0.1:9/?token=test",
		"http://localhost:9/",
		"https://example.com/",
	} {
		t.Run(raw, func(t *testing.T) {
			u, err := url.Parse(raw)
			if err != nil {
				t.Fatal(err)
			}
			err = openSystemBrowser(u)
			if err == nil {
				t.Fatal("expected error; system-browser fallback is forbidden")
			}
			if !errors.Is(err, ErrNoChromium) {
				t.Fatalf("want ErrNoChromium, got %v", err)
			}
		})
	}
}
