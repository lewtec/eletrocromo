package ios

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild_GoOnly_Counter(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	counterDir := filepath.Join(repoRoot, "examples", "counter")
	if _, err := os.Stat(filepath.Join(counterDir, "main.go")); err != nil {
		t.Skip("examples/counter not present")
	}

	work := t.TempDir()
	var buf bytes.Buffer
	res, err := Build(BuildOptions{
		Config: Config{
			PackageID: "br.tec.lew.eletrocromo.counter",
			AppName:   "Counter",
			GoMain:    ".",
		},
		BaseDir:     counterDir,
		WorkDir:     work,
		KeepWorkDir: true,
		GoOnly:      true,
		Stdout:      &buf,
		Stderr:      &buf,
	})
	require.NoError(t, err, buf.String())
	_, err = os.Stat(filepath.Join(work, "project.yml"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(work, "Assets.xcassets", "AppIcon.appiconset", "AppIcon.png"))
	require.NoError(t, err)
	logoDir := filepath.Join(work, "Assets.xcassets", "SplashLogo.imageset")
	for _, name := range []string{"SplashLogo.png", "SplashLogo@2x.png", "SplashLogo@3x.png"} {
		_, err := os.Stat(filepath.Join(logoDir, name))
		require.NoError(t, err)
	}
	if runtime.GOOS != "darwin" {
		assert.Empty(t, res.ArchivePath)
		return
	}
	require.NotEmpty(t, res.ArchivePath)
	st, err := os.Stat(res.ArchivePath)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, st.Size(), int64(1000))
	_, err = os.Stat(strings.TrimSuffix(res.ArchivePath, ".a") + ".h")
	require.NoError(t, err)
}

func TestBuild_FullRequiresDarwin(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("on darwin, full build is allowed")
	}
	_, err := Build(BuildOptions{
		Config:  Config{PackageID: "br.tec.lew.x", AppName: "X", GoMain: "."},
		BaseDir: t.TempDir(),
		OutApp:  filepath.Join(t.TempDir(), "X.app"),
	})
	require.ErrorIs(t, err, ErrDarwinRequired)
}
