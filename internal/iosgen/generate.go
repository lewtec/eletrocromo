package iosgen

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
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
	Product    string
	AppNameXML string
	VersionXML string
	CodeString string
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
	if err := prepareOutDir(out, opts.Force); err != nil {
		return err
	}

	data := templateData{
		Config:     cfg,
		Product:    cfg.ProductName(),
		AppNameXML: xmlEscape(cfg.AppName),
		VersionXML: xmlEscape(cfg.VersionName),
		CodeString: fmt.Sprintf("%d", cfg.VersionCode),
	}
	if err := walkTemplate(data, out); err != nil {
		return err
	}
	raw, err := encodeConfigJSON(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "eletrocromo.json"), raw, 0o644)
}

func prepareOutDir(out string, force bool) error {
	st, err := os.Stat(out)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return os.MkdirAll(out, 0o755)
		}
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("%w: %s", ErrOutPathNotDir, out)
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	if !force {
		return fmt.Errorf("%w: %s", ErrOutDirNotEmpty, out)
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(out, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func walkTemplate(data templateData, out string) error {
	return fs.WalkDir(templateFS, "template", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel("template", p)
		if err != nil {
			return err
		}
		rel = filepath.FromSlash(rel)
		raw, err := templateFS.ReadFile(p)
		if err != nil {
			return err
		}
		destRel, body, err := renderFile(rel, raw, data)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		dest := filepath.Join(out, destRel)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dest, body, 0o644)
	})
}

func renderFile(rel string, raw []byte, data templateData) (destRel string, body []byte, err error) {
	destRel = rel
	if !strings.HasSuffix(rel, ".tmpl") {
		return destRel, raw, nil
	}
	destRel = strings.TrimSuffix(rel, ".tmpl")
	name := path.Base(filepath.ToSlash(rel))
	tmpl, err := template.New(name).Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return "", nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", nil, err
	}
	return destRel, buf.Bytes(), nil
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
