package eletrocromo

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"runtime"

	"github.com/lewtec/lewkit/x/driver/webview"
	_ "github.com/lewtec/lewkit/x/driver/webview/prelude"
	"github.com/lewtec/lewkit/x/thread"
)

// appView is the desktop window. Tests substitute openDesktopView.
type appView interface {
	Done() <-chan struct{}
	Close() error
}

// openDesktopView opens the system web view. Tests may replace it.
var openDesktopView = openSystemView

func openSystemView(ctx context.Context, cfg webview.Config) (appView, error) {
	if runtime.GOOS == "darwin" && !thread.Bound() {
		// WKWebView events must run on the process main thread.
		// Call App.Run from main on macOS.
		thread.Bind()
	}
	return webview.Open(ctx, cfg)
}

// windowHandler serves the app inside the web view. The view does not
// listen on a port, so each request is stamped with the session cookie
// before the usual auth check.
func (a *App) windowHandler() http.Handler {
	return adaptViewRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.AddCookie(&http.Cookie{Name: AUTH_COOKIE_KEY, Value: a.AuthToken})
		a.ServeHTTP(w, r)
	}))
}

// adaptViewRequest turns an app://viewN/path request into a loopback path
// the app already knows how to route. The web view does not send headers,
// so a body with no Content-Type is treated as a form. Redirects are written
// back onto the view origin; a Location of "/" would otherwise leave it.
func adaptViewRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := requestOrigin(r)
		normalizeViewRequest(r)
		if r.Header.Get("Content-Type") == "" && requestHasBody(r.Method) {
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		next.ServeHTTP(&viewResponse{ResponseWriter: w, origin: origin}, r)
	})
}

func requestOrigin(r *http.Request) *url.URL {
	if r.URL == nil || r.URL.Scheme == "" || r.URL.Host == "" {
		return &url.URL{Scheme: "app", Host: "view"}
	}
	return &url.URL{Scheme: r.URL.Scheme, Host: r.URL.Host}
}

func normalizeViewRequest(r *http.Request) {
	if r.URL == nil {
		r.URL = &url.URL{Path: "/"}
	}
	u := *r.URL
	if u.Path == "" {
		u.Path = "/"
	}
	u.Scheme = "http"
	u.Host = "127.0.0.1"
	r.URL = &u
	r.Host = u.Host
	r.RequestURI = u.RequestURI()
}

func requestHasBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}

type viewResponse struct {
	http.ResponseWriter
	origin *url.URL
	wrote  bool
}

func (w *viewResponse) WriteHeader(status int) {
	if w.wrote {
		return
	}
	w.wrote = true
	if loc := w.Header().Get("Location"); loc != "" {
		w.Header().Set("Location", rewriteLocation(w.origin, loc))
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *viewResponse) Write(payload []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(payload)
}

func rewriteLocation(origin *url.URL, loc string) string {
	u, err := url.Parse(loc)
	if err != nil || origin == nil || origin.Host == "" {
		return loc
	}
	if u.IsAbs() && !sameViewHost(u) {
		return loc
	}
	if u.Path == "" {
		u.Path = "/"
	}
	u.Scheme = origin.Scheme
	u.Host = origin.Host
	return u.String()
}

func sameViewHost(u *url.URL) bool {
	host := u.Hostname()
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

// urlHandler forwards view requests to an already-running http(s) origin.
// The base query (token handshake) is kept when the view request has none.
func urlHandler(base *url.URL) http.Handler {
	return adaptViewRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dest := *base
		if r.URL.Path != "" {
			dest.Path = r.URL.Path
		}
		if r.URL.RawQuery != "" {
			dest.RawQuery = r.URL.RawQuery
		}
		req, err := http.NewRequestWithContext(r.Context(), r.Method, dest.String(), r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		if _, err := io.Copy(w, resp.Body); err != nil {
			return
		}
	}))
}

// waitDesktopView blocks until ctx is cancelled or the window closes.
// On macOS, when Run is on the bound main thread, this pumps AppKit.
func waitDesktopView(ctx context.Context, view appView) {
	if runtime.GOOS == "darwin" && thread.On() {
		loopCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		go func() {
			select {
			case <-view.Done():
			case <-ctx.Done():
			}
			cancel()
		}()
		thread.Loop(loopCtx)
		return
	}
	select {
	case <-ctx.Done():
	case <-view.Done():
	}
}
