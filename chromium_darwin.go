//go:build darwin

package eletrocromo

// macOS browsers under /Applications are often not on PATH when the process is
// started from LaunchServices or a GUI launcher. Stable Chrome/Chromium/etc.
// are already on the cross-platform list; prepend the darwin extras (Arc,
// channel builds) so GetChromium finds them the same way linux/windows extras
// cover absolute installs.
func init() {
	chromiumLikes = append(darwinExtraChromiumLikes(), chromiumLikes...)
}
