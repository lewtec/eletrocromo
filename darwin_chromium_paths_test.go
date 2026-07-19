package eletrocromo

import "testing"

func TestDarwinExtraChromiumLikes_IncludesCommonInstalls(t *testing.T) {
	paths := darwinExtraChromiumLikes()
	want := []string{
		"/Applications/Arc.app/Contents/MacOS/Arc",
		"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
		"/Applications/Google Chrome Beta.app/Contents/MacOS/Google Chrome Beta",
		"/Applications/Microsoft Edge Beta.app/Contents/MacOS/Microsoft Edge Beta",
		"/Applications/Brave Browser Beta.app/Contents/MacOS/Brave Browser Beta",
	}
	assertPathSetContains(t, paths, want)
}

func TestDarwinExtraChromiumLikes_NonEmpty(t *testing.T) {
	if got := darwinExtraChromiumLikes(); len(got) == 0 {
		t.Fatal("darwinExtraChromiumLikes returned empty list")
	}
}
