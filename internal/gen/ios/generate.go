package ios

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/eletrocromo/internal/gen/common"
)

//go:embed all:template clangwrap.sh
var templateFS embed.FS

// Options controls Create.
type Options struct {
	OutDir string
	Force  bool
	Config Config
}

type templateData struct {
	Config
	Product            string
	ShareProduct       string
	AppNameXML         string
	VersionXML         string
	CodeString         string
	PlistURLTypes      string
	PlistDocumentTypes string
	WakeScheme         string
	AppGroupID         string
	EmbedShare         bool
	ShareActivationXML string
}

// Create writes an ephemeral XcodeGen iOS host under opts.OutDir.
func Create(opts Options) error {
	cfg, err := opts.Config.withDefaults()
	if err != nil {
		return err
	}
	out := strings.TrimSpace(opts.OutDir)
	if out == "" {
		return ErrOutDirRequired
	}
	out, err = filepath.Abs(out)
	if err != nil {
		return err
	}
	if err := common.PrepareOutDir(out, opts.Force); err != nil {
		return err
	}

	product := cfg.ProductName()
	data := templateData{
		Config:             cfg,
		Product:            product,
		ShareProduct:       product + "Share",
		AppNameXML:         common.XMLEscape(cfg.AppName),
		VersionXML:         common.XMLEscape(cfg.VersionName),
		CodeString:         fmt.Sprintf("%d", cfg.VersionCode),
		PlistURLTypes:      cfg.Capabilities.PlistURLTypes(cfg.PackageID),
		PlistDocumentTypes: cfg.Capabilities.PlistDocumentTypes(),
		WakeScheme:         cfg.Capabilities.WakeScheme(cfg.PackageID),
		AppGroupID:         common.AppGroupID(cfg.PackageID),
		EmbedShare:         cfg.Capabilities.Files != nil || cfg.Capabilities.Share != nil,
		ShareActivationXML: cfg.Capabilities.ShareActivationXML(),
	}
	if err := common.WalkTemplate(templateFS, data, out); err != nil {
		return err
	}
	raw, err := encodeConfigJSON(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "eletrocromo.json"), raw, 0o644)
}
