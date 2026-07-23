// Package apkgen generates ad-hoc Android WebView host projects from an
// embedded template (PhoneGap/Expo-style), keyed by reverse-domain package ID.
//
// The core eletrocromo library stays free of the Android SDK; this package only
// writes a Gradle/Kotlin tree that runs a multiarch Go binary and opens WebView.
package apkgen

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/lewtec/eletrocromo"
)

//go:embed all:template
var templateFS embed.FS

// Config is the project identity written into the generated tree and
// eletrocromo.json (re-run / rebuild input).
type Config struct {
	// PackageID is the Android applicationId and Kotlin package (App.ID).
	PackageID string `json:"package_id"`
	// AppName is the launcher label.
	AppName string `json:"app_name"`
	// VersionName is Android versionName (default "0.1.0").
	VersionName string `json:"version_name"`
	// VersionCode is Android versionCode (default 1).
	VersionCode int `json:"version_code"`
	// GoMain is a path (often relative to the generated project) to the Go
	// main package directory used by scripts/build-go.sh.
	GoMain string `json:"go_main"`
}

// templateData is passed to text/template for file bodies.
type templateData struct {
	Config
	// RootProjectName is a filesystem-safe Gradle rootProject.name.
	RootProjectName string
	// PackagePath is PackageID with dots → slashes (Kotlin source dir).
	PackagePath string
}

// Options controls Create.
type Options struct {
	// OutDir is the destination project directory (created if missing).
	OutDir string
	// Force overwrites an existing non-empty OutDir.
	Force bool
	// Config is required (PackageID + AppName at minimum after defaults).
	Config Config
}

// Create materializes an Android host project under opts.OutDir.
func Create(opts Options) error {
	cfg, err := normalizeConfig(opts.Config)
	if err != nil {
		return err
	}
	out := strings.TrimSpace(opts.OutDir)
	if out == "" {
		return fmt.Errorf("out dir is required")
	}
	out, err = filepath.Abs(out)
	if err != nil {
		return err
	}
	if err := prepareOutDir(out, opts.Force); err != nil {
		return err
	}

	data := templateData{
		Config:          cfg,
		RootProjectName: rootProjectName(cfg.PackageID, cfg.AppName),
		PackagePath:     strings.ReplaceAll(cfg.PackageID, ".", "/"),
	}

	if err := walkTemplate(data, out); err != nil {
		return err
	}
	return writeConfigJSON(out, cfg)
}

func normalizeConfig(cfg Config) (Config, error) {
	cfg.PackageID = strings.TrimSpace(cfg.PackageID)
	if err := eletrocromo.ValidateAppID(cfg.PackageID); err != nil {
		return Config{}, fmt.Errorf("package id: %w", err)
	}
	cfg.AppName = strings.TrimSpace(cfg.AppName)
	if cfg.AppName == "" {
		// Last label of reverse-domain (br.tec.lew.counter → counter).
		parts := strings.Split(cfg.PackageID, ".")
		cfg.AppName = parts[len(parts)-1]
	}
	if cfg.VersionName == "" {
		cfg.VersionName = "0.1.0"
	}
	if cfg.VersionCode <= 0 {
		cfg.VersionCode = 1
	}
	if strings.TrimSpace(cfg.GoMain) == "" {
		cfg.GoMain = "."
	}
	return cfg, nil
}

func prepareOutDir(out string, force bool) error {
	st, err := os.Stat(out)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(out, 0o755)
		}
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("out path exists and is not a directory: %s", out)
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	if !force {
		return fmt.Errorf("out dir is not empty (use --force): %s", out)
	}
	// Wipe contents but keep the directory node.
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(out, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func rootProjectName(packageID, appName string) string {
	// Prefer app name when it is a simple identifier; else last package label.
	name := strings.TrimSpace(appName)
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		case r == ' ':
			return '-'
		default:
			return -1
		}
	}, name)
	if name == "" {
		parts := strings.Split(packageID, ".")
		name = parts[len(parts)-1]
	}
	return name
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
		// Use slash paths from embed; convert for OS.
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
		mode := fs.FileMode(0o644)
		if strings.HasSuffix(dest, ".sh") {
			mode = 0o755
		}
		return os.WriteFile(dest, body, mode)
	})
}

func renderFile(rel string, raw []byte, data templateData) (destRel string, body []byte, err error) {
	destRel = rel
	// Kotlin sources: path under package ID + strip .tmpl
	if strings.HasPrefix(filepath.ToSlash(rel), "app/src/main/kotlin/") && strings.HasSuffix(rel, ".tmpl") {
		base := strings.TrimSuffix(filepath.Base(rel), ".tmpl")
		destRel = filepath.Join("app", "src", "main", "java", data.PackagePath, base)
	} else if strings.HasSuffix(rel, ".tmpl") {
		destRel = strings.TrimSuffix(rel, ".tmpl")
	}

	// Binary-ish or pure static: still run through template if markers present.
	// Always use text/template for consistency (templates are UTF-8 text).
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

func writeConfigJSON(out string, cfg Config) error {
	// Re-encode for stable key order / pretty print (struct tags).
	type fileCfg struct {
		SchemaVersion int      `json:"schema_version"`
		PackageID     string   `json:"package_id"`
		AppName       string   `json:"app_name"`
		VersionName   string   `json:"version_name"`
		VersionCode   int      `json:"version_code"`
		GoMain        string   `json:"go_main"`
		Generator     string   `json:"generator"`
		ABIs          []string `json:"abis"`
	}
	doc := fileCfg{
		SchemaVersion: 1,
		PackageID:     cfg.PackageID,
		AppName:       cfg.AppName,
		VersionName:   cfg.VersionName,
		VersionCode:   cfg.VersionCode,
		GoMain:        cfg.GoMain,
		Generator:     "eletrocromo android create",
		ABIs:          []string{"arm64-v8a", "armeabi-v7a", "x86_64"},
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	// Template already wrote eletrocromo.json; overwrite with canonical JSON.
	return os.WriteFile(filepath.Join(out, "eletrocromo.json"), raw, 0o644)
}
