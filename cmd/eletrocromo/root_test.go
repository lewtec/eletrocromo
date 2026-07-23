package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoot_HelpListsAndroid(t *testing.T) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "android") {
		t.Fatalf("help missing android:\n%s", s)
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
