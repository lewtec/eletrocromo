// Package open delivers OS file/URL launches to the Go app.
//
// Sources (no HTTP): ELETROCROMO_OPEN, argv, and Cache/open.jsonl
// (hosts append one JSON object per line).
package open

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lewtec/eletrocromo/driver/dirs"
	_ "github.com/lewtec/eletrocromo/driver/dirs/os"
)

var (
	ErrNilHandler  = errors.New("nil handler")
	ErrEmptyURL    = errors.New("empty url")
	ErrEmptyPaths  = errors.New("empty paths")
	ErrUnknownKind = errors.New("unknown kind")
)

// FileName is the JSONL drop under Cache.
const FileName = "open.jsonl"

// EnvOpen is a whitespace-separated list of URLs or file paths (cold start).
const EnvOpen = "ELETROCROMO_OPEN"

// Kind is what the OS asked the app to open.
type Kind int

const (
	KindURL Kind = iota + 1
	KindFiles
)

// Event is one launch or share.
type Event struct {
	Kind  Kind
	URL   string
	Paths []string
}

type wire struct {
	Kind  string   `json:"kind"`
	URL   string   `json:"url,omitempty"`
	Paths []string `json:"paths,omitempty"`
}

// FilePath is Cache/open.jsonl.
func FilePath(cacheDir string) string {
	return filepath.Join(cacheDir, FileName)
}

// Listen consumes env, argv, then tails Cache/open.jsonl until ctx is done.
func Listen(ctx context.Context, appID string, handle func(Event) error) error {
	if handle == nil {
		return ErrNilHandler
	}
	d, err := dirs.Resolve(ctx, appID)
	if err != nil {
		return err
	}
	for _, ev := range Collect(os.Args[1:], os.Getenv(EnvOpen)) {
		if err := handle(ev); err != nil {
			return err
		}
	}
	return TailFile(ctx, FilePath(d.Cache), handle)
}

// Collect turns argv tokens and ELETROCROMO_OPEN into events.
func Collect(args []string, env string) []Event {
	var out []Event
	for _, tok := range tokens(args, env) {
		if ev, ok := Token(tok); ok {
			out = append(out, ev)
		}
	}
	return out
}

func tokens(args []string, env string) []string {
	var out []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		out = append(out, a)
	}
	for a := range strings.FieldsSeq(env) {
		out = append(out, a)
	}
	return out
}

// Token classifies one argv/env item.
func Token(s string) (Event, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Event{}, false
	}
	if strings.Contains(s, "://") {
		return Event{Kind: KindURL, URL: s}, true
	}
	if filepath.IsAbs(s) || fileLooksLikePath(s) {
		return Event{Kind: KindFiles, Paths: []string{s}}, true
	}
	return Event{}, false
}

func fileLooksLikePath(s string) bool {
	if _, err := os.Stat(s); err == nil {
		return true
	}
	return strings.ContainsAny(s, `/\`)
}

// ParseLine decodes one JSONL object.
func ParseLine(line []byte) (Event, error) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return Event{}, nil
	}
	var w wire
	if err := json.Unmarshal(line, &w); err != nil {
		return Event{}, fmt.Errorf("open line: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(w.Kind)) {
	case "url":
		if strings.TrimSpace(w.URL) == "" {
			return Event{}, ErrEmptyURL
		}
		return Event{Kind: KindURL, URL: w.URL}, nil
	case "files":
		if len(w.Paths) == 0 {
			return Event{}, ErrEmptyPaths
		}
		return Event{Kind: KindFiles, Paths: append([]string(nil), w.Paths...)}, nil
	default:
		return Event{}, fmt.Errorf("%w: %q", ErrUnknownKind, w.Kind)
	}
}

// TailFile reads existing lines then polls for appends.
func TailFile(ctx context.Context, path string, handle func(Event) error) error {
	var offset int64
	tick := time.NewTicker(80 * time.Millisecond)
	defer tick.Stop()
	for {
		n, err := consume(path, offset, handle)
		if err != nil {
			return err
		}
		offset = n
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}

func consume(path string, offset int64, handle func(Event) error) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return offset, nil
		}
		return offset, err
	}
	defer f.Close()
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return offset, err
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 4096), 1024*1024)
	for sc.Scan() {
		ev, err := ParseLine(sc.Bytes())
		if err != nil {
			return offset, err
		}
		if ev.Kind == 0 {
			offset += int64(len(sc.Bytes()) + 1)
			continue
		}
		if err := handle(ev); err != nil {
			return offset, err
		}
		offset += int64(len(sc.Bytes()) + 1)
	}
	return offset, sc.Err()
}
