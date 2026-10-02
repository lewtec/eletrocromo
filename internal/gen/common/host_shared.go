package common

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
)

//go:embed host_shared.swift
var hostSharedSwift string

// embed import is required for //go:embed. Reference it so the import stays used.
var _ embed.FS

// WriteHostSharedSwift installs Sources/HostShared.swift, the OpenDrop and
// ShareWatch helpers that the iOS and macOS hosts both compile.
func WriteHostSharedSwift(outDir string) error {
	out := strings.TrimSpace(outDir)
	if out == "" {
		return ErrOutDirRequired
	}
	out, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	dir := filepath.Join(out, "Sources")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "HostShared.swift"), []byte(hostSharedSwift), 0o644)
}
