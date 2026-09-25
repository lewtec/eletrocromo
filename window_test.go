package eletrocromo

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/webview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.Error(t, app.Run())
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
	require.ErrorIs(t, app.Run(), errNoDisplay)
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
			require.NoError(t, v.Close())
		}()
		return v, nil
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	app := App{
		ID:      "br.tec.lew.test.window",
		Context: ctx,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := io.WriteString(w, "pong")
			require.NoError(t, err)
		}),
	}
	errCh := make(chan error, 1)
	go func() { errCh <- app.Run() }()

	var cfg webview.Config
	select {
	case cfg = <-opened:
	case <-time.After(2 * time.Second):
		require.Fail(t, "window was not opened")
	}
	require.NotEmpty(t, cfg.Profile)
	rec := httptest.NewRecorder()
	cfg.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "pong", rec.Body.String())

	cancel()
	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.Fail(t, "Run did not return")
	}
}

func TestWindowHandler_RoutesViewURL(t *testing.T) {
	var paths []string
	var seenOp, seenHost string
	app := &App{
		AuthToken: "secret",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			paths = append(paths, r.URL.Path)
			seenHost = r.Host
			require.NoError(t, r.ParseForm())
			if op := r.Form.Get("op"); op != "" {
				seenOp = op
			}
			switch r.URL.Path {
			case "/go":
				http.Redirect(w, r, "/", http.StatusSeeOther)
			case "/away":
				http.Redirect(w, r, "https://example.com/docs", http.StatusSeeOther)
			default:
				w.Header().Set("Content-Type", "text/html")
				_, err := io.WriteString(w, "page "+r.URL.Path)
				require.NoError(t, err)
			}
		}),
	}
	handler := app.windowHandler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "app://view9/go", strings.NewReader("op=inc"))
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("Location"))
	assert.Equal(t, "page /", rec.Body.String())
	assert.Equal(t, "inc", seenOp)
	assert.Equal(t, "127.0.0.1", seenHost)
	assert.Equal(t, []string{"/go", "/"}, paths)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "app://view9/away", nil)
	handler.ServeHTTP(rec, req)
	assert.Equal(t, "https://example.com/docs", rec.Header().Get("Location"))

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "app://view9", nil)
	handler.ServeHTTP(rec, req)
	assert.Equal(t, "/", paths[len(paths)-1])
	assert.Equal(t, "page /", rec.Body.String())
}

func TestURLHandler_ForwardsPathAndQuery(t *testing.T) {
	var got string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.RequestURI()
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(upstream.Close)

	base, err := http.NewRequest(http.MethodGet, upstream.URL+"/?token=abc", nil)
	require.NoError(t, err)
	handler := urlHandler(base.URL)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/count", nil))
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "/count?token=abc", got)
}
