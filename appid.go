package eletrocromo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// reverseDomainPattern is a conservative Android-style package / reverse-DNS id:
// at least two dot-separated labels of lowercase alphanumerics/underscores.
var reverseDomainPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)

// Sentinel errors for ValidateAppID (errors.Is / wrap with %w).
var (
	ErrAppIDRequired       = errors.New("app id is required (reverse-domain, e.g. br.tec.lew.myapp)")
	ErrAppIDTooLong        = errors.New("app id too long")
	ErrAppIDPathSeparators = errors.New("app id must not contain path separators")
	ErrAppIDFormat         = errors.New("app id must be reverse-domain notation (e.g. br.tec.lew.myapp)")
)

// ValidateAppID checks reverse-domain application identity (e.g. br.tec.lew.counter).
// The same string isolates Helium --user-data-dir and is intended for future APK package names.
func ValidateAppID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrAppIDRequired
	}
	if len(id) > 200 {
		return ErrAppIDTooLong
	}
	if strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
		return ErrAppIDPathSeparators
	}
	if !reverseDomainPattern.MatchString(id) {
		return fmt.Errorf("%w: %q", ErrAppIDFormat, id)
	}
	return nil
}

// ProfileDir returns the Helium --user-data-dir for appID:
// $XDG_DATA_HOME/eletrocromo/profiles/<appID> (or OS data-dir equivalent).
func ProfileDir(appID string) (string, error) {
	if err := ValidateAppID(appID); err != nil {
		return "", err
	}
	base, err := userDataDir()
	if err != nil {
		return "", fmt.Errorf("profile dir: %w", err)
	}
	dir := filepath.Join(base, "eletrocromo", "profiles", appID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("profile dir: %w", err)
	}
	return dir, nil
}

func userDataDir() (string, error) {
	if v := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return userDataDirFor(runtime.GOOS, home, os.Getenv), nil
}

// userDataDirFor picks the OS data directory when XDG_DATA_HOME is unset.
// Uses runtime GOOS (not path heuristics) so a Linux home that happens to
// contain Library/Application Support is not misclassified as macOS.
func userDataDirFor(goos, home string, getenv func(string) string) string {
	switch goos {
	case "windows":
		if v := strings.TrimSpace(getenv("LOCALAPPDATA")); v != "" {
			return v
		}
		return filepath.Join(home, "AppData", "Local")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support")
	default:
		// Linux and other Unix: XDG base dir when XDG_DATA_HOME is unset.
		return filepath.Join(home, ".local", "share")
	}
}
