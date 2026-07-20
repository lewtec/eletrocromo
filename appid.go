package eletrocromo

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// reverseDomainPattern is a conservative Android-style package / reverse-DNS id:
// at least two dot-separated labels of lowercase alphanumerics/underscores.
var reverseDomainPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)

// ValidateAppID checks reverse-domain application identity (e.g. br.tec.lew.counter).
// The same string isolates Helium --user-data-dir and is intended for future APK package names.
func ValidateAppID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("app id is required (reverse-domain, e.g. br.tec.lew.myapp)")
	}
	if len(id) > 200 {
		return fmt.Errorf("app id too long")
	}
	if strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
		return fmt.Errorf("app id must not contain path separators")
	}
	if !reverseDomainPattern.MatchString(id) {
		return fmt.Errorf("app id %q must be reverse-domain notation (e.g. br.tec.lew.myapp)", id)
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
	if os.PathSeparator == '\\' {
		if v := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); v != "" {
			return v, nil
		}
		return filepath.Join(home, "AppData", "Local"), nil
	}
	// macOS
	if st, err := os.Stat(filepath.Join(home, "Library", "Application Support")); err == nil && st.IsDir() {
		return filepath.Join(home, "Library", "Application Support"), nil
	}
	return filepath.Join(home, ".local", "share"), nil
}
