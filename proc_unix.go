//go:build unix

package eletrocromo

import (
	"os/exec"
	"syscall"
	"time"
)

// heliumKillGrace is how long killProcessTree waits after SIGTERM before SIGKILL.
// Chromium helpers often need the hard kill; the grace lets a clean exit win first.
// Tests may shrink this to keep the suite fast.
var heliumKillGrace = 500 * time.Millisecond

// putInOwnProcessGroup makes the child the leader of a new process group so we
// can signal the whole Helium/Chromium tree (helpers + GPU process, etc.).
func putInOwnProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killProcessTree signals the process group (negative PID). Safe if already dead.
// Sends SIGTERM first, then SIGKILL only if the main process is still alive after
// heliumKillGrace — so a cooperative exit is not immediately hard-killed.
func killProcessTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	// Prefer graceful stop; Chromium often needs a hard kill for helpers.
	_ = syscall.Kill(-pid, syscall.SIGTERM)

	deadline := time.Now().Add(heliumKillGrace)
	for {
		// Check the process we started (not the whole group): helpers may
		// churn while the leader is still up, and group probe is racy.
		if err := syscall.Kill(pid, 0); err != nil {
			return
		}
		if !time.Now().Before(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}
