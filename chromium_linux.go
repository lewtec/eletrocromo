//go:build linux

package eletrocromo

// Linux distro packages often install browsers under names that are not on the
// cross-platform list (e.g. brave-browser vs brave, microsoft-edge-stable vs
// msedge). Absolute installs under /opt and /snap are frequently missing from
// PATH when the process is started from a GUI launcher. Prepend those paths
// and append the package names so GetChromium finds common installs.
func init() {
	chromiumLikes = append(linuxExtraChromiumLikes(), chromiumLikes...)
	chromiumLikes = append(chromiumLikes,
		"brave-browser",
		"microsoft-edge-stable",
		"microsoft-edge",
		"vivaldi-stable",
	)
}
