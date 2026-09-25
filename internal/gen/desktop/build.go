// Package desktop cross-compiles an eletrocromo app for Linux or Windows.
// The icon is attached without writing into the app module: a Go -overlay
// supplies the Windows .syso or the Linux embedded PNG.
package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lewtec/eletrocromo/internal/gen/goenv"
	"github.com/lewtec/eletrocromo/internal/winres"
)

const (
	linuxIconName = "eletrocromo_icon.png"
	linuxIconGo   = "eletrocromo_icon.go"
)

// ErrOS is returned when GOOS is not linux or windows.
var ErrOS = errors.New("desktop build supports linux and windows")

// Options is one cross-compile.
type Options struct {
	GOOS      string
	GOARCH    string
	ModuleDir string
	Out       string
	IconICO   string // windows; empty skips the PE icon
	IconPNG   string // linux; empty skips the embedded PNG
	Stdout    io.Writer
	Stderr    io.Writer
}

// Build runs go build for Options.GOOS. CGO is off.
func Build(ctx context.Context, opts Options) error {
	goos := strings.TrimSpace(opts.GOOS)
	if goos != "linux" && goos != "windows" {
		return fmt.Errorf("%w: %s", ErrOS, goos)
	}
	arch := strings.TrimSpace(opts.GOARCH)
	if arch == "" {
		arch = runtime.GOARCH
	}
	mod := strings.TrimSpace(opts.ModuleDir)
	out := strings.TrimSpace(opts.Out)
	if mod == "" || out == "" {
		return errors.New("module dir and output path are required")
	}
	var err error
	if mod, err = filepath.Abs(mod); err != nil {
		return err
	}
	if out, err = filepath.Abs(out); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	stdout, stderr := opts.Stdout, opts.Stderr
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	work, err := os.MkdirTemp("", "eletrocromo-desktop-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(work) }()

	overlay := map[string]string{}
	switch goos {
	case "windows":
		// The linker opens .syso files by path, so the object has to sit in
		// the module for the duration of go build. It is removed afterwards.
		if ico := strings.TrimSpace(opts.IconICO); ico != "" {
			syso := filepath.Join(mod, "rsrc_windows_"+arch+".syso")
			if _, statErr := os.Stat(syso); statErr != nil {
				if err := winres.Syso(syso, arch, ico); err != nil {
					return err
				}
				defer func() { _ = os.Remove(syso) }()
			}
		}
	default:
		if png := strings.TrimSpace(opts.IconPNG); png != "" {
			pkg, err := goPackage(mod)
			if err != nil {
				return err
			}
			dst := filepath.Join(work, linuxIconName)
			if err := copyFile(png, dst); err != nil {
				return err
			}
			src := filepath.Join(work, linuxIconGo)
			body := fmt.Sprintf("package %s\n\nimport _ \"embed\"\n\n//go:embed %s\nvar eletrocromoIconPNG []byte\n\nvar _ = eletrocromoIconPNG\n", pkg, linuxIconName)
			if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
				return err
			}
			overlay[filepath.Join(mod, linuxIconGo)] = src
			overlay[filepath.Join(mod, linuxIconName)] = dst
		}
	}

	args := []string{"build", "-o", out}
	if len(overlay) > 0 {
		raw, err := json.Marshal(map[string]any{"Replace": overlay})
		if err != nil {
			return err
		}
		overlayPath := filepath.Join(work, "overlay.json")
		if err := os.WriteFile(overlayPath, raw, 0o644); err != nil {
			return err
		}
		args = append(args, "-overlay", overlayPath)
	}
	args = append(args, ".")
	if _, err := fmt.Fprintf(stdout, "eletrocromo: go build GOOS=%s GOARCH=%s\n", goos, arch); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = mod
	cmd.Env = goenv.Merge(os.Environ(), "GOOS="+goos, "GOARCH="+arch)
	var errBuf bytes.Buffer
	cmd.Stdout = stdout
	cmd.Stderr = io.MultiWriter(stderr, &errBuf)
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			return fmt.Errorf("go build %s/%s: %w", goos, arch, err)
		}
		return fmt.Errorf("go build %s/%s: %w\n%s", goos, arch, err, msg)
	}
	_, err = fmt.Fprintf(stdout, "eletrocromo: %s\n", out)
	return err
}

func goPackage(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return "", err
		}
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "package ") {
				return strings.Fields(line)[1], nil
			}
		}
	}
	return "", fmt.Errorf("no go package in %s", dir)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
