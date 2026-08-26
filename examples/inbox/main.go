// Inbox demos eletrocromo url/files capabilities and AppDirs.
//
//	mise run example:inbox
//	mise run ios:inbox:sim   # build, boot Simulator, install, ping
//	mise run ios:inbox:ping  # openurl + drop a .md into Cache/open.jsonl
package main

import (
	"context"
	"io"
	"log"
	"net/http"
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
		item.Kind = "url"
		item.Text = ev.URL
	case open.KindFiles:
		item.Kind = "files"
		item.Text = strings.Join(ev.Paths, ", ")
		if len(ev.Paths) > 0 {
			item.Body = readHead(ev.Paths[0], 2048)
		}
	default:
		return nil
	}
	s.add(item)
	return nil
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
		if err := page(scheme, d, st.list(), inboxNames(d.Inbox)).Render(r.Context(), w); err != nil {
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
