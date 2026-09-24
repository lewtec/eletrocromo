package mac

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/lewtec/eletrocromo/internal/gen/common"
)

func TestConfigWithDefaults_PreservesHostFields(t *testing.T) {
	t.Parallel()
	got, err := (Config{
		PackageID:   " br.tec.lew.counter ",
		AppName:     "Counter",
		VersionName: "1.2.3",
		VersionCode: 4,
		GoMain:      "app",
		Icon:        "icon.png",
		Capabilities: common.Capabilities{
			URL: &common.URLCap{Schemes: []string{"MyApp"}},
		},
	}).withDefaults()
	if err != nil {
		t.Fatal(err)
	}
	if got.PackageID != "br.tec.lew.counter" || got.AppName != "Counter" ||
		got.VersionName != "1.2.3" || got.VersionCode != 4 ||
		got.GoMain != "app" || got.Icon != "icon.png" {
		t.Fatalf("%+v", got)
	}
	if got.Capabilities.URL == nil || len(got.Capabilities.URL.Schemes) != 1 || got.Capabilities.URL.Schemes[0] != "myapp" {
		t.Fatalf("capabilities = %+v", got.Capabilities.URL)
	}
}

func TestConfigWithDefaults_DefaultsAppName(t *testing.T) {
	t.Parallel()
	got, err := (Config{
		PackageID:   "br.tec.lew.myapp",
		VersionName: "1.0.0",
		VersionCode: 1,
	}).withDefaults()
	if err != nil {
		t.Fatal(err)
	}
	if got.AppName != "myapp" || got.GoMain != "." {
		t.Fatalf("%+v", got)
	}
}

func TestEncodeConfigJSON_Generator(t *testing.T) {
	t.Parallel()
	raw, err := encodeConfigJSON(Config{
		PackageID:   "br.tec.lew.counter",
		AppName:     "Counter",
		VersionName: "1.0.0",
		VersionCode: 1,
		GoMain:      ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["generator"] != "eletrocromo-macos" {
		t.Fatalf("generator = %v", doc["generator"])
	}
	if doc["package_id"] != "br.tec.lew.counter" || doc["app_name"] != "Counter" {
		t.Fatalf("doc = %s", raw)
	}
	if !strings.HasSuffix(string(raw), "\n") {
		t.Fatal("missing trailing newline")
	}
}
