package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCapabilities_UnknownKey(t *testing.T) {
	_, err := ParseCapabilities([]byte(`{"camera":{"usage":"x"}}`))
	require.ErrorIs(t, err, ErrCapabilityUnknown)
}

func TestParseCapabilities_URLAndFiles(t *testing.T) {
	c, err := ParseCapabilities([]byte(`{
		"url":{"schemes":["MyApp"]},
		"files":{"types":[{"ext":"md","mime":"text/markdown"}]}
	}`))
	require.NoError(t, err)
	require.NotNil(t, c.URL)
	require.Len(t, c.URL.Schemes, 1)
	assert.Equal(t, "myapp", c.URL.Schemes[0])
	require.NotNil(t, c.Files)
	require.NotEmpty(t, c.Files.Types)
	assert.Equal(t, ".md", c.Files.Types[0].Ext)
	assert.Equal(t, "text/markdown", c.Files.Types[0].MIME)
}

func TestValidate_ReservedScheme(t *testing.T) {
	c := Capabilities{URL: &URLCap{Schemes: []string{"https"}}}
	require.ErrorIs(t, c.Validate(), ErrSchemeInvalid)
}

func TestAndroidIntentFilters(t *testing.T) {
	c := Capabilities{
		URL:   &URLCap{Schemes: []string{"myapp"}},
		Files: &FilesCap{Types: []FileType{{Ext: ".pdf", MIME: "application/pdf"}}},
	}
	s := c.AndroidIntentFilters()
	assert.Contains(t, s, `android:scheme="myapp"`)
	assert.Contains(t, s, `android:mimeType="application/pdf"`)
	img := Capabilities{Files: &FilesCap{Types: []FileType{{Ext: ".jpg", MIME: "image/jpeg"}}}}
	is := img.AndroidIntentFilters()
	assert.Contains(t, is, `android:mimeType="image/*"`)
	assert.Contains(t, is, "SEND_MULTIPLE")
}

func TestPlistFragments(t *testing.T) {
	c := Capabilities{
		URL:   &URLCap{Schemes: []string{"myapp"}},
		Files: &FilesCap{Types: []FileType{{Ext: ".md", MIME: "text/markdown"}}},
	}
	u := c.PlistURLTypes("br.tec.lew.demo")
	assert.Contains(t, u, "CFBundleURLTypes")
	assert.Contains(t, u, "myapp")
	d := c.PlistDocumentTypes()
	assert.Contains(t, d, "CFBundleDocumentTypes")
	assert.Contains(t, d, "md")
	assert.Contains(t, d, "LSSupportsOpeningDocumentsInPlace")
	assert.Contains(t, d, "UTImportedTypeDeclarations")
	assert.Contains(t, d, "LSHandlerRank")
}
