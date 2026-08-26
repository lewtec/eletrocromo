// Package mac generates an ephemeral macOS WKWebView host and packages
// a CGo-less darwin Go binary into an unsigned Debug .app.
//
// The importable eletrocromo library stays free of Xcode; this package only
// writes an XcodeGen tree that runs the Go server and opens WKWebView.
package mac

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lewtec/eletrocromo"
	"github.com/lewtec/eletrocromo/internal/gen/common"
	"github.com/lewtec/eletrocromo/internal/version"
)

// Config / path sentinels.
var (
	ErrConfigPathEmpty = common.ErrConfigPathEmpty
	ErrGoMainNotDir    = common.ErrGoMainNotDir
	ErrOutDirRequired  = common.ErrOutDirRequired
	ErrOutPathNotDir   = common.ErrOutPathNotDir
	ErrOutDirNotEmpty  = common.ErrOutDirNotEmpty
)

// HelperName is the Go child binary inside Contents/MacOS.
const HelperName = "eletrocromo-server"

// Config is the project identity written into the generated tree.
type Config struct {
	PackageID    string              `json:"package_id"`
	AppName      string              `json:"app_name"`
	VersionName  string              `json:"version_name"`
	VersionCode  int                 `json:"version_code"`
	GoMain       string              `json:"go_main"`
	Icon         string              `json:"icon,omitempty"`
	Capabilities common.Capabilities `json:"capabilities,omitempty"`
}

// ProductName is a filesystem-safe Xcode PRODUCT_NAME / .app stem.
func (c Config) ProductName() string {
	return common.ProductName(c.PackageID, c.AppName)
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
	if err := c.Capabilities.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func encodeConfigJSON(cfg Config) ([]byte, error) {
	doc := struct {
		SchemaVersion int                 `json:"schema_version"`
		PackageID     string              `json:"package_id"`
		AppName       string              `json:"app_name"`
		VersionName   string              `json:"version_name"`
		VersionCode   int                 `json:"version_code"`
		GoMain        string              `json:"go_main"`
		Icon          string              `json:"icon,omitempty"`
		Generator     string              `json:"generator"`
		Capabilities  common.Capabilities `json:"capabilities,omitempty"`
	}{
		SchemaVersion: 1,
		PackageID:     cfg.PackageID,
		AppName:       cfg.AppName,
		VersionName:   cfg.VersionName,
		VersionCode:   cfg.VersionCode,
		GoMain:        cfg.GoMain,
		Icon:          cfg.Icon,
		Generator:     "eletrocromo-macos",
		Capabilities:  cfg.Capabilities,
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// ResolveGoMain returns an absolute directory containing the Go main package.
func ResolveGoMain(goMain, baseDir string) (string, error) {
	return common.ResolveGoMain(goMain, baseDir)
}

// DefaultOutApp is dist/<app_name>.app under cwd.
func DefaultOutApp(appName, cwd string) string {
	return common.DefaultOutApp(appName, cwd)
}
