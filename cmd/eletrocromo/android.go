package main

import (
	"github.com/spf13/cobra"
)

func newAndroidCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "android",
		Short: "Android WebView host / APK packaging",
		Long:  "Build or scaffold Android projects that wrap a Go eletrocromo app in WebView (PhoneGap/Expo-style).",
	}
	cmd.AddCommand(newAndroidCreateCmd())
	cmd.AddCommand(newAndroidBuildCmd())
	return cmd
}
