package icons

import (
	"path/filepath"
	"testing"
)

func TestResolveIconRoot_KeepsExplicit(t *testing.T) {
	t.Parallel()
	got, err := ResolveIconRoot("  /tmp/icons  ", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got != "/tmp/icons" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveIconRoot_GeneratesDefault(t *testing.T) {
	work := t.TempDir()
	got, err := ResolveIconRoot("", work)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(work, ".eletrocromo-icons")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if !Complete(got) {
		t.Fatal("expected complete default tree")
	}
}
