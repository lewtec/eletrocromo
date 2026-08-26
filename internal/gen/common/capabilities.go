package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Capability parse/validate sentinels.
var (
	ErrCapabilityUnknown = errors.New("unknown capability")
	ErrCapabilityEmpty   = errors.New("capability is empty")
	ErrSchemeInvalid     = errors.New("invalid url scheme")
	ErrFileTypeInvalid   = errors.New("invalid file type")
)

var schemePat = regexp.MustCompile(`^[a-z][a-z0-9+\-.]*$`)

// reservedSchemes cannot be claimed as app handlers.
var reservedSchemes = map[string]struct{}{
	"http": {}, "https": {}, "file": {}, "ftp": {}, "data": {},
	"javascript": {}, "about": {}, "blob": {}, "ws": {}, "wss": {},
}

// Capabilities is the closed catalog from eletrocromo.json.
type Capabilities struct {
	URL   *URLCap   `json:"url,omitempty"`
	Files *FilesCap `json:"files,omitempty"`
}

// URLCap registers custom URL schemes.
type URLCap struct {
	Schemes []string `json:"schemes"`
}

// FilesCap registers open-with / share types.
type FilesCap struct {
	Types []FileType `json:"types"`
}

// FileType is one extension + mime pair.
type FileType struct {
	Ext  string `json:"ext"`
	MIME string `json:"mime"`
}

// Empty reports whether any capability is set.
func (c Capabilities) Empty() bool {
	return (c.URL == nil || len(c.URL.Schemes) == 0) &&
		(c.Files == nil || len(c.Files.Types) == 0)
}

// ParseCapabilities decodes the capabilities object. Unknown keys fail.
func ParseCapabilities(raw json.RawMessage) (Capabilities, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return Capabilities{}, nil
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return Capabilities{}, fmt.Errorf("capabilities: %w", err)
	}
	for k := range probe {
		switch k {
		case "url", "files":
		default:
			return Capabilities{}, fmt.Errorf("%w: %q", ErrCapabilityUnknown, k)
		}
	}
	var c Capabilities
	if err := json.Unmarshal(raw, &c); err != nil {
		return Capabilities{}, fmt.Errorf("capabilities: %w", err)
	}
	if err := c.Validate(); err != nil {
		return Capabilities{}, err
	}
	return c, nil
}

// Validate checks declared capabilities. Empty is valid.
func (c Capabilities) Validate() error {
	if c.URL != nil {
		if len(c.URL.Schemes) == 0 {
			return fmt.Errorf("url: %w", ErrCapabilityEmpty)
		}
		seen := map[string]struct{}{}
		for i, s := range c.URL.Schemes {
			s = strings.ToLower(strings.TrimSpace(s))
			s = strings.TrimSuffix(s, "://")
			if !schemePat.MatchString(s) {
				return fmt.Errorf("%w: %q", ErrSchemeInvalid, c.URL.Schemes[i])
			}
			if _, bad := reservedSchemes[s]; bad {
				return fmt.Errorf("%w: reserved scheme %q", ErrSchemeInvalid, s)
			}
			if _, ok := seen[s]; ok {
				return fmt.Errorf("%w: duplicate scheme %q", ErrSchemeInvalid, s)
			}
			seen[s] = struct{}{}
			c.URL.Schemes[i] = s
		}
	}
	if c.Files != nil {
		if len(c.Files.Types) == 0 {
			return fmt.Errorf("files: %w", ErrCapabilityEmpty)
		}
		for i, ft := range c.Files.Types {
			ext, mime, err := normalizeFileType(ft)
			if err != nil {
				return err
			}
			c.Files.Types[i] = FileType{Ext: ext, MIME: mime}
		}
	}
	return nil
}

