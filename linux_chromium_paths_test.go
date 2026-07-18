package eletrocromo

import "testing"

func TestLinuxExtraChromiumLikes_IncludesCommonInstalls(t *testing.T) {
	paths := linuxExtraChromiumLikes()
	want := []string{
		"/opt/google/chrome/google-chrome",
		"/opt/brave.com/brave/brave",
		"/opt/microsoft/msedge/msedge",
		"/opt/vivaldi/vivaldi",
		"/snap/bin/chromium",
		"/usr/lib/chromium/chromium",
		"/usr/lib/chromium-browser/chromium-browser",
	}
	assertPathSetContains(t, paths, want)
}

func TestLinuxExtraChromiumLikes_NonEmpty(t *testing.T) {
	if got := linuxExtraChromiumLikes(); len(got) == 0 {
		t.Fatal("linuxExtraChromiumLikes returned empty list")
	}
}
