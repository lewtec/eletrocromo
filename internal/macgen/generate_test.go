package macgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		id, name, want string
	}{
		{"br.tec.lew.counter", "Counter", "Counter"},
		{"br.tec.lew.counter", "My App", "My-App"},
		{"br.tec.lew.counter", "", "counter"},
		{"br.tec.lew.x", "!!!", "x"},
	}
	for _, tt := range tests {
		t.Run(tt.want+"/"+tt.name, func(t *testing.T) {
			t.Parallel()
			got := productName(tt.id, tt.name)
			if got != tt.want {
				t.Fatalf("productName(%q, %q) = %q; want %q", tt.id, tt.name, got, tt.want)
			}
		})
	}
}

func TestDefaultOutApp(t *testing.T) {
	t.Parallel()
	got := DefaultOutApp("Counter", "/tmp/proj")
	want := filepath.Join("/tmp/proj", "dist", "Counter.app")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	got = DefaultOutApp("Counter.app", "/tmp/proj")
	if got != want {
		t.Fatalf("suffix: got %q want %q", got, want)
	}
}

func TestCreate_WritesHost(t *testing.T) {
	out := t.TempDir()
	err := Create(Options{
		OutDir: out,
		Config: Config{
			PackageID: "br.tec.lew.counter",
			AppName:   "Counter",
			GoMain:    ".",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	mustExist := []string{
		"eletrocromo.json",
		"project.yml",
		"Info.plist",
		"README.md",
		"Sources/AppDelegate.swift",
		"Sources/ServerProcess.swift",
		"Sources/MainWindow.swift",
	}
	for _, rel := range mustExist {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}

	yml, err := os.ReadFile(filepath.Join(out, "project.yml"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(yml)
	if !strings.Contains(s, "PRODUCT_BUNDLE_IDENTIFIER: br.tec.lew.counter") {
		t.Fatalf("bundle id missing:\n%s", s)
	}
	if !strings.Contains(s, "ENABLE_APP_SANDBOX: NO") {
		t.Fatalf("sandbox not off:\n%s", s)
	}

	plist, err := os.ReadFile(filepath.Join(out, "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	ps := string(plist)
	if !strings.Contains(ps, "br.tec.lew.counter") {
		t.Fatalf("plist id:\n%s", ps)
	}
	if !strings.Contains(ps, "NSAllowsLocalNetworking") {
		t.Fatalf("plist ATS:\n%s", ps)
	}

	jsonb, err := os.ReadFile(filepath.Join(out, "eletrocromo.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(jsonb), `"package_id": "br.tec.lew.counter"`) {
		t.Fatalf("json: %s", jsonb)
	}

	swift, err := os.ReadFile(filepath.Join(out, "Sources/ServerProcess.swift"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(swift), "ELETROCROMO_NO_UI") {
		t.Fatalf("helper env missing:\n%s", swift)
	}
}

func TestCreate_RejectsBadID(t *testing.T) {
	err := Create(Options{OutDir: t.TempDir(), Config: Config{PackageID: "Not an id"}})
	if err == nil {
		t.Fatal("expected error")
	}
}
