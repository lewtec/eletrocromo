package share

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppendAndParse(t *testing.T) {
	dir := t.TempDir()
	path := FilePath(dir)
	item := Item{Text: "hello", URL: "https://example.com"}
	if err := AppendFile(path, item); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseLine(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "hello" || got.URL != "https://example.com" {
		t.Fatalf("%#v", got)
	}
}

func TestValidate_Empty(t *testing.T) {
	if err := (Item{}).validate(); err == nil {
		t.Fatal("want error")
	}
}

func TestValidate_AbsPath(t *testing.T) {
	if err := (Item{Paths: []string{"rel.txt"}}).validate(); err == nil {
		t.Fatal("want error")
	}
	p := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (Item{Paths: []string{p}}).validate(); err != nil {
		t.Fatal(err)
	}
}
