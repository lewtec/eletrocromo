package eletrocromo

import "testing"

func TestWindowsExtraChromiumLikes_ProgramFiles(t *testing.T) {
	paths := windowsExtraChromiumLikes("")
	want := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe`,
		`C:\Program Files (x86)\BraveSoftware\Brave-Browser\Application\brave.exe`,
	}
	assertPathSetContains(t, paths, want)
}

func TestWindowsExtraChromiumLikes_IncludesUserLocalInstalls(t *testing.T) {
	local := `C:\Users\dev\AppData\Local`
	paths := windowsExtraChromiumLikes(local)
	want := []string{
		local + `\Google\Chrome\Application\chrome.exe`,
		local + `\Microsoft\Edge\Application\msedge.exe`,
		local + `\BraveSoftware\Brave-Browser\Application\brave.exe`,
	}
	assertPathSetContains(t, paths, want)
	// Program Files entries must still be present when LocalAppData is set.
	assertPathSetContains(t, paths, []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
	})
}

func assertPathSetContains(t *testing.T, haveList, want []string) {
	t.Helper()
	have := make(map[string]struct{}, len(haveList))
	for _, p := range haveList {
		have[p] = struct{}{}
	}
	for _, p := range want {
		if _, ok := have[p]; !ok {
			t.Errorf("missing path %q", p)
		}
	}
}
