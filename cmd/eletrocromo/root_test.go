package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/eletrocromo/internal/icons"
)

func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	oldErr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	out, runErr := captureStdout(t, func() error {
		return run(t.Context(), args)
	})
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stderr = oldErr
	var errBuf bytes.Buffer
	if _, err := io.Copy(&errBuf, r); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return out + errBuf.String(), runErr
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	var buf bytes.Buffer
	copied := make(chan error, 1)
	go func() {
		_, err := io.Copy(&buf, r)
		copied <- err
	}()
	runErr := fn()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = old
	if err := <-copied; err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String(), runErr
}

func TestRoot_HelpListsBuild(t *testing.T) {
	out, err := runCLI(t, "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "build") {
		t.Fatalf("help missing build:\n%s", out)
	}
}

func TestBuild_BareErrors(t *testing.T) {
	_, err := runCLI(t, "build")
	if err == nil {
		t.Fatal("expected error for bare build")
	}
	if !errors.Is(err, ErrMissingBuildTarget) {
		t.Fatalf("want ErrMissingBuildTarget, got %v", err)
	}
}

func TestBuildIcons_Default(t *testing.T) {
	dir := t.TempDir()
	out, err := runCLI(t, "build", "icons", "--output", filepath.Join(dir, "icons"), "--refresh-icons")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(dir, "icons", "manifest.json")); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultConfigPath(t *testing.T) {
	dir := t.TempDir()
	if got := defaultConfigPath(dir); got != "" {
		t.Fatalf("empty dir: got %q", got)
	}
	path := filepath.Join(dir, "eletrocromo.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := defaultConfigPath(dir); got != path {
		t.Fatalf("got %q want %q", got, path)
	}
}

func TestResolveIconSource(t *testing.T) {
	cwd := t.TempDir()
	base := filepath.Join(cwd, "cfg")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	// flag wins over config
	got := resolveIconSource(cwd, "flag.png", base, "cfg.png")
	want := filepath.Join(cwd, "flag.png")
	if got != want {
		t.Fatalf("flag: got %q want %q", got, want)
	}
	// config only, relative to baseDir
	got = resolveIconSource(cwd, "", base, "cfg.png")
	want = filepath.Join(base, "cfg.png")
	if got != want {
		t.Fatalf("cfg: got %q want %q", got, want)
	}
	// empty → default mark
	if got = resolveIconSource(cwd, "", base, ""); got != "" {
		t.Fatalf("empty: got %q", got)
	}
	// absolute flag preserved
	abs := filepath.Join(cwd, "abs.png")
	if got = resolveIconSource(cwd, abs, base, "cfg.png"); got != abs {
		t.Fatalf("abs flag: got %q want %q", got, abs)
	}
}

func TestEnsureBuildIcons(t *testing.T) {
	out := filepath.Join(t.TempDir(), "icons")
	var buf bytes.Buffer
	root, err := ensureBuildIcons(&buf, "", out, false)
	if err != nil {
		t.Fatal(err)
	}
	if !icons.Complete(root) {
		t.Fatalf("generated tree incomplete: %s", root)
	}
	if !strings.Contains(buf.String(), "icons →") {
		t.Fatalf("generate log: %s", buf.String())
	}

	buf.Reset()
	again, err := ensureBuildIcons(&buf, "", out, false)
	if err != nil {
		t.Fatal(err)
	}
	if again != out {
		t.Fatalf("reuse root: got %q want %q", again, out)
	}
	if !strings.Contains(buf.String(), "already present") {
		t.Fatalf("reuse log: %s", buf.String())
	}

	buf.Reset()
	if _, err := ensureBuildIcons(&buf, "", out, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "icons →") {
		t.Fatalf("refresh log: %s", buf.String())
	}
}

func TestRunIconsThen(t *testing.T) {
	out := filepath.Join(t.TempDir(), "icons")
	var got string
	log, err := captureStdout(t, func() error {
		return runIconsThen(t.Context(), iconThen{src: "", out: out, refresh: false, name: "work"}, func(iconRoot string) error {
			got = iconRoot
			return nil
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if !icons.Complete(got) {
		t.Fatalf("work ran without a complete tree: %s", got)
	}
	if !strings.Contains(log, "icons →") {
		t.Fatalf("generate log: %s", log)
	}
}

func TestResolveIconOutput(t *testing.T) {
	cwd := t.TempDir()
	got := resolveIconOutput(cwd, "")
	want := filepath.Join(cwd, "dist/icons")
	if got != want {
		t.Fatalf("default: got %q want %q", got, want)
	}
	got = resolveIconOutput(cwd, "out")
	want = filepath.Join(cwd, "out")
	if got != want {
		t.Fatalf("relative: got %q want %q", got, want)
	}
	abs := filepath.Join(cwd, "abs-out")
	if got = resolveIconOutput(cwd, abs); got != abs {
		t.Fatalf("abs: got %q want %q", got, abs)
	}
}

func TestVersionCmd(t *testing.T) {
	out, err := runCLI(t, "version")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatal("empty version output")
	}
}

func TestAndroidCreate_RequiredFlags(t *testing.T) {
	_, err := runCLI(t, "android", "create")
	if err == nil {
		t.Fatal("expected error without --id/--out")
	}
}

func TestAndroidCreate_WritesProject(t *testing.T) {
	outDir := t.TempDir()
	// Cobra reuses process; run into empty subdir.
	dest := filepath.Join(outDir, "proj")

	buf, err := runCLI(t,
		"android", "create",
		"--id", "br.tec.lew.cli_test",
		"--name", "CLITest",
		"--out", dest,
		"--go-main", ".",
	)
	if err != nil {
		t.Fatalf("%v\n%s", err, buf)
	}
	if _, err := os.Stat(filepath.Join(dest, "eletrocromo.json")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf, "br.tec.lew.cli_test") {
		t.Fatalf("stdout: %s", buf)
	}
}
