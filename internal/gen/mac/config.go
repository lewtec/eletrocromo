// Package mac generates an ephemeral macOS WKWebView host and packages
// a CGo-less darwin Go binary into an unsigned Debug .app.
//
// The importable eletrocromo library stays free of Xcode; this package only
// writes an XcodeGen tree that runs the Go server and opens WKWebView.
package mac

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/lewtec/eletrocromo"
	"github.com/lewtec/eletrocromo/internal/version"
)

// Config / path sentinels.
var (
	ErrConfigPathEmpty = errors.New("config path is empty")
	ErrGoMainNotDir    = errors.New("go_main must be a directory (main package)")
	ErrOutDirRequired  = errors.New("out dir is required")
	ErrOutPathNotDir   = errors.New("out path exists and is not a directory")
	ErrOutDirNotEmpty  = errors.New("out dir is not empty (use --force)")
)

// HelperName is the Go child binary inside Contents/MacOS.
const HelperName = "eletrocromo-server"

// Config is the project identity written into the generated tree.
type Config struct {
	PackageID   string `json:"package_id"`
	AppName     string `json:"app_name"`
	VersionName string `json:"version_name"`
	VersionCode int    `json:"version_code"`
	GoMain      string `json:"go_main"`
	Icon        string `json:"icon,omitempty"`
}

// ProductName is a filesystem-safe Xcode PRODUCT_NAME / .app stem.
func (c Config) ProductName() string {
	return productName(c.PackageID, c.AppName)
}

func productName(packageID, appName string) string {
	name := strings.TrimSpace(appName)
	name = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_':
			return r
		case r == ' ':
			return '-'
		default:
			return -1
		}
	}, name)
	name = strings.Trim(name, "-_")
	if name == "" {
		parts := strings.Split(packageID, ".")
		name = parts[len(parts)-1]
	}
	if name == "" {
		name = "App"
	}
	return name
}

func (c Config) withDefaults() (Config, error) {
	c.PackageID = strings.TrimSpace(c.PackageID)
	if err := eletrocromo.ValidateAppID(c.PackageID); err != nil {
		return Config{}, fmt.Errorf("package id: %w", err)
	}
	c.AppName = strings.TrimSpace(c.AppName)
	if c.AppName == "" {
		parts := strings.Split(c.PackageID, ".")
		c.AppName = parts[len(parts)-1]
	}
	if c.VersionName == "" || c.VersionCode <= 0 {
		info := version.Resolve()
		if c.VersionName == "" {
			c.VersionName = info.AndroidName()
		}
		if c.VersionCode <= 0 {
			c.VersionCode = version.AndroidCodeFrom(info.Version, 0)
		}
	}
	if strings.TrimSpace(c.GoMain) == "" {
		c.GoMain = "."
	}
	return c, nil
}

func encodeConfigJSON(cfg Config) ([]byte, error) {
	doc := struct {
		SchemaVersion int    `json:"schema_version"`
		PackageID     string `json:"package_id"`
		AppName       string `json:"app_name"`
		VersionName   string `json:"version_name"`
		VersionCode   int    `json:"version_code"`
		GoMain        string `json:"go_main"`
		Icon          string `json:"icon,omitempty"`
		Generator     string `json:"generator"`
	}{
		SchemaVersion: 1,
		PackageID:     cfg.PackageID,
		AppName:       cfg.AppName,
		VersionName:   cfg.VersionName,
		VersionCode:   cfg.VersionCode,
		GoMain:        cfg.GoMain,
		Icon:          cfg.Icon,
		Generator:     "eletrocromo-macos",
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// ResolveGoMain returns an absolute directory containing the Go main package.
func ResolveGoMain(goMain, baseDir string) (string, error) {
	goMain = strings.TrimSpace(goMain)
	if goMain == "" {
		goMain = "."
	}
	if !filepath.IsAbs(goMain) {
		goMain = filepath.Join(baseDir, goMain)
	}
	abs, err := filepath.Abs(goMain)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("go_main %q: %w", abs, err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrGoMainNotDir, abs)
	}
	return abs, nil
}

// DefaultOutApp is dist/<app_name>.app under cwd.
func DefaultOutApp(appName, cwd string) string {
	name := strings.TrimSpace(appName)
	name = strings.TrimSuffix(name, ".app")
	if name == "" {
		name = "App"
	}
	return filepath.Join(cwd, "dist", name+".app")
}
