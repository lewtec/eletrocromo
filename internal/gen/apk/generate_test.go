package apk

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/eletrocromo/internal/gen/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreate_PackageIDLayout(t *testing.T) {
	out := t.TempDir()
	err := Create(Options{
		OutDir: out,
		Config: Config{
			PackageID: "br.tec.lew.counter",
			AppName:   "Counter",
			GoMain:    "../../examples/counter",
		},
	})
	require.NoError(t, err)

	mustExist := []string{
		"eletrocromo.json",
		"settings.gradle.kts",
		"app/build.gradle.kts",
		"app/src/main/AndroidManifest.xml",
		"app/src/main/java/br/tec/lew/counter/MainActivity.kt",
		"app/src/main/java/br/tec/lew/counter/ServerService.kt",
		"app/src/main/res/xml/network_security_config.xml",
		"app/src/main/res/layout/activity_main.xml",
		"scripts/build-go.sh",
		"README.md",
	}
	for _, rel := range mustExist {
		_, err := os.Stat(filepath.Join(out, rel))
		assert.NoError(t, err, "missing %s", rel)
	}

	gradle, err := os.ReadFile(filepath.Join(out, "app/build.gradle.kts"))
	require.NoError(t, err)
	s := string(gradle)
	assert.Contains(t, s, `applicationId = "br.tec.lew.counter"`)
	assert.Contains(t, s, `namespace = "br.tec.lew.counter"`)

	mainKt, err := os.ReadFile(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/MainActivity.kt"))
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(mainKt), "package br.tec.lew.counter\n"))

	cfg, err := os.ReadFile(filepath.Join(out, "eletrocromo.json"))
	require.NoError(t, err)
	assert.Contains(t, string(cfg), `"package_id": "br.tec.lew.counter"`)
	assert.Contains(t, string(cfg), `"go_main": "../../examples/counter"`)

	manifest, err := os.ReadFile(filepath.Join(out, "app/src/main/AndroidManifest.xml"))
	require.NoError(t, err)
	assert.NotContains(t, string(manifest), `android:scheme=`)

	sh, err := os.ReadFile(filepath.Join(out, "scripts/build-go.sh"))
	require.NoError(t, err)
	info, err := os.Stat(filepath.Join(out, "scripts/build-go.sh"))
	require.NoError(t, err)
	assert.True(t, info.Mode()&0o111 != 0)
	assert.Contains(t, string(sh), "GOOS=android")
}

func TestCreate_CapabilitiesIntentFilters(t *testing.T) {
	out := t.TempDir()
	err := Create(Options{
		OutDir: out,
		Config: Config{
			PackageID: "br.tec.lew.counter",
			AppName:   "Counter",
			GoMain:    ".",
			Capabilities: common.Capabilities{
				URL:   &common.URLCap{Schemes: []string{"myapp"}},
				Files: &common.FilesCap{Types: []common.FileType{{Ext: ".md", MIME: "text/markdown"}}},
			},
		},
	})
	require.NoError(t, err)
	manifest, err := os.ReadFile(filepath.Join(out, "app/src/main/AndroidManifest.xml"))
	require.NoError(t, err)
	s := string(manifest)
	assert.Contains(t, s, `android:scheme="myapp"`)
	assert.Contains(t, s, `android:mimeType="text/markdown"`)
	_, err = os.Stat(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/OpenDrop.kt"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/ShareOut.kt"))
	require.NoError(t, err)
	man, err := os.ReadFile(filepath.Join(out, "app/src/main/AndroidManifest.xml"))
	require.NoError(t, err)
	assert.Contains(t, string(man), "FileProvider")
}

func TestCreate_RejectsBadID(t *testing.T) {
	err := Create(Options{
		OutDir: t.TempDir(),
		Config: Config{PackageID: "Not.Valid"},
	})
	require.Error(t, err)
}

func TestCreate_RequiresForceWhenNonEmpty(t *testing.T) {
	out := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(out, "keep"), []byte("x"), 0o644))
	err := Create(Options{
		OutDir: out,
		Config: Config{PackageID: "br.tec.lew.x"},
	})
	require.Error(t, err)
	require.NoError(t, Create(Options{
		OutDir: out,
		Force:  true,
		Config: Config{PackageID: "br.tec.lew.x", AppName: "X"},
	}))
	_, err = os.Stat(filepath.Join(out, "keep"))
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestCreate_DefaultAppNameFromID(t *testing.T) {
	out := t.TempDir()
	require.NoError(t, Create(Options{
		OutDir: out,
		Config: Config{PackageID: "br.tec.lew.myapp"},
	}))
	stringsXML, err := os.ReadFile(filepath.Join(out, "app/src/main/res/values/strings.xml"))
	require.NoError(t, err)
	assert.Contains(t, string(stringsXML), ">myapp</string>")
}
