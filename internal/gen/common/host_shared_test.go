package common

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteHostSharedSwift(t *testing.T) {
	out := t.TempDir()
	if err := WriteHostSharedSwift(out); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(out, "Sources", "HostShared.swift"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, needle := range []string{"func materialize", "func append", "func jsonLine", "func decode"} {
		if !strings.Contains(s, needle) {
			t.Fatalf("missing %s", needle)
		}
	}
	if err := WriteHostSharedSwift("  "); !errors.Is(err, ErrOutDirRequired) {
		t.Fatalf("got %v", err)
	}
}
