//go:build !android

package eletrocromo

// configureDNSForPlatform is a no-op on non-Android hosts.
func configureDNSForPlatform() {}
