//go:build linux

package eletrocromo

import "testing"

func TestChromiumLikes_IncludesLinuxPackageNames(t *testing.T) {
	want := []string{
		"brave-browser",
		"microsoft-edge-stable",
		"microsoft-edge",
		"vivaldi-stable",
	}
	have := make(map[string]struct{}, len(chromiumLikes))
	for _, name := range chromiumLikes {
		have[name] = struct{}{}
	}
	for _, name := range want {
		if _, ok := have[name]; !ok {
			t.Errorf("chromiumLikes missing Linux package name %q", name)
		}
	}
}
