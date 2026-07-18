package eletrocromo

// linuxExtraChromiumLikes returns absolute install paths for common
// Chromium-like browsers on Linux. Kept free of GOOS build tags so unit
// tests run on any CI; only the linux init hook prepends these to
// chromiumLikes.
//
// Distro packages usually also put a name on PATH, but GUI launchers and
// minimal environments often omit /opt and /snap from PATH. Absolute paths
// cover those installs the same way windowsExtraChromiumLikes covers
// Program Files.
func linuxExtraChromiumLikes() []string {
	return []string{
		"/opt/google/chrome/google-chrome",
		"/opt/google/chrome/chrome",
		"/opt/brave.com/brave/brave",
		"/opt/microsoft/msedge/msedge",
		"/opt/vivaldi/vivaldi",
		"/snap/bin/chromium",
		"/usr/lib/chromium/chromium",
		"/usr/lib/chromium-browser/chromium-browser",
	}
}
