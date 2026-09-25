package ios

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/eletrocromo/internal/gen/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExcludedArch(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "x86_64", excludedArch("arm64"))
	assert.Equal(t, "arm64", excludedArch("x86_64"))
}

func TestNormalizeSDK(t *testing.T) {
	t.Parallel()
	got, err := normalizeSDK("")
	require.NoError(t, err)
	assert.Equal(t, SDKSimulator, got)
	got, err = normalizeSDK("simulator")
	require.NoError(t, err)
	assert.Equal(t, SDKSimulator, got)
	got, err = normalizeSDK("device")
	require.NoError(t, err)
	assert.Equal(t, SDKDevice, got)
	_, err = normalizeSDK("watchos")
	require.Error(t, err)
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
	require.NoError(t, err)

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
		"Assets.xcassets/SplashLogo.imageset/Contents.json",
		"LaunchScreen.storyboard",
	}
	for _, rel := range mustExist {
		_, err := os.Stat(filepath.Join(out, rel))
		assert.NoError(t, err, "missing %s", rel)
	}

	yml, err := os.ReadFile(filepath.Join(out, "project.yml"))
	require.NoError(t, err)
	s := string(yml)
	assert.Contains(t, s, "PRODUCT_BUNDLE_IDENTIFIER: br.tec.lew.counter")
	assert.Contains(t, s, "platform: iOS")
	assert.Contains(t, s, "TARGETED_DEVICE_FAMILY: \"1,2\"")
	assert.Contains(t, s, "SWIFT_OBJC_BRIDGING_HEADER")
	assert.Contains(t, s, "ENABLE_DEBUG_DYLIB: NO")
	assert.Contains(t, s, "LaunchScreen.storyboard")
	assert.NotContains(t, s, "type: app-extension")
	assert.Contains(t, s, "CODE_SIGNING_ALLOWED: YES")

	plist, err := os.ReadFile(filepath.Join(out, "Info.plist"))
	require.NoError(t, err)
	ps := string(plist)
	assert.Contains(t, ps, "br.tec.lew.counter")
	assert.Contains(t, ps, "NSAllowsLocalNetworking")
	assert.Contains(t, ps, "LSRequiresIPhoneOS")
	assert.Contains(t, ps, "UILaunchStoryboardName")
	assert.Contains(t, ps, "LaunchScreen")
	assert.NotContains(t, ps, "UILaunchScreen")

	story, err := os.ReadFile(filepath.Join(out, "LaunchScreen.storyboard"))
	require.NoError(t, err)
	ssb := string(story)
	assert.Contains(t, ssb, `constant="120"`)
	assert.Contains(t, ssb, "SplashLogo")

	jsonb, err := os.ReadFile(filepath.Join(out, "eletrocromo.json"))
	require.NoError(t, err)
	assert.Contains(t, string(jsonb), `"package_id": "br.tec.lew.counter"`)
	assert.Contains(t, string(jsonb), `"generator": "eletrocromo-ios"`)

	swift, err := os.ReadFile(filepath.Join(out, "Sources/ServerProcess.swift"))
	require.NoError(t, err)
	ss := string(swift)
	assert.Contains(t, ss, "ELETROCROMO_READY")
	assert.Contains(t, ss, "EletrocromoStart")
	assert.Contains(t, ss, "dirs.cache.path")

	ui, err := os.ReadFile(filepath.Join(out, "Sources/RootViewController.swift"))
	require.NoError(t, err)
	us := string(ui)
	assert.Contains(t, us, "WKWebView")
	assert.Contains(t, us, "UIRefreshControl")
	assert.Contains(t, us, "SplashLogo")
	assert.Contains(t, us, "Try again")
	assert.Contains(t, us, "openExternal")
	assert.Contains(t, us, "revealIfStuck")
	assert.NotContains(t, us, "UIBarButtonItem")
	assert.NotContains(t, us, "arrow.clockwise")
	assert.Contains(t, us, "UIApplication.shared.open")

	delegate, err := os.ReadFile(filepath.Join(out, "Sources/AppDelegate.swift"))
	require.NoError(t, err)
	ds := string(delegate)
	assert.NotContains(t, ds, "UINavigationController")
	assert.Contains(t, ds, "quietSplash")
	assert.Contains(t, ds, "applicationDidBecomeActive")

	hdr, err := os.ReadFile(filepath.Join(out, "Sources/eletrocromo-Bridging-Header.h"))
	require.NoError(t, err)
	assert.Contains(t, string(hdr), "libeletrocromo.h")
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
	assert.Contains(t, ps, "UTImportedTypeDeclarations")
	assert.Contains(t, ps, "LSHandlerRank")
	yml, err := os.ReadFile(filepath.Join(out, "project.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(yml), "type: app-extension")
	_, err = os.Stat(filepath.Join(out, "ShareExtension/ShareViewController.swift"))
	require.NoError(t, err)
	extPlist, err := os.ReadFile(filepath.Join(out, "ShareExtension/Info.plist"))
	require.NoError(t, err)
	assert.Contains(t, string(extPlist), "NSExtensionActivationSupportsImageWithMaxCount")
	_, err = os.Stat(filepath.Join(out, "Sources/OpenDrop.swift"))
	require.NoError(t, err)
	openDrop, err := os.ReadFile(filepath.Join(out, "Sources/OpenDrop.swift"))
	require.NoError(t, err)
	assert.Contains(t, string(openDrop), "adoptGroupLine")
	shareSrc, err := os.ReadFile(filepath.Join(out, "ShareExtension/ShareViewController.swift"))
	require.NoError(t, err)
	assert.Contains(t, string(shareSrc), "loadFileRepresentation")
}

func TestCreate_RejectsBadID(t *testing.T) {
	err := Create(Options{OutDir: t.TempDir(), Config: Config{PackageID: "Not an id"}})
	require.Error(t, err)
}

func TestBridgeSource_ExportsStart(t *testing.T) {
	t.Parallel()
	assert.Contains(t, iosBridgeSource, "//export EletrocromoStart")
	assert.Contains(t, iosBridgeSource, "ELETROCROMO_NO_UI")
	assert.Contains(t, iosBridgeSource, "ELETROCROMO_READY_FILE")
	assert.Contains(t, iosBridgeSource, "ELETROCROMO_CACHE_DIR")
	assert.Contains(t, iosBridgeSource, "ELETROCROMO_DATA_DIR")
	assert.Contains(t, iosBridgeSource, "ELETROCROMO_CONFIG_DIR")
	assert.Contains(t, iosBridgeSource, "main()")
}
