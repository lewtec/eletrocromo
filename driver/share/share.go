// Package share offers content to other apps via the system share sheet.
//
// Call Out from Go. Packaged hosts watch Cache/share.jsonl (import hostfile).
// Desktop Run() uses the desktop impl (clipboard / open).
package share

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/eletrocromo/driver"
	"github.com/lewtec/eletrocromo/driver/dirs"
	_ "github.com/lewtec/eletrocromo/driver/dirs/os"
)

// FileName is the JSONL drop under Cache (host tails this).
const FileName = "share.jsonl"

var (
	ErrEmptyItem = errors.New("share item is empty")
)

// Item is one outbound share. At least one of Text, URL, Paths must be set.
type Item struct {
	Title string   `json:"title,omitempty"`
	Text  string   `json:"text,omitempty"`
	URL   string   `json:"url,omitempty"`
	Paths []string `json:"paths,omitempty"`
}

// Driver presents Item to the OS share UI.
type Driver interface {
	Out(ctx context.Context, item Item) error
}

// FilePath is Cache/share.jsonl.
func FilePath(cacheDir string) string {
	return filepath.Join(cacheDir, FileName)
}

// Out presents item using the selected driver.
func Out(ctx context.Context, item Item) error {
	if err := item.validate(); err != nil {
		return err
	}
	return driver.With(ctx, func(d Driver) error {
		return d.Out(ctx, item)
	})
}

func (item Item) validate() error {
	if strings.TrimSpace(item.Text) == "" && strings.TrimSpace(item.URL) == "" && len(item.Paths) == 0 {
		return ErrEmptyItem
	}
	for _, p := range item.Paths {
		if !filepath.IsAbs(p) {
			return fmt.Errorf("share path must be absolute: %q", p)
		}
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("share path: %w", err)
		}
	}
	return nil
}

// Encode writes one JSONL line.
func Encode(item Item) ([]byte, error) {
	raw, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// ParseLine decodes one JSONL object.
func ParseLine(line []byte) (Item, error) {
	var item Item
	if err := json.Unmarshal(line, &item); err != nil {
		return Item{}, fmt.Errorf("share line: %w", err)
	}
	return item, item.validate()
}

// AppendFile writes item onto path (creates the file).
func AppendFile(path string, item Item) error {
	if err := item.validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	line, err := Encode(item)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(line)
	return err
}

// ResolvePath is Cache/share.jsonl for appID.
func ResolvePath(ctx context.Context, appID string) (string, error) {
	d, err := dirs.Resolve(ctx, appID)
	if err != nil {
		return "", err
	}
	return FilePath(d.Cache), nil
}
