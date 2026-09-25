package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAndroidName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"devel", "0.0.0-devel"},
		{"v1.2.3", "1.2.3"},
		{"1.2.3", "1.2.3"},
		{"v1.2.3-5-gabcdef", "1.2.3-5-gabcdef"},
		{"", "0.0.0-devel"},
	}
	for _, tc := range cases {
		got := Info{Version: tc.in}.AndroidName()
		assert.Equal(t, tc.want, got)
	}
}

func TestAndroidCodeFrom_Semver(t *testing.T) {
	assert.Equal(t, 1_002_003, AndroidCodeFrom("v1.2.3", 0))
	// 0*1e6 + 1*1e3 + 0 = 1000
	assert.Equal(t, 1000, AndroidCodeFrom("0.1.0", 99))
	assert.Equal(t, 42, AndroidCodeFrom("devel", 42))
	assert.Equal(t, 1, AndroidCodeFrom("abcdef", 0))
}

func TestResolve_Defaults(t *testing.T) {
	// Do not mutate package vars permanently beyond test — save/restore.
	oldV, oldC, oldD, oldB := Version, Commit, Date, BuiltBy
	t.Cleanup(func() {
		Version, Commit, Date, BuiltBy = oldV, oldC, oldD, oldB
	})
	Version, Commit, Date, BuiltBy = "devel", "", "", ""
	info := Resolve()
	require.NotEmpty(t, info.Version)
	assert.Contains(t, info.String(), info.Version)
}

func TestGoBuildLdflags(t *testing.T) {
	lf := Info{Version: "v1.0.0", Commit: "abc", Date: "2026-01-01T00:00:00Z", BuiltBy: "test"}.GoBuildLdflags()
	assert.Contains(t, lf, "-s -w")
	assert.Equal(t, "-s -w -X github.com/lewtec/lewkit/x/release.version=v1.0.0", lf)
}
