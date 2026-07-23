package main

import (
	"fmt"

	"github.com/lewtec/eletrocromo/internal/version"
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	info := version.Resolve()
	cmd := &cobra.Command{
		Use:           "eletrocromo",
		Short:         "Tooling for eletrocromo apps (Android host generator, …)",
		Long:          "CLI for packaging and scaffolding eletrocromo apps. The runtime library is imported as github.com/lewtec/eletrocromo.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       info.String(),
	}
	cmd.SetVersionTemplate(fmt.Sprintf("%s\n", "{{.Version}}"))
	cmd.AddCommand(newAndroidCmd())
	cmd.AddCommand(newVersionCmd())
	return cmd
}
