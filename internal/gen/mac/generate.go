package mac

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/eletrocromo/internal/gen/common"
)

//go:embed all:template
var templateFS embed.FS

// Options controls Create.
type Options struct {
	OutDir string
	Force  bool
	Config Config
}

type templateData struct {
	Config
	Product    string
	AppNameXML string
	VersionXML string
	CodeString string
}

// Create writes an ephemeral XcodeGen host under opts.OutDir.
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

	data := templateData{
		Config:     cfg,
		Product:    cfg.ProductName(),
		AppNameXML: common.XMLEscape(cfg.AppName),
		VersionXML: common.XMLEscape(cfg.VersionName),
		CodeString: fmt.Sprintf("%d", cfg.VersionCode),
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
