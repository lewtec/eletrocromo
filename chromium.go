package eletrocromo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// heliumCandidates are local Helium binary names (workspaced registry bin: helium).
var heliumCandidates = []string{
	"helium",
}

// ErrNoChromium is returned when Helium cannot be resolved (local PATH or
// workspaced ensure of registry helium-browser).
var ErrNoChromium = errors.New("no Helium browser host found")

// ErrHeliumLaunch is returned when the Helium process fails to stay up after Start.
var ErrHeliumLaunch = errors.New("helium failed to launch")

// heliumStartupGrace is how long we wait for the process to prove it stays up.
// Immediate crash (bad flags, missing libs, wrapper exit) surfaces as Run error.
var heliumStartupGrace = 2 * time.Second

// lookPath is exec.LookPath; tests may override.
var lookPath = exec.LookPath

// GetChromium returns a local Helium binary path if present on PATH.
// It does not download or call workspaced — see ResolveBrowserHost.
// Name kept for compatibility; only Helium is supported as the app window host.
func GetChromium() (string, error) {
	for _, ch := range heliumCandidates {
		path, err := lookPath(ch)
		if errors.Is(err, exec.ErrNotFound) {
			continue
		}
		if err != nil {
			continue
		}
		return path, nil
	}
	return "", ErrNoChromium
}

// resolveBrowserHost is the host-resolve implementation; tests may override.
var resolveBrowserHost = ResolveBrowserHost

// ResolveBrowserHost finds Helium for --app launch.
//
// Order (SPEC): local Helium → ensure via workspaced (tool which helium-browser
// helium, bootstrapping workspaced if needed) → error.
// Never opens Chrome/Edge/system browser.
//
// Set ELETROCROMO_NO_ENSURE=1 to skip network ensure (tests/CI).
func ResolveBrowserHost(ctx context.Context) (string, error) {
	if path, err := GetChromium(); err == nil {
		return path, nil
	}
	if ensureDisabled() {
		return "", fmt.Errorf("%w: install Helium, or allow ensure (unset ELETROCROMO_NO_ENSURE)", ErrNoChromium)
	}
	path, err := ensureHeliumBrowser(ctx)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrNoChromium, err)
	}
	return path, nil
}

func ensureDisabled() bool {
	v := strings.TrimSpace(os.Getenv("ELETROCROMO_NO_ENSURE"))
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

// appWindow is a started Helium process with a single Wait owner.
type appWindow struct {
	cmd    *exec.Cmd
	stderr bytes.Buffer
	stderrMu sync.Mutex
	waitc  chan error // holds Wait result once
}

// startAppWindow starts Helium with an isolated user-data-dir and --app URL.
// On Unix the child is put in its own process group so stop() can kill the tree.
func startAppWindow(bin, rawURL, userDataDir string) (*appWindow, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("invalid URL scheme: %s", u.Scheme)
	}
	if userDataDir == "" {
		return nil, fmt.Errorf("user data dir is required")
	}
	w := &appWindow{waitc: make(chan error, 1)}
	// Chromium-family app window + dedicated profile so apps do not share
	// cookies/sessions or steal each other's windows.
	w.cmd = exec.Command(bin,
		"--user-data-dir="+userDataDir,
		"--no-first-run",
		"--no-default-browser-check",
		"--app="+u.String(),
	)
	putInOwnProcessGroup(w.cmd)
	w.cmd.Stderr = &lockedWriter{mu: &w.stderrMu, w: &w.stderr}
	// Drop stdout noise from Chromium; keep stderr for launch diagnostics.
	w.cmd.Stdout = nil
	if err := w.cmd.Start(); err != nil {
		return nil, err
	}
	go func() {
		w.waitc <- w.cmd.Wait()
	}()
	return w, nil
}

type lockedWriter struct {
	mu *sync.Mutex
	w  *bytes.Buffer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// awaitStartup returns an error if Helium exits within grace (failed launch).
// If still running after grace, returns nil; call watchExit to reap later.
func (w *appWindow) awaitStartup(grace time.Duration) error {
	if grace <= 0 {
		grace = heliumStartupGrace
	}
	timer := time.NewTimer(grace)
	defer timer.Stop()
	select {
	case err := <-w.waitc:
		return wrapHeliumExit(err, w.stderrSnapshot())
	case <-timer.C:
		return nil
	}
}

// watchExit invokes onExit once when the process exits (successful or not).
// Must only be called after awaitStartup returned nil (process still running).
func (w *appWindow) watchExit(onExit func(error)) {
	go func() {
		err := <-w.waitc
		if onExit != nil {
			onExit(err)
		}
	}()
}

// stop kills the Helium process tree (process group on Unix) if still running.
func (w *appWindow) stop() {
	if w == nil {
		return
	}
	killProcessTree(w.cmd)
}

func (w *appWindow) stderrSnapshot() string {
	w.stderrMu.Lock()
	defer w.stderrMu.Unlock()
	return w.stderr.String()
}

func wrapHeliumExit(err error, stderr string) error {
	msg := strings.TrimSpace(stderr)
	if len(msg) > 512 {
		msg = "…" + msg[len(msg)-512:]
	}
	if err == nil {
		if msg != "" {
			return fmt.Errorf("%w: process exited immediately (%s)", ErrHeliumLaunch, msg)
		}
		return fmt.Errorf("%w: process exited immediately", ErrHeliumLaunch)
	}
	if msg != "" {
		return fmt.Errorf("%w: %w (%s)", ErrHeliumLaunch, err, msg)
	}
	return fmt.Errorf("%w: %w", ErrHeliumLaunch, err)
}

// LaunchChromium resolves Helium then opens the URL in app mode (--app).
// Uses a temporary profile under the OS temp dir when no App.ID is available;
// prefer App.Run which uses reverse-domain profile isolation.
func LaunchChromium(ctx context.Context, u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: %s", u.Scheme)
	}
	bin, err := resolveBrowserHost(ctx)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "eletrocromo-launch-*")
	if err != nil {
		return err
	}
	w, err := startAppWindow(bin, u.String(), dir)
	if err != nil {
		return err
	}
	if err := w.awaitStartup(heliumStartupGrace); err != nil {
		w.stop()
		return err
	}
	// Detached for this helper: reap when process dies, ignore result.
	w.watchExit(func(error) {})
	return nil
}
