package main

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/lewtec/eletrocromo/internal/gen/apk"
	"github.com/lewtec/eletrocromo/internal/gen/common"
)

func TestHostOutApp(t *testing.T) {
	explicit := buildCmdFlags{out: "  out.app "}
	if got := explicit.hostOutApp("Counter", "br.tec.lew.counter", "/tmp/proj"); got != "out.app" {
		t.Fatalf("explicit: %q", got)
	}
	fallback := buildCmdFlags{}
	got := fallback.hostOutApp("", "br.tec.lew.counter", "/tmp/proj")
	want := filepath.Join("/tmp/proj", "dist", "counter.app")
	if got != want {
		t.Fatalf("default: got %q want %q", got, want)
	}
	goOnly := buildCmdFlags{goOnly: true}
	if got := goOnly.hostOutApp("Counter", "br.tec.lew.counter", "/tmp/proj"); got != "" {
		t.Fatalf("go-only: %q", got)
	}
}

func TestIOSAndMacConfigFromAPK(t *testing.T) {
	src := apk.Config{
		PackageID:    "br.tec.lew.counter",
		AppName:      "Counter",
		VersionName:  "1.2.3",
		VersionCode:  4,
		GoMain:       "./cmd",
		Icon:         "icon.png",
		Capabilities: common.Capabilities{Share: &common.ShareCap{}},
	}
	iosCfg := iosConfigFromAPK(src)
	macCfg := macConfigFromAPK(src)
	if iosCfg.PackageID != src.PackageID || iosCfg.AppName != src.AppName ||
		iosCfg.VersionName != src.VersionName || iosCfg.VersionCode != src.VersionCode ||
		iosCfg.GoMain != src.GoMain || iosCfg.Icon != src.Icon || iosCfg.Capabilities.Share == nil {
		t.Fatalf("ios: %+v", iosCfg)
	}
	if macCfg.PackageID != src.PackageID || macCfg.AppName != src.AppName ||
		macCfg.VersionName != src.VersionName || macCfg.VersionCode != src.VersionCode ||
		macCfg.GoMain != src.GoMain || macCfg.Icon != src.Icon || macCfg.Capabilities.Share == nil {
		t.Fatalf("mac: %+v", macCfg)
	}
}

func TestBuild_MissingPackageID(t *testing.T) {
	for _, target := range []string{"android", "ios", "macos"} {
		t.Run(target, func(t *testing.T) {
			t.Chdir(t.TempDir())
			cmd := newRootCmd()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs([]string{"build", target, "--go-only"})
			err := cmd.Execute()
			if !errors.Is(err, ErrMissingPackageID) {
				t.Fatalf("%v\n%s", err, buf.String())
			}
		})
	}
}
