package eletrocromo

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/webview"
)

var errNoDisplay = errors.New("no display")

type fakeView struct {
	done chan struct{}
	once sync.Once
}

func (v *fakeView) Done() <-chan struct{} { return v.done }

func (v *fakeView) Close() error {
	v.once.Do(func() { close(v.done) })
	return nil
}

func TestRun_RequiresAppID(t *testing.T) {
	app := App{
		Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		Context: t.Context(),
	}
	if err := app.Run(); err == nil {
		t.Fatal("expected error for missing App.ID")
	}
}

func TestRun_DesktopOpenError(t *testing.T) {
	orig := openDesktopView
	t.Cleanup(func() { openDesktopView = orig })
	openDesktopView = func(context.Context, webview.Config) (appView, error) {
		return nil, errNoDisplay
	}
	app := App{
		ID:      "br.tec.lew.test.open",
		Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		Context: t.Context(),
	}
	err := app.Run()
	if !errors.Is(err, errNoDisplay) {
		t.Fatalf("got %v", err)
	}
}

func TestRun_DesktopServesHandler(t *testing.T) {
	orig := openDesktopView
	t.Cleanup(func() { openDesktopView = orig })
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	opened := make(chan webview.Config, 1)
	openDesktopView = func(ctx context.Context, cfg webview.Config) (appView, error) {
		opened <- cfg
		v := &fakeView{done: make(chan struct{})}
		go func() {
			<-ctx.Done()
			if err := v.Close(); err != nil {
				t.Errorf("close: %v", err)
			}
		}()
		return v, nil
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	app := App{
		ID:      "br.tec.lew.test.window",
		Context: ctx,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := io.WriteString(w, "pong"); err != nil {
				t.Errorf("write: %v", err)
			}
		}),
	}
	errCh := make(chan error, 1)
	go func() { errCh <- app.Run() }()

	var cfg webview.Config
	select {
	case cfg = <-opened:
	case <-time.After(2 * time.Second):
		t.Fatal("window was not opened")
	}
	if cfg.Profile == "" {
		t.Fatal("profile dir was empty")
	}
	rec := httptest.NewRecorder()
	cfg.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "pong" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return")
	}
}

func TestURLHandler_ForwardsPathAndQuery(t *testing.T) {
	var got string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.RequestURI()
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(upstream.Close)

	base, err := http.NewRequest(http.MethodGet, upstream.URL+"/?token=abc", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := urlHandler(base.URL)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/count", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d", rec.Code)
	}
	if got != "/count?token=abc" {
		t.Fatalf("upstream URI %q", got)
	}
}
