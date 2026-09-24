package eletrocromo

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"runtime"

	"github.com/lewtec/lewkit/x/driver/webview"
	_ "github.com/lewtec/lewkit/x/driver/webview/webkitgtk"
	_ "github.com/lewtec/lewkit/x/driver/webview/webview2"
	_ "github.com/lewtec/lewkit/x/driver/webview/wkwebview"
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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.AddCookie(&http.Cookie{Name: AUTH_COOKIE_KEY, Value: a.AuthToken})
		a.ServeHTTP(w, r)
	})
}

// urlHandler forwards view requests to an already-running http(s) origin.
// The base query (token handshake) is kept when the view request has none.
func urlHandler(base *url.URL) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	})
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
