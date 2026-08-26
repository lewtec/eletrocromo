package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildMacOS_Help(t *testing.T) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"build", "macos", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "--go-only") {
		t.Fatalf("help missing --go-only:\n%s", s)
	}
}

func TestBuildMacOS_GoOnly_Counter(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	// cmd tests run from cmd/eletrocromo
	counter := filepath.Join(repoRoot, "examples", "counter", "eletrocromo.json")
	if _, err := os.Stat(counter); err != nil {
		t.Skip(err)
	}
	work := t.TempDir()
	iconsOut := filepath.Join(t.TempDir(), "icons")
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{
		"build", "macos",
		"--config", counter,
		"--go-only",
		"--workdir", work,
		"--output", iconsOut,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	if _, err := os.Stat(filepath.Join(work, "bin", "eletrocromo-server")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "helper:") {
		t.Fatalf("stdout: %s", buf.String())
	}
}