func normalizeFileType(ft FileType) (ext, mime string, err error) {
	ext = strings.ToLower(strings.TrimSpace(ft.Ext))
	ext = strings.TrimPrefix(ext, ".")
	if ext == "" || strings.ContainsAny(ext, `/\`) {
		return "", "", fmt.Errorf("%w: ext %q", ErrFileTypeInvalid, ft.Ext)
	}
	ext = "." + ext
	mime = strings.ToLower(strings.TrimSpace(ft.MIME))
	if mime == "" || !strings.Contains(mime, "/") || strings.ContainsAny(mime, " \t") {
		return "", "", fmt.Errorf("%w: mime %q", ErrFileTypeInvalid, ft.MIME)
	}
	return ext, mime, nil
}

// AndroidIntentFilters is extra <intent-filter> XML for MainActivity.
func (c Capabilities) AndroidIntentFilters() string {
	var b strings.Builder
	if c.URL != nil {
		for _, s := range c.URL.Schemes {
			b.WriteString(`
        <intent-filter>
            <action android:name="android.intent.action.VIEW" />
            <category android:name="android.intent.category.DEFAULT" />
            <category android:name="android.intent.category.BROWSABLE" />
            <data android:scheme="` + xmlAttr(s) + `" />
        </intent-filter>
`)
		}
	}
	if c.Files != nil {
		for _, ft := range c.Files.Types {
			b.WriteString(`
        <intent-filter>
            <action android:name="android.intent.action.VIEW" />
            <action android:name="android.intent.action.SEND" />
            <category android:name="android.intent.category.DEFAULT" />
            <data android:mimeType="` + xmlAttr(ft.MIME) + `" />
        </intent-filter>
`)
		}
	}
	return b.String()
}

// PlistURLTypes is CFBundleURLTypes XML, or empty.
func (c Capabilities) PlistURLTypes(packageID string) string {
	if c.URL == nil || len(c.URL.Schemes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\t<key>CFBundleURLTypes</key>\n\t<array>\n\t\t<dict>\n")
	b.WriteString("\t\t\t<key>CFBundleURLName</key>\n\t\t\t<string>")
	b.WriteString(XMLEscape(packageID))
	b.WriteString("</string>\n\t\t\t<key>CFBundleURLSchemes</key>\n\t\t\t<array>\n")
	for _, s := range c.URL.Schemes {
		b.WriteString("\t\t\t\t<string>")
		b.WriteString(XMLEscape(s))
		b.WriteString("</string>\n")
	}
	b.WriteString("\t\t\t</array>\n\t\t</dict>\n\t</array>\n")
	return b.String()
}

// PlistDocumentTypes is CFBundleDocumentTypes XML, or empty.
func (c Capabilities) PlistDocumentTypes() string {
	if c.Files == nil || len(c.Files.Types) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\t<key>CFBundleDocumentTypes</key>\n\t<array>\n")
	for _, ft := range c.Files.Types {
		name := strings.TrimPrefix(ft.Ext, ".")
		b.WriteString("\t\t<dict>\n")
		b.WriteString("\t\t\t<key>CFBundleTypeName</key>\n\t\t\t<string>")
		b.WriteString(XMLEscape(name))
		b.WriteString("</string>\n")
		b.WriteString("\t\t\t<key>CFBundleTypeRole</key>\n\t\t\t<string>Viewer</string>\n")
		b.WriteString("\t\t\t<key>CFBundleTypeExtensions</key>\n\t\t\t<array>\n")
		b.WriteString("\t\t\t\t<string>")
		b.WriteString(XMLEscape(name))
		b.WriteString("</string>\n\t\t\t</array>\n")
		b.WriteString("\t\t\t<key>LSItemContentTypes</key>\n\t\t\t<array>\n")
		b.WriteString("\t\t\t\t<string>")
		b.WriteString(XMLEscape(utiFor(ft.MIME)))
		b.WriteString("</string>\n\t\t\t</array>\n")
		b.WriteString("\t\t</dict>\n")
	}
	b.WriteString("\t</array>\n")
	return b.String()
}

func xmlAttr(s string) string {
	return XMLEscape(s)
}

func utiFor(mime string) string {
	switch mime {
	case "application/pdf":
		return "com.adobe.pdf"
	case "text/plain":
		return "public.plain-text"
	case "text/markdown":
		return "net.daringfireball.markdown"
	case "image/png":
		return "public.png"
	case "image/jpeg", "image/jpg":
		return "public.jpeg"
	default:
		return "public.data"
	}
}
