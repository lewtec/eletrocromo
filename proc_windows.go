//go:build windows

package eletrocromo

import "os/exec"

func putInOwnProcessGroup(cmd *exec.Cmd) {
	// Windows job objects are the proper analogue; out of scope for Linux-first.
}

func killProcessTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
