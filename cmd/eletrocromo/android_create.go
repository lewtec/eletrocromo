package main

import (
	"fmt"
	"path/filepath"

	"github.com/lewtec/eletrocromo/internal/apkgen"
	"github.com/spf13/cobra"
)

func newAndroidCreateCmd() *cobra.Command {
	var (
		id      string
		name    string
		out     string
		goMain  string
		version string
		code    int
		force   bool
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"init"},
		Short:   "Generate an Android WebView host project",
		Long: `Create an ad-hoc Android Gradle project for a reverse-domain package id
(App.ID / applicationId). The shell runs a multiarch Go binary and loads
the UI in system WebView.

Example:
  eletrocromo android create \
    --id br.tec.lew.eletrocromo.counter \
    --name Counter \
    --go-main ../../examples/counter \
    --out ./dist/android-counter`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			absOut, err := filepath.Abs(out)
			if err != nil {
				return err
			}
			if err := apkgen.Create(apkgen.Options{
				OutDir: absOut,
				Force:  force,
				Config: apkgen.Config{
					PackageID:   id,
					AppName:     name,
					VersionName: version,
					VersionCode: code,
					GoMain:      goMain,
				},
			}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created Android host project\n  package: %s\n  out:     %s\n", id, absOut)
			fmt.Fprintf(cmd.OutOrStdout(), "next:\n  1. set go_main in eletrocromo.json if needed\n  2. cd %s && ./scripts/build-go.sh\n  3. gradle wrapper && ./gradlew assembleDebug\n", absOut)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "reverse-domain package id / applicationId (required)")
	cmd.Flags().StringVar(&name, "name", "", "launcher label (default: last label of --id)")
	cmd.Flags().StringVar(&out, "out", "", "output project directory (required)")
	cmd.Flags().StringVar(&goMain, "go-main", ".", "Go main package directory (stored in eletrocromo.json)")
	cmd.Flags().StringVar(&version, "version", "0.1.0", "Android versionName")
	cmd.Flags().IntVar(&code, "code", 1, "Android versionCode")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite non-empty --out")
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
