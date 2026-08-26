package open

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCollect_EnvAndArgv(t *testing.T) {
	got := Collect([]string{"-test.v", "myapp://item/1", "-flag"}, " /tmp/a.pdf ")
	if len(got) != 2 {
		t.Fatalf("len=%d %#v", len(got), got)
	}
	if got[0].Kind != KindURL || got[0].URL != "myapp://item/1" {
		t.Fatalf("url: %#v", got[0])
	}
	if got[1].Kind != KindFiles || got[1].Paths[0] != "/tmp/a.pdf" {
		t.Fatalf("file: %#v", got[1])
	}
}

func TestParseLine(t *testing.T) {
	ev, err := ParseLine([]byte(`{"kind":"url","url":"myapp://x"}`))
	if err != nil || ev.Kind != KindURL || ev.URL != "myapp://x" {
		t.Fatalf("%#v %v", ev, err)
	}
	ev, err = ParseLine([]byte(`{"kind":"files","paths":["/a"]}`))
	if err != nil || ev.Kind != KindFiles || ev.Paths[0] != "/a" {
		t.Fatalf("%#v %v", ev, err)
	}
}

func TestTailFile_ReadsAppend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	ctx := t.Context()

	got := make(chan Event, 2)
	errCh := make(chan error, 1)
	go func() {
		errCh <- TailFile(ctx, path, func(ev Event) error {
			got <- ev
			return nil
		})
	}()

	if err := os.WriteFile(path, []byte("{\"kind\":\"url\",\"url\":\"myapp://one\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-got:
		if ev.URL != "myapp://one" {
			t.Fatalf("%#v", ev)
		}
	case err := <-errCh:
		t.Fatal(err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout first line")
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{\"kind\":\"files\",\"paths\":[\"/b\"]}\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-got:
		if ev.Kind != KindFiles || ev.Paths[0] != "/b" {
			t.Fatalf("%#v", ev)
		}
	case err := <-errCh:
		t.Fatal(err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout second line")
	}
}
