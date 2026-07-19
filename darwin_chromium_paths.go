package eletrocromo

// darwinExtraChromiumLikes returns absolute install paths for Chromium-like
// browsers on macOS that are not already on the cross-platform list.
// Kept free of GOOS build tags so unit tests run on any CI; only the darwin
// init hook prepends these to chromiumLikes.
//
// Stable Chrome/Chromium/Brave/Edge/Vivaldi/Opera under /Applications are
// already in chromiumLikes. This list covers Arc (popular, Chromium-based)
// and the main Google/Microsoft/Brave channel builds that ship as separate
// .app bundles and are often absent from PATH when the process is started
// from a GUI.
func darwinExtraChromiumLikes() []string {
	return []string{
		"/Applications/Arc.app/Contents/MacOS/Arc",
		"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
		"/Applications/Google Chrome Beta.app/Contents/MacOS/Google Chrome Beta",
		"/Applications/Microsoft Edge Beta.app/Contents/MacOS/Microsoft Edge Beta",
		"/Applications/Brave Browser Beta.app/Contents/MacOS/Brave Browser Beta",
	}
}
