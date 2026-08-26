// Inbox demos eletrocromo url/files capabilities and AppDirs.
//
//	mise run example:inbox
//	mise run ios:inbox:sim   # build, boot Simulator, install, ping
//	mise run ios:inbox:ping  # openurl + drop a .md into Cache/open.jsonl
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lewtec/eletrocromo"
	"github.com/lewtec/eletrocromo/driver/dirs"
	_ "github.com/lewtec/eletrocromo/driver/dirs/os"
	"github.com/lewtec/eletrocromo/driver/open"
	"github.com/lewtec/eletrocromo/driver/share"
	_ "github.com/lewtec/eletrocromo/driver/share/desktop"
	_ "github.com/lewtec/eletrocromo/driver/share/hostfile"
)

const (
	appID  = "br.tec.lew.eletrocromo.inbox"
	scheme = "eletrocromo-inbox"
)

type event struct {
	When time.Time
	Kind string
	Text string
	Body string
}

type state struct {
	mu     sync.Mutex
	events []event
}

func (s *state) add(ev event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append([]event{ev}, s.events...)
	if len(s.events) > 20 {
		s.events = s.events[:20]
	}
}

func (s *state) list() []event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]event, len(s.events))
	copy(out, s.events)
	return out
}

func (s *state) onOpen(ev open.Event) error {
	item := event{When: time.Now()}
	switch ev.Kind {
	case open.KindURL:
		if name := inboxNameFromURL(ev.URL); name != "" {
			if path, ok := inboxFile(name); ok {
				item.Kind = "url"
				item.Text = name
				item.Body = describeURL(ev.URL) + "\n" + path
				s.add(item)
				return nil
			}
		}
		item.Kind = "url"
		item.Text = ev.URL
		item.Body = describeURL(ev.URL)
	case open.KindFiles:
		item.Kind = "files"
		item.Text = strings.Join(ev.Paths, ", ")
		if len(ev.Paths) > 0 && looksText(ev.Paths[0]) {
			item.Body = readHead(ev.Paths[0], 2048)
		}
	default:
		return nil
	}
	s.add(item)
	return nil
}

func inboxNameFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	for _, key := range []string{"name", "file", "path"} {
		if n := sanitizeName(u.Query().Get(key)); n != "" {
			return n
		}
	}
	if u.Host == "inbox" {
		return sanitizeName(strings.TrimPrefix(u.Path, "/"))
	}
	if rest, ok := strings.CutPrefix(strings.TrimPrefix(u.Path, "/"), "inbox/"); ok {
		return sanitizeName(rest)
	}
	if u.Opaque != "" {
		if rest, ok := strings.CutPrefix(u.Opaque, "inbox/"); ok {
			return sanitizeName(rest)
		}
	}
	return ""
}

func sanitizeName(n string) string {
	n = filepath.Base(strings.TrimSpace(n))
	if n == "." || n == ".." || n == "" {
		return ""
	}
	return n
}

func inboxFile(name string) (string, bool) {
	d, err := dirs.Resolve(context.Background(), appID)
	if err != nil {
		return "", false
	}
	path := filepath.Join(d.Inbox, name)
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	return path, true
}

func describeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "parse: " + err.Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "scheme=%s", u.Scheme)
	if u.Opaque != "" {
		fmt.Fprintf(&b, " opaque=%s", u.Opaque)
	}
	if u.Host != "" {
		fmt.Fprintf(&b, " host=%s", u.Host)
	}
	if u.Path != "" {
		fmt.Fprintf(&b, " path=%s", u.Path)
	}
	if u.RawQuery != "" {
		fmt.Fprintf(&b, " query=%s", u.RawQuery)
	}
	if u.Fragment != "" {
		fmt.Fprintf(&b, " frag=%s", u.Fragment)
	}
	return b.String()
}

func looksText(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".txt", ".json", ".csv", ".xml", ".html", ".htm", ".go":
		return true
	default:
		return false
	}
}

type urlProbe struct {
	Href  string
	Label string
}

func urlProbes(scheme string, names []string) []urlProbe {
	out := []urlProbe{
		{scheme + "://from-webview", "from-webview"},
		{scheme + "://", "empty host"},
		{scheme + "://x/y?q=1&empty=&sp=a+b#frag", "query+frag"},
		{scheme + "://caf%C3%A9/%E6%97%A5%E6%9C%AC%E8%AA%9E", "unicode"},
		{scheme + ":opaque-rest", "opaque"},
		{scheme + "://inbox?name=missing.bin", "missing name"},
	}
	if len(names) > 0 {
		out = append(out, urlProbe{
			Href:  scheme + "://inbox?name=" + url.QueryEscape(names[0]),
			Label: "name=" + names[0],
		})
	}
	return out
}

func readHead(path string, n int) string {
	f, err := os.Open(path)
	if err != nil {
		return err.Error()
	}
	defer f.Close()
	buf := make([]byte, n+1)
	got, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return err.Error()
	}
	text := string(buf[:min(got, n)])
	if got > n {
		text += "…"
	}
	return text
}

func inboxNames(inbox string) []string {
	ents, err := os.ReadDir(inbox)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range ents {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

//go:generate go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	st := &state{}
	mux := http.NewServeMux()
	mux.HandleFunc("/note", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		text := strings.TrimSpace(r.Form.Get("text"))
		if text == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		d, err := dirs.Resolve(r.Context(), appID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		notes := filepath.Join(d.Data, "notes")
		if err := os.MkdirAll(notes, 0o700); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		name := time.Now().Format("20060102-150405") + ".txt"
		if err := os.WriteFile(filepath.Join(notes, name), []byte(text+"\n"), 0o600); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		st.add(event{When: time.Now(), Kind: "note", Text: name, Body: text})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("/share", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		text := strings.TrimSpace(r.Form.Get("text"))
		if text == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if err := share.Out(r.Context(), share.Item{Text: text}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		st.add(event{When: time.Now(), Kind: "share", Text: text})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("/share-file", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		name := filepath.Base(strings.TrimSpace(r.Form.Get("name")))
		if name == "." || name == ".." {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		d, err := dirs.Resolve(r.Context(), appID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		path := filepath.Join(d.Inbox, name)
		if err := share.Out(r.Context(), share.Item{Paths: []string{path}}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		st.add(event{When: time.Now(), Kind: "share", Text: name})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("/open", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		raw := strings.TrimSpace(r.Form.Get("url"))
		if raw == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		ev, ok := open.Token(raw)
		if !ok {
			ev = open.Event{Kind: open.KindURL, URL: raw}
		}
		_ = st.onOpen(ev)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		d, err := dirs.Resolve(r.Context(), appID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		names := inboxNames(d.Inbox)
		if err := page(scheme, d, st.list(), names, urlProbes(scheme, names)).Render(r.Context(), w); err != nil {
			log.Printf("template: %v", err)
		}
	})

	app := eletrocromo.App{
		ID:      appID,
		Handler: mux,
		Context: ctx,
		OnOpen:  st.onOpen,
	}
	log.Printf("inbox example: %s://  (md/txt files)", scheme)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
