package common

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareOutDir_CreatesAndForceWipes(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "host")
	if err := PrepareOutDir(out, false); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "stale.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := PrepareOutDir(out, false); !errors.Is(err, ErrOutDirNotEmpty) {
		t.Fatalf("want ErrOutDirNotEmpty, got %v", err)
	}
	if err := PrepareOutDir(out, true); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("force wipe left %d entries", len(entries))
	}
}

func TestPrepareOutDir_RejectsFile(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "notdir")
	if err := os.WriteFile(out, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := PrepareOutDir(out, true); !errors.Is(err, ErrOutPathNotDir) {
		t.Fatalf("want ErrOutPathNotDir, got %v", err)
	}
}

func TestResolveGoMain_RejectsFile(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "main.go")
	if err := os.WriteFile(file, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveGoMain(file, root)
	if !errors.Is(err, ErrGoMainNotDir) {
		t.Fatalf("want ErrGoMainNotDir, got %v", err)
	}
}
