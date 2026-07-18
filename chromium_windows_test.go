//go:build windows

package eletrocromo

import (
	"os"
	"testing"
)

func TestChromiumLikes_IncludesWindowsInstallPaths(t *testing.T) {
	// Integration: init() prepended windowsExtraChromiumLikes into chromiumLikes.
	want := windowsExtraChromiumLikes(os.Getenv("LOCALAPPDATA"))
	have := make(map[string]struct{}, len(chromiumLikes))
	for _, name := range chromiumLikes {
		have[name] = struct{}{}
	}
	for _, name := range want {
		if _, ok := have[name]; !ok {
			t.Errorf("chromiumLikes missing Windows install path %q", name)
		}
	}
}
