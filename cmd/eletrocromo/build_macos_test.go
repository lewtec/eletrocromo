package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildMacOS_Help(t *testing.T) {
	out, err := runCLI(t, "build", "macos", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "--go-only") {
		t.Fatalf("help missing --go-only:\n%s", out)
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
	buf, err := runCLI(t,
		"build", "macos",
		"--config", counter,
		"--go-only",
		"--workdir", work,
		"--output", iconsOut,
	)
	if err != nil {
		t.Fatalf("%v\n%s", err, buf)
	}
	if _, err := os.Stat(filepath.Join(work, "bin", "eletrocromo-server")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf, "helper:") {
		t.Fatalf("stdout: %s", buf)
	}
}
