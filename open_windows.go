//go:build windows

package eletrocromo

import (
	"net/url"
	"os/exec"
)

func openSystemBrowser(u *url.URL) error {
	// cmd /c start "" <url> — empty title argument so URLs with & are safe.
	cmd := exec.Command("cmd", "/c", "start", "", u.String())
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
