package apk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Splash error paths must not render the authenticated READY URL (token in query).
func TestCreate_MainActivityRedactsErrorURLs(t *testing.T) {
	out := t.TempDir()
	require.NoError(t, Create(Options{
		OutDir: out,
		Config: Config{
			PackageID: "br.tec.lew.counter",
			AppName:   "Counter",
			GoMain:    ".",
		},
	}))

	mainKt, err := os.ReadFile(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/MainActivity.kt"))
	require.NoError(t, err)
	s := string(mainKt)

	assert.Contains(t, s, "fun redactUrlForDisplay")
	assert.Contains(t, s, "redactUrlForDisplay(request.url")
	// Raw request.url must not appear in splash detail strings.
	assert.NotContains(t, s, "\n${request.url}")
	if strings.Contains(s, "append(reqUrl)") {
		// reqUrl must be assigned from redactUrlForDisplay
		assert.Contains(t, s, "val reqUrl = redactUrlForDisplay(")
	}
	// Generated source should not hardcode a sample token query in error UI.
	assert.NotContains(t, s, "?token=")
}
