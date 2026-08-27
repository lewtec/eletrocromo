package icons

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolveIconRoot returns iconRoot when set, otherwise generates the default
// icon tree under workDir/.eletrocromo-icons (apk/ios/mac Build fallback).
func ResolveIconRoot(iconRoot, workDir string) (string, error) {
	iconRoot = strings.TrimSpace(iconRoot)
	if iconRoot != "" {
		return iconRoot, nil
	}
	tmpIcons := filepath.Join(workDir, ".eletrocromo-icons")
	if _, err := Generate(Options{OutputDir: tmpIcons, Force: true}); err != nil {
		return "", fmt.Errorf("default icons: %w", err)
	}
	return tmpIcons, nil
}
