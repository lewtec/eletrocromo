package eletrocromo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Registry tool / binary names for Helium (workspaced catalog).
const (
	heliumBrowserTool = "helium-browser"
	heliumBrowserBin  = "helium"
)

// ErrEnsureHeliumEmptyPath is returned when workspaced tool which succeeds but
// prints no binary path (callers can use errors.Is).
var ErrEnsureHeliumEmptyPath = errors.New("workspaced tool which returned empty path")

// workspacedPathOverride is set by ELETROCROMO_WORKSPACED when non-empty.
func workspacedPathOverride() string {
	return strings.TrimSpace(os.Getenv("ELETROCROMO_WORKSPACED"))
}

// commandOutput runs name with args and returns stdout. Tests may override.
// On failure the returned error wraps the underlying Run error (via %w) so
// callers can use errors.As for *exec.ExitError. When the context is already
// done (cancel/deadline), that error is joined in as well: CommandContext often
// surfaces a killed child as "signal: killed" rather than ctx.Err(), which used
// to make errors.Is(err, context.Canceled) fail for ResolveBrowserHost/App.Run.
// Stderr text is included when present.
var commandOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			err = errors.Join(err, ctxErr)
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return stdout.Bytes(), fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
		}
		return stdout.Bytes(), fmt.Errorf("%s %s: %s: %w", name, strings.Join(args, " "), msg, err)
	}
	return stdout.Bytes(), nil
}

// ensureHeliumBrowser returns the path to a helium binary, installing via
// workspaced tool which helium-browser helium when needed.
func ensureHeliumBrowser(ctx context.Context) (string, error) {
	ws, err := resolveWorkspaced(ctx)
	if err != nil {
		return "", err
	}
	out, err := commandOutput(ctx, ws, "tool", "which", heliumBrowserTool, heliumBrowserBin)
	if err != nil {
		return "", fmt.Errorf("workspaced ensure %s: %w", heliumBrowserTool, err)
	}
	path := strings.TrimSpace(string(out))
	// tool which may print multiple lines; take the last non-empty.
	if lines := strings.Split(path, "\n"); len(lines) > 1 {
		for i := len(lines) - 1; i >= 0; i-- {
			if s := strings.TrimSpace(lines[i]); s != "" {
				path = s
				break
			}
		}
	}
	if path == "" {
		return "", fmt.Errorf("%w: %s %s", ErrEnsureHeliumEmptyPath, heliumBrowserTool, heliumBrowserBin)
	}
	return path, nil
}

// resolveWorkspaced returns a workspaced binary path: env override, PATH, or bootstrap.
func resolveWorkspaced(ctx context.Context) (string, error) {
	if p := workspacedPathOverride(); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("ELETROCROMO_WORKSPACED=%q: %w", p, err)
		}
		return p, nil
	}
	if p, err := lookPath("workspaced"); err == nil {
		return p, nil
	}
	return bootstrapWorkspaced(ctx)
}
