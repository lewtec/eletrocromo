package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/eletrocromo/internal/icons"
)

func TestRoot_HelpListsBuild(t *testing.T) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "build") {
		t.Fatalf("help missing build:\n%s", s)
	}
}

func TestBuild_BareErrors(t *testing.T) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"build"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for bare build")
	}
	if !errors.Is(err, ErrMissingBuildTarget) {
		t.Fatalf("want ErrMissingBuildTarget, got %v", err)
	}
}

func TestBuildIcons_Default(t *testing.T) {
	dir := t.TempDir()
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"build", "icons", "--output", filepath.Join(dir, "icons"), "--refresh-icons"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
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
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	var got string
	err := runIconsThen(cmd, iconThen{src: "", out: out, refresh: false, name: "work"}, func(iconRoot string) error {
		got = iconRoot
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !icons.Complete(got) {
		t.Fatalf("work ran without a complete tree: %s", got)
	}
	if !strings.Contains(buf.String(), "icons →") {
		t.Fatalf("generate log: %s", buf.String())
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
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Fatal("empty version output")
	}
}

func TestAndroidCreate_RequiredFlags(t *testing.T) {
	cmd := newRootCmd()
	var errBuf bytes.Buffer
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"android", "create"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error without --id/--out")
	}
}

func TestAndroidCreate_WritesProject(t *testing.T) {
	outDir := t.TempDir()
	// Cobra reuses process; run into empty subdir.
	dest := filepath.Join(outDir, "proj")

	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{
		"android", "create",
		"--id", "br.tec.lew.cli_test",
		"--name", "CLITest",
		"--out", dest,
		"--go-main", ".",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	if _, err := os.Stat(filepath.Join(dest, "eletrocromo.json")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "br.tec.lew.cli_test") {
		t.Fatalf("stdout: %s", buf.String())
	}
}
