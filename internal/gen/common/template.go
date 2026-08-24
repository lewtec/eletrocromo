package common

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

// WalkTemplate copies src's template/ tree into out, rendering *.tmpl files.
func WalkTemplate(src fs.FS, data any, out string) error {
	return fs.WalkDir(src, "template", func(p string, d fs.DirEntry, err error) error {
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
		raw, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		destRel, body, err := RenderFile(rel, raw, data)
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

// RenderFile copies a static file or executes a *.tmpl body.
func RenderFile(rel string, raw []byte, data any) (destRel string, body []byte, err error) {
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

// XMLEscape encodes characters that would break plist / XML text nodes.
func XMLEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
