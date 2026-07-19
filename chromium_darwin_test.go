//go:build darwin

package eletrocromo

import "testing"

func TestChromiumLikes_IncludesDarwinAbsolutePaths(t *testing.T) {
	// Integration: init() prepended darwinExtraChromiumLikes into chromiumLikes.
	want := darwinExtraChromiumLikes()
	have := make(map[string]struct{}, len(chromiumLikes))
	for _, name := range chromiumLikes {
		have[name] = struct{}{}
	}
	for _, name := range want {
		if _, ok := have[name]; !ok {
			t.Errorf("chromiumLikes missing Darwin install path %q", name)
		}
	}
}
