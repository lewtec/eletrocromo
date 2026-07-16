//go:build darwin

package eletrocromo

import (
	"net/url"
	"os/exec"
)

func openSystemBrowser(u *url.URL) error {
	cmd := exec.Command("/usr/bin/open", u.String())
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
