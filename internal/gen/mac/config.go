// Package mac generates an ephemeral macOS WKWebView host and packages
// a CGo-less darwin Go binary into an unsigned Debug .app.
//
// The importable eletrocromo library stays free of Xcode; this package only
// writes an XcodeGen tree that runs the Go server and opens WKWebView.
package mac

import (
	"github.com/lewtec/eletrocromo/internal/gen/common"
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

// Config is the macOS host identity. It is a named common.HostConfig so this
// package can attach methods; on-disk JSON still goes through EncodeHostJSON.
type Config common.HostConfig

// ProductName is a filesystem-safe Xcode PRODUCT_NAME / .app stem.
func (c Config) ProductName() string {
	return common.ProductName(c.PackageID, c.AppName)
}

func (c Config) withDefaults() (Config, error) {
	id, err := common.ApplyHostDefaults(common.HostConfig(c))
	if err != nil {
		return Config{}, err
	}
	return Config(id), nil
}

func encodeConfigJSON(cfg Config) ([]byte, error) {
	return common.EncodeHostJSON(common.HostConfig(cfg), "eletrocromo-macos")
}

// ResolveGoMain returns an absolute directory containing the Go main package.
func ResolveGoMain(goMain, baseDir string) (string, error) {
	return common.ResolveGoMain(goMain, baseDir)
}

// DefaultOutApp is dist/<app_name>.app under cwd.
func DefaultOutApp(appName, cwd string) string {
	return common.DefaultOutApp(appName, cwd)
}
