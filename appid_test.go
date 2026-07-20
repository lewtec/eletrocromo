package eletrocromo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAppID(t *testing.T) {
	ok := []string{
		"br.tec.lew.counter",
		"com.example.app",
		"a.b",
		"org.foo_bar.baz",
	}
	for _, id := range ok {
		if err := ValidateAppID(id); err != nil {
			t.Errorf("%q: %v", id, err)
		}
	}
	bad := []string{
		"",
		"counter",
		"Counter.app",
		"com.Example.app",
		"../evil",
		"com/example",
		"com.example/app",
		".com.example",
		"com.",
		"1com.example",
	}
	for _, id := range bad {
		if err := ValidateAppID(id); err == nil {
			t.Errorf("%q: want error", id)
		}
	}
}

func TestProfileDir_IsolatesByAppID(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)

	a, err := ProfileDir("br.tec.lew.counter")
	if err != nil {
		t.Fatal(err)
	}
	b, err := ProfileDir("br.tec.lew.basic")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("profiles must differ by app id")
	}
	if !strings.HasPrefix(a, root) || !strings.HasPrefix(b, root) {
		t.Fatalf("profiles not under XDG_DATA_HOME: %q %q", a, b)
	}
	if filepath.Base(a) != "br.tec.lew.counter" {
		t.Fatalf("base = %q", filepath.Base(a))
	}
	st, err := os.Stat(a)
	if err != nil || !st.IsDir() {
		t.Fatalf("profile dir missing: %v", err)
	}
}
