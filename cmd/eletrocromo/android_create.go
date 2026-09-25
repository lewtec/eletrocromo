package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lewtec/eletrocromo/internal/gen/apk"
	"github.com/lewtec/lewkit/x/cmd"
)

type androidCreateCmd struct {
	id      cmd.StringArg   `long:"id" help:"reverse-domain package id / applicationId (required)"`
	name    cmd.StringArg   `long:"name" default:"" help:"launcher label (default: last label of --id)"`
	out     cmd.StringArg   `long:"out" help:"output project directory (required)"`
	goMain  cmd.StringArg   `long:"go-main" default:"." help:"Go main package directory (stored in eletrocromo.json)"`
	version cmd.StringArg   `long:"version" default:"" help:"Android versionName (default: from VCS / -X)"`
	code    cmd.IntArg[int] `long:"code" default:"0" help:"Android versionCode (default: from version / git)"`
	force   cmd.Flag        `long:"force" help:"overwrite non-empty --out"`
}

func (androidCreateCmd) Description() string {
	return "Generate an Android WebView host project. Not the happy path; prefer \"GOOS=android eletrocromo build\"."
}

func (c *androidCreateCmd) Run(context.Context) error {
	absOut, err := filepath.Abs(c.out.Value())
	if err != nil {
		return err
	}
	cfg := apk.Config{
		PackageID:   c.id.Value(),
		AppName:     c.name.Value(),
		GoMain:      c.goMain.Value(),
		VersionName: c.version.Value(),
		VersionCode: c.code.Value(),
	}
	if err := apk.Create(apk.Options{
		OutDir: absOut,
		Force:  c.force.Value(),
		Config: cfg,
	}); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(os.Stdout, "created Android host project\n  package: %s\n  out:     %s\n", c.id.Value(), absOut); err != nil {
		return err
	}
	_, err = fmt.Fprintf(os.Stdout, "next:\n  1. set go_main in eletrocromo.json if needed\n  2. cd %s && ./scripts/build-go.sh\n  3. gradle wrapper && ./gradlew assembleDebug\n", absOut)
	return err
}
