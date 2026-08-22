package iosgen

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

func TestExcludedArch(t *testing.T) {
	t.Parallel()
	if got := excludedArch("arm64"); got != "x86_64" {
		t.Fatalf("arm64: got %q", got)
	}
	if got := excludedArch("x86_64"); got != "arm64" {
		t.Fatalf("x86_64: got %q", got)
	}
}

func TestNormalizeSDK(t *testing.T) {
	t.Parallel()
	got, err := normalizeSDK("")
	if err != nil || got != SDKSimulator {
		t.Fatalf("empty: got %q %v", got, err)
	}
	got, err = normalizeSDK("simulator")
	if err != nil || got != SDKSimulator {
		t.Fatalf("simulator: got %q %v", got, err)
	}
	got, err = normalizeSDK("device")
	if err != nil || got != SDKDevice {
		t.Fatalf("device: got %q %v", got, err)
	}
	if _, err := normalizeSDK("watchos"); err == nil {
		t.Fatal("expected error for watchos")
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
		"Sources/RootViewController.swift",
		"Sources/eletrocromo-Bridging-Header.h",
		"Assets.xcassets/AppIcon.appiconset/Contents.json",
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
	if !strings.Contains(s, "platform: iOS") {
		t.Fatalf("platform missing:\n%s", s)
	}
	if !strings.Contains(s, "TARGETED_DEVICE_FAMILY: \"1,2\"") {
		t.Fatalf("device family missing:\n%s", s)
	}
	if !strings.Contains(s, "SWIFT_OBJC_BRIDGING_HEADER") {
		t.Fatalf("bridging header missing:\n%s", s)
	}
	if !strings.Contains(s, "ENABLE_DEBUG_DYLIB: NO") {
		t.Fatalf("debug dylib not off:\n%s", s)
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
	if !strings.Contains(ps, "LSRequiresIPhoneOS") {
		t.Fatalf("plist iPhoneOS:\n%s", ps)
	}

	jsonb, err := os.ReadFile(filepath.Join(out, "eletrocromo.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(jsonb), `"package_id": "br.tec.lew.counter"`) {
		t.Fatalf("json: %s", jsonb)
	}
	if !strings.Contains(string(jsonb), `"generator": "eletrocromo-ios"`) {
		t.Fatalf("generator: %s", jsonb)
	}

	swift, err := os.ReadFile(filepath.Join(out, "Sources/ServerProcess.swift"))
	if err != nil {
		t.Fatal(err)
	}
	ss := string(swift)
	if !strings.Contains(ss, "ELETROCROMO_READY") {
		t.Fatalf("ready prefix missing:\n%s", ss)
	}
	if !strings.Contains(ss, "EletrocromoStart") {
		t.Fatalf("c-archive entry missing:\n%s", ss)
	}

	ui, err := os.ReadFile(filepath.Join(out, "Sources/RootViewController.swift"))
	if err != nil {
		t.Fatal(err)
	}
	us := string(ui)
	if !strings.Contains(us, "WKWebView") {
		t.Fatalf("webview missing:\n%s", us)
	}
	if !strings.Contains(us, "arrow.clockwise") {
		t.Fatalf("reload symbol missing:\n%s", us)
	}
	if !strings.Contains(us, "UIApplication.shared.open") {
		t.Fatalf("off-loopback open missing:\n%s", us)
	}

	hdr, err := os.ReadFile(filepath.Join(out, "Sources/eletrocromo-Bridging-Header.h"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(hdr), "libeletrocromo.h") {
		t.Fatalf("header: %s", hdr)
	}
}

func TestCreate_RejectsBadID(t *testing.T) {
	err := Create(Options{OutDir: t.TempDir(), Config: Config{PackageID: "Not an id"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBridgeSource_ExportsStart(t *testing.T) {
	t.Parallel()
	if !strings.Contains(iosBridgeSource, "//export EletrocromoStart") {
		t.Fatal("missing export")
	}
	if !strings.Contains(iosBridgeSource, "ELETROCROMO_NO_UI") {
		t.Fatal("missing NO_UI")
	}
	if !strings.Contains(iosBridgeSource, "ELETROCROMO_READY_FILE") {
		t.Fatal("missing READY_FILE")
	}
	if !strings.Contains(iosBridgeSource, "main()") {
		t.Fatal("missing main() call")
	}
}
