package apk

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/eletrocromo/internal/gen/common"
)

func TestResolveGoMain_UsesCommonSentinel(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "main.go")
	if err := os.WriteFile(file, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveGoMain(file, root)
	if !errors.Is(err, ErrGoMainNotDir) || !errors.Is(err, common.ErrGoMainNotDir) {
		t.Fatalf("got %v", err)
	}

	got, err := ResolveGoMain(".", root)
	if err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != abs {
		t.Fatalf("got %q want %q", got, abs)
	}
}

func TestLoadConfig_EmptyPath(t *testing.T) {
	_, _, err := LoadConfig("  ")
	if !errors.Is(err, ErrConfigPathEmpty) || !errors.Is(err, common.ErrConfigPathEmpty) {
		t.Fatalf("got %v", err)
	}
}
