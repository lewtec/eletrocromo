//go:build unix

package eletrocromo

import (
	"os/exec"
	"syscall"
)

// putInOwnProcessGroup makes the child the leader of a new process group so we
// can signal the whole Helium/Chromium tree (helpers + GPU process, etc.).
func putInOwnProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killProcessTree signals the process group (negative PID). Safe if already dead.
func killProcessTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	// Prefer graceful stop; Chromium often needs a hard kill for helpers.
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}
