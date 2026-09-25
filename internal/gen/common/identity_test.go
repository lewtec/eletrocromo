package common

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/lewtec/eletrocromo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyHostDefaults_FillsAppNameAndGoMain(t *testing.T) {
	t.Parallel()
	got, err := ApplyHostDefaults(HostConfig{
		PackageID:   " br.tec.lew.counter ",
		VersionName: "1.0.0",
		VersionCode: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, "br.tec.lew.counter", got.PackageID)
	assert.Equal(t, "counter", got.AppName)
	assert.Equal(t, ".", got.GoMain)
	assert.Equal(t, "1.0.0", got.VersionName)
	assert.Equal(t, 1, got.VersionCode)
}

func TestApplyHostDefaults_RejectsBadPackageID(t *testing.T) {
	t.Parallel()
	_, err := ApplyHostDefaults(HostConfig{})
	require.ErrorIs(t, err, eletrocromo.ErrAppIDRequired)
}

func TestEncodeHostJSON(t *testing.T) {
	t.Parallel()
	raw, err := EncodeHostJSON(HostConfig{
		PackageID:   "br.tec.lew.counter",
		AppName:     "Counter",
		VersionName: "1.2.3",
		VersionCode: 4,
		GoMain:      ".",
		Icon:        "icon.png",
	}, "eletrocromo-ios")
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(string(raw), "\n"))
	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))
	assert.Equal(t, float64(1), doc["schema_version"])
	assert.Equal(t, "eletrocromo-ios", doc["generator"])
	assert.Equal(t, "br.tec.lew.counter", doc["package_id"])
	assert.Equal(t, "icon.png", doc["icon"])
}
