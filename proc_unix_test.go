//go:build unix

package eletrocromo

import (
	"bufio"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// TestMain: when re-exec'd as a stubborn child, swallow SIGTERM and hang.
// Only SIGKILL (from killProcessTree after grace) should end this process.
func TestMain(m *testing.M) {
	if os.Getenv("ELETROCROMO_STUBBORN_CHILD") == "1" {
		// Notify+drain is more reliable than signal.Ignore across Go versions.
		ch := make(chan os.Signal, 8)
		signal.Notify(ch, syscall.SIGTERM)
		go func() {
			for range ch {
			}
		}()
		_, _ = os.Stdout.WriteString("ready\n")
		select {}
	}
	os.Exit(m.Run())
}

func TestKillProcessTree_GracefulOnSIGTERM(t *testing.T) {
	// Child exits 0 on SIGTERM (cooperative Helium/wrapper path).
	cmd := exec.Command("sh", "-c", `trap 'exit 0' TERM; while true; do sleep 0.05; done`)
	putInOwnProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	prev := heliumKillGrace
	heliumKillGrace = 2 * time.Second
	t.Cleanup(func() { heliumKillGrace = prev })

	start := time.Now()
	killProcessTree(cmd)
	select {
	case err := <-done:
		// Exited via SIGTERM path (exit 0 from trap) or as signalled TERM — not KILL.
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				if status, ok := ee.Sys().(syscall.WaitStatus); ok {
					if status.Signaled() && status.Signal() == syscall.SIGKILL {
						t.Fatalf("got SIGKILL; want cooperative SIGTERM exit: %v", err)
					}
				}
			}
		}
	case <-time.After(3 * time.Second):
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		t.Fatal("process did not exit after killProcessTree")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("cooperative kill took too long: %v (should return soon after SIGTERM)", elapsed)
	}
}

func TestKillProcessTree_SIGKILLAfterGrace(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "ELETROCROMO_STUBBORN_CHILD=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	putInOwnProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	// Wait until SIGTERM is swallowed (child prints ready).
	readyCh := make(chan error, 1)
	go func() {
		br := bufio.NewReader(stdout)
		line, err := br.ReadString('\n')
		if err != nil {
			readyCh <- err
			return
		}
		if line != "ready\n" {
			readyCh <- errString("unexpected ready line: " + line)
			return
		}
		readyCh <- nil
	}()
	select {
	case err := <-readyCh:
		if err != nil {
			t.Fatalf("child ready: %v", err)
		}
	case err := <-done:
		t.Fatalf("child exited before ready: %v", err)
	case <-time.After(5 * time.Second):
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		t.Fatal("timeout waiting for child ready")
	}

	prev := heliumKillGrace
	heliumKillGrace = 100 * time.Millisecond
	t.Cleanup(func() { heliumKillGrace = prev })

	// Confirm SIGTERM alone does not kill it.
	if err := syscall.Kill(cmd.Process.Pid, syscall.SIGTERM); err != nil {
		t.Fatalf("SIGTERM probe: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := syscall.Kill(cmd.Process.Pid, 0); err != nil {
		t.Fatalf("child died on SIGTERM despite handler: %v", err)
	}

	start := time.Now()
	killProcessTree(cmd)
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected non-nil Wait after SIGKILL")
		}
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("want ExitError, got %T %v", err, err)
		}
		status, ok := ee.Sys().(syscall.WaitStatus)
		if !ok {
			t.Fatalf("want WaitStatus, got %T", ee.Sys())
		}
		if !status.Signaled() || status.Signal() != syscall.SIGKILL {
			t.Fatalf("want SIGKILL, got signaled=%v signal=%v err=%v", status.Signaled(), status.Signal(), err)
		}
	case <-time.After(3 * time.Second):
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		t.Fatal("stubborn process did not exit after grace+SIGKILL")
	}
	if elapsed := time.Since(start); elapsed < 80*time.Millisecond {
		t.Fatalf("returned before grace elapsed: %v", elapsed)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func TestKillProcessTree_NilSafe(t *testing.T) {
	killProcessTree(nil)
	killProcessTree(&exec.Cmd{})
}
