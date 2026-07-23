package main

import (
	"fmt"

	"github.com/lewtec/eletrocromo/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build version (goreleaser -X / VCS / git)",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), version.Resolve().String())
		},
	}
}
