package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStampPackagingVersion_FillsEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	vi, name, code := StampPackagingVersion(dir, "", 0)
	assert.NotEmpty(t, vi.Version)
	assert.NotEmpty(t, name)
	assert.Greater(t, code, 0)
}

func TestStampPackagingVersion_KeepsExplicit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, name, code := StampPackagingVersion(dir, "9.8.7", 42)
	assert.Equal(t, "9.8.7", name)
	assert.Equal(t, 42, code)
}
