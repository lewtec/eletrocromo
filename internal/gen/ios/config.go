// Package ios generates an ephemeral iOS WKWebView host and packages
// a GOOS=ios c-archive into a Debug .app.
//
// iOS cannot exec a helper binary. The Go app is linked in-process
// (EletrocromoStart) and the host waits on ELETROCROMO_READY_FILE.
// The importable eletrocromo library stays free of Xcode.
package ios

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

// ArchiveName is the c-archive stem written under lib/.
const ArchiveName = "libeletrocromo"

// BridgeGoName is overlaid into the app main package during the archive build.
const BridgeGoName = "eletrocromo_ios_bridge.go"

// Config is the iOS host identity. It is a named common.HostConfig so this
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
	return common.EncodeHostJSON(common.HostConfig(cfg), "eletrocromo-ios")
}

// ResolveGoMain returns an absolute directory containing the Go main package.
func ResolveGoMain(goMain, baseDir string) (string, error) {
	return common.ResolveGoMain(goMain, baseDir)
}

// DefaultOutApp is dist/<app_name>.app under cwd.
func DefaultOutApp(appName, cwd string) string {
	return common.DefaultOutApp(appName, cwd)
}
