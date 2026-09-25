package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildIOS_Help(t *testing.T) {
	out, err := runCLI(t, "build", "ios", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "--go-only") {
		t.Fatalf("help missing --go-only:\n%s", out)
	}
	if !strings.Contains(out, "--sdk") {
		t.Fatalf("help missing --sdk:\n%s", out)
	}
}

func TestBuildIOS_GoOnly_Counter(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	counter := filepath.Join(repoRoot, "examples", "counter", "eletrocromo.json")
	if _, err := os.Stat(counter); err != nil {
		t.Skip(err)
	}
	work := t.TempDir()
	iconsOut := filepath.Join(t.TempDir(), "icons")
	buf, err := runCLI(t,
		"build", "ios",
		"--config", counter,
		"--go-only",
		"--workdir", work,
		"--output", iconsOut,
	)
	if err != nil {
		t.Fatalf("%v\n%s", err, buf)
	}
	if _, err := os.Stat(filepath.Join(work, "project.yml")); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "darwin" {
		if _, err := os.Stat(filepath.Join(work, "lib", "libeletrocromo.a")); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf, "archive:") {
			t.Fatalf("stdout: %s", buf)
		}
	}
}
