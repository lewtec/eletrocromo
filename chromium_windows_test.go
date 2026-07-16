//go:build windows

package eletrocromo

import "testing"

func TestChromiumLikes_IncludesWindowsInstallPaths(t *testing.T) {
	want := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe`,
		`C:\Program Files (x86)\BraveSoftware\Brave-Browser\Application\brave.exe`,
	}
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
