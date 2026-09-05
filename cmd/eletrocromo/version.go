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
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), version.Resolve().String())
			return err
		},
	}
}
