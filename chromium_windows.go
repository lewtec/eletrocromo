//go:build windows

package eletrocromo

import "os"

// Windows browsers are often installed under Program Files or per-user under
// LocalAppData, neither of which is necessarily on PATH. Prepend common
// absolute install locations so GetChromium finds Chrome, Edge, and Brave.
func init() {
	chromiumLikes = append(windowsExtraChromiumLikes(os.Getenv("LOCALAPPDATA")), chromiumLikes...)
}
