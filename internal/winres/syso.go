// Package winres writes a Windows COFF object that embeds an .ico as the exe icon.
package winres

import "github.com/akavel/rsrc/rsrc"

// Syso writes a .syso the Go linker picks up for GOOS=windows.
// arch is a GOARCH name: 386, amd64, arm, or arm64.
func Syso(path, arch, icoPath string) error {
	return rsrc.Embed(path, arch, "", icoPath)
}
