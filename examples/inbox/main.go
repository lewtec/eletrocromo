// Inbox demos eletrocromo url/files capabilities and AppDirs.
//
//	mise run example:inbox
//	mise run ios:inbox:sim   # build, boot Simulator, install, ping
//	mise run ios:inbox:ping  # openurl + drop a .md into Cache/open.jsonl
package main

import (
	"context"
	"html/template"
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

var page = template.Must(template.New("inbox").Funcs(template.FuncMap{
	"fmtTime": func(t time.Time) string { return t.Format("15:04:05") },
}).Parse(`<!DOCTYPE html>
<html lang="en" data-theme="light">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta http-equiv="refresh" content="3">
  <title>Inbox</title>
  <link href="https://cdn.jsdelivr.net/npm/daisyui@5" rel="stylesheet" type="text/css" />
  <script src="https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4"></script>
</head>
<body class="min-h-screen">
  <div class="navbar bg-base-200">
    <div class="navbar-start">
      <span class="text-lg font-semibold px-2">Inbox</span>
    </div>
    <div class="navbar-end">
      <a class="btn btn-ghost btn-sm" href="/">reload</a>
    </div>
  </div>

  <main class="max-w-lg mx-auto p-4 flex flex-col gap-4">
    <div role="alert" class="alert">
      <span>Custom scheme <code>{{.Scheme}}://</code> and <code>.md</code> / <code>.txt</code> files. This page reloads every 3s.</span>
    </div>

    <div class="card card-border">
      <div class="card-body">
        <h2 class="card-title">Try it</h2>
        <div class="card-actions">
          <a class="btn btn-primary" href="{{.Scheme}}://from-webview">Open {{.Scheme}}://from-webview</a>
        </div>
        <form method="POST" action="/note" class="fieldset">
          <legend class="fieldset-legend">Write into Data</legend>
          <input class="input w-full" type="text" name="text" placeholder="a note" required>
          <button class="btn" type="submit">Save note</button>
        </form>
        <form method="POST" action="/share" class="fieldset">
          <legend class="fieldset-legend">Share out</legend>
          <input class="input w-full" type="text" name="text" placeholder="text to share" required>
          <button class="btn" type="submit">Share</button>
        </form>
      </div>
    </div>

    <div class="card card-border">
      <div class="card-body">
        <h2 class="card-title">Dirs</h2>
        <ul class="list">
          <li class="list-row"><span>Data</span><span class="list-col-grow font-mono text-xs break-all">{{.Dirs.Data}}</span></li>
          <li class="list-row"><span>Cache</span><span class="list-col-grow font-mono text-xs break-all">{{.Dirs.Cache}}</span></li>
          <li class="list-row"><span>Config</span><span class="list-col-grow font-mono text-xs break-all">{{.Dirs.Config}}</span></li>
          <li class="list-row"><span>Inbox</span><span class="list-col-grow font-mono text-xs break-all">{{.Dirs.Inbox}}</span></li>
        </ul>
      </div>
    </div>

    <div class="card card-border">
      <div class="card-body">
        <h2 class="card-title">Opens</h2>
        {{if not .Events}}
          <p class="opacity-70">None yet. From the Mac: <code>mise run ios:inbox:ping</code></p>
        {{else}}
          <ul class="list">
            {{range .Events}}
              <li class="list-row">
                <span class="badge">{{.Kind}}</span>
                <div class="list-col-grow">
                  <div class="font-mono text-sm break-all">{{.Text}}</div>
                  <div class="text-xs opacity-70">{{fmtTime .When}}</div>
                  {{if .Body}}<pre class="mt-2 text-xs whitespace-pre-wrap">{{.Body}}</pre>{{end}}
                </div>
              </li>
            {{end}}
          </ul>
        {{end}}
      </div>
    </div>

    <div class="card card-border">
      <div class="card-body">
        <h2 class="card-title">Inbox files</h2>
        {{if not .Inbox}}
          <p class="opacity-70">Empty</p>
        {{else}}
          <ul class="list">
            {{range .Inbox}}
              <li class="list-row font-mono text-sm break-all">{{.}}</li>
            {{end}}
          </ul>
        {{end}}
      </div>
    </div>
  </main>
</body>
</html>
`))

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
		err = page.Execute(w, map[string]any{
			"Scheme": scheme,
			"Dirs":   d,
			"Events": st.list(),
			"Inbox":  inboxNames(d.Inbox),
		})
		if err != nil {
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
