package mac

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/eletrocromo/internal/gen/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	require.NoError(t, err)

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
		_, err := os.Stat(filepath.Join(out, rel))
		assert.NoError(t, err, "missing %s", rel)
	}

	yml, err := os.ReadFile(filepath.Join(out, "project.yml"))
	require.NoError(t, err)
	s := string(yml)
	assert.Contains(t, s, "PRODUCT_BUNDLE_IDENTIFIER: br.tec.lew.counter")
	assert.Contains(t, s, "ENABLE_APP_SANDBOX: NO")

	plist, err := os.ReadFile(filepath.Join(out, "Info.plist"))
	require.NoError(t, err)
	ps := string(plist)
	assert.Contains(t, ps, "br.tec.lew.counter")
	assert.Contains(t, ps, "NSAllowsLocalNetworking")

	jsonb, err := os.ReadFile(filepath.Join(out, "eletrocromo.json"))
	require.NoError(t, err)
	assert.Contains(t, string(jsonb), `"package_id": "br.tec.lew.counter"`)

	swift, err := os.ReadFile(filepath.Join(out, "Sources/ServerProcess.swift"))
	require.NoError(t, err)
	assert.Contains(t, string(swift), "ELETROCROMO_NO_UI")

	ui, err := os.ReadFile(filepath.Join(out, "Sources/MainWindow.swift"))
	require.NoError(t, err)
	assert.Contains(t, string(ui), "NSTitlebarAccessoryViewController")
	assert.Contains(t, string(ui), "arrow.clockwise")
	assert.Contains(t, string(ui), "openExternal")
}

func TestCreate_CapabilitiesPlist(t *testing.T) {
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
	plist, err := os.ReadFile(filepath.Join(out, "Info.plist"))
	require.NoError(t, err)
	ps := string(plist)
	assert.Contains(t, ps, "CFBundleURLTypes")
	assert.Contains(t, ps, "myapp")
	assert.Contains(t, ps, "CFBundleDocumentTypes")
	_, err = os.Stat(filepath.Join(out, "Sources/OpenDrop.swift"))
	require.NoError(t, err)
}

func TestCreate_RejectsBadID(t *testing.T) {
	err := Create(Options{OutDir: t.TempDir(), Config: Config{PackageID: "Not an id"}})
	require.Error(t, err)
}
