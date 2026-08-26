package dirs_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/eletrocromo"
	"github.com/lewtec/eletrocromo/driver/dirs"
	_ "github.com/lewtec/eletrocromo/driver/dirs/os"
)

func TestResolve_InboxUnderCache(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ELETROCROMO_DATA_DIR", filepath.Join(root, "data"))
	t.Setenv("ELETROCROMO_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("ELETROCROMO_CONFIG_DIR", filepath.Join(root, "config"))

	got, err := dirs.Resolve(t.Context(), "br.tec.lew.counter")
	if err != nil {
		t.Fatal(err)
	}
	wantInbox := filepath.Join(root, "cache", "inbox")
	if got.Inbox != wantInbox {
		t.Fatalf("Inbox = %q; want %q", got.Inbox, wantInbox)
	}
	if !strings.HasPrefix(got.Inbox, got.Cache+string(filepath.Separator)) && got.Inbox != filepath.Join(got.Cache, "inbox") {
		t.Fatalf("Inbox %q is not under Cache %q", got.Inbox, got.Cache)
	}
	for _, dir := range []string{got.Data, got.Cache, got.Config, got.Inbox} {
		st, err := os.Stat(dir)
		if err != nil || !st.IsDir() {
			t.Fatalf("dir %q: %v", dir, err)
		}
	}
}

func TestResolve_RejectsBadAppID(t *testing.T) {
	_, err := dirs.Resolve(t.Context(), "not-an-id")
	if !errors.Is(err, eletrocromo.ErrAppIDNotReverseDNS) {
		t.Fatalf("err = %v; want ErrAppIDNotReverseDNS", err)
	}
}
