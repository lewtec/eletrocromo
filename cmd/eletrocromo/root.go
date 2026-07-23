package main

import (
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "eletrocromo",
		Short:         "Tooling for eletrocromo apps (Android host generator, …)",
		Long:          "CLI for packaging and scaffolding eletrocromo apps. The runtime library is imported as github.com/lewtec/eletrocromo.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(newAndroidCmd())
	return cmd
}
