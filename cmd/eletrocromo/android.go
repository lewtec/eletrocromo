package main

import (
	"github.com/spf13/cobra"
)

func newAndroidCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "android",
		Short: "Android WebView host (APK packaging scaffold)",
		Long:  "Generate and manage ad-hoc Android projects that wrap a Go eletrocromo app in WebView (PhoneGap/Expo-style).",
	}
	cmd.AddCommand(newAndroidCreateCmd())
	return cmd
}
