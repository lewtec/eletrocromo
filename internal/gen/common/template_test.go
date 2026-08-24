package common

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestWalkTemplate_RendersAndCopies(t *testing.T) {
	src := fstest.MapFS{
		"template/Info.plist.tmpl": {Data: []byte(`<name>{{.Name}}</name>`)},
		"template/static.txt":      {Data: []byte("keep")},
	}
	out := t.TempDir()
	if err := WalkTemplate(src, struct{ Name string }{Name: "App"}, out); err != nil {
		t.Fatal(err)
	}
	plist, err := os.ReadFile(filepath.Join(out, "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if string(plist) != "<name>App</name>" {
		t.Fatalf("plist: %q", plist)
	}
	static, err := os.ReadFile(filepath.Join(out, "static.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(static) != "keep" {
		t.Fatalf("static: %q", static)
	}
}
