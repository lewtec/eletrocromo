//go:build linux

package eletrocromo

// Linux distro packages often install browsers under names that are not on the
// cross-platform list (e.g. brave-browser vs brave, microsoft-edge-stable vs
// msedge). Append them so GetChromium finds common installs via PATH.
func init() {
	chromiumLikes = append(chromiumLikes,
		"brave-browser",
		"microsoft-edge-stable",
		"microsoft-edge",
		"vivaldi-stable",
	)
}
