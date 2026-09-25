package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lewtec/eletrocromo/internal/gen/apk"
	"github.com/lewtec/eletrocromo/internal/gen/desktop"
	"github.com/lewtec/lewkit/x/cmd"
)

type linuxCmd struct{ desktopCmd }

func (linuxCmd) Description() string {
	return "Cross-compile a Linux binary (CGO off). The icon defaults to the built-in atom mark and is embedded in the ELF."
}

func (c *linuxCmd) Run(ctx context.Context) error {
	return c.build(ctx, "linux")
}

type windowsCmd struct{ desktopCmd }

func (windowsCmd) Description() string {
	return "Cross-compile a Windows exe (CGO off). The icon defaults to the built-in atom mark and is stored in the PE resources."
}

func (c *windowsCmd) Run(ctx context.Context) error {
	return c.build(ctx, "windows")
}

type desktopCmd struct {
	hostFlags
	arch cmd.StringArg `long:"arch" default:"" help:"GOARCH (default: this machine)"`
}

func (c *desktopCmd) build(ctx context.Context, goos string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	cfg, baseDir, err := c.load(cwd)
	if err != nil {
		return err
	}
	mod, err := apk.ResolveGoMain(cfg.GoMain, baseDir)
	if err != nil {
		return err
	}
	arch := strings.TrimSpace(c.arch.Value())
	if arch == "" {
		arch = runtime.GOARCH
	}
	iconSrc := resolveIconSource(cwd, c.icon.Value(), baseDir, cfg.Icon)
	iconOut := resolveIconOutput(cwd, c.output.Value())
	var ico, png string
	err = runIconsThen(ctx, iconThen{src: iconSrc, out: iconOut, refresh: c.refresh.Value(), name: goos}, func(iconRoot string) error {
		ico = filepath.Join(iconRoot, "windows", "icon.ico")
		png = filepath.Join(iconRoot, "linux", "icon-256.png")
		if _, err := os.Stat(ico); err != nil {
			ico = ""
		}
		if _, err := os.Stat(png); err != nil {
			png = ""
		}
		out := strings.TrimSpace(c.out.Value())
		if out == "" {
			slug := filepath.Base(baseDir)
			if slug == "." || slug == string(filepath.Separator) {
				slug = "app"
			}
			out = filepath.Join(cwd, "dist", slug+"-"+goos+"-"+arch)
			if goos == "windows" {
				out += ".exe"
			}
		}
		return desktop.Build(ctx, desktop.Options{
			GOOS:      goos,
			GOARCH:    arch,
			ModuleDir: mod,
			Out:       out,
			IconICO:   ico,
			IconPNG:   png,
			Stdout:    os.Stdout,
			Stderr:    os.Stderr,
		})
	})
	return err
}
