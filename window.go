//go:build !android

package eletrocromo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
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

// runDesktop opens the system web view and blocks until it closes or ctx ends.
// The handler runs in-process. There is no loopback listener on this path.
func (a *App) runDesktop(ctx context.Context, cancel context.CancelFunc) error {
	profileDir, err := ProfileDir(a.ID)
	if err != nil {
		return err
	}
	log.Printf("opening web view (profile %s)", profileDir)
	view, err := openDesktopView(ctx, webview.Config{
		Profile: profileDir,
		Handler: a.windowHandler(),
	})
	if err != nil {
		cancel()
		a.WaitGroup.Wait()
		return fmt.Errorf("open web view: %w", err)
	}
	go func() {
		select {
		case <-view.Done():
			log.Printf("web view closed")
			cancel()
		case <-ctx.Done():
		}
	}()
	waitDesktopView(ctx, view)
	if err := view.Close(); err != nil {
		log.Printf("close web view: %v", err)
	}
	cancel()
	a.WaitGroup.Wait()
	return nil
}

func launchBrowserURL(ctx context.Context, urlStr, appID string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("parse app url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: %s", errInvalidURLScheme, u.Scheme)
	}
	profileDir, err := ProfileDir(appID)
	if err != nil {
		return err
	}
	view, err := openDesktopView(ctx, webview.Config{
		Profile: profileDir,
		Handler: urlHandler(u),
	})
	if err != nil {
		return err
	}
	go func() {
		select {
		case <-ctx.Done():
			if err := view.Close(); err != nil {
				log.Printf("close web view: %v", err)
			}
		case <-view.Done():
		}
	}()
	return nil
}

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
		rec := serveView(next, r)
		for range 8 {
			if !isRedirect(rec.status) {
				break
			}
			target, ok := sameOriginTarget(origin, r, rec.header.Get("Location"))
			if !ok {
				break
			}
			r.Method = http.MethodGet
			r.URL = target
			r.Host = target.Host
			r.RequestURI = target.RequestURI()
			r.Body = http.NoBody
			r.ContentLength = 0
			r.Header.Del("Content-Type")
			rec = serveView(next, r)
		}
		out := &viewResponse{ResponseWriter: w, origin: origin}
		for key, values := range rec.header {
			for _, value := range values {
				out.Header().Add(key, value)
			}
		}
		out.WriteHeader(rec.status)
		if rec.body.Len() > 0 {
			_, _ = out.Write(rec.body.Bytes())
		}
	})
}

type bufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
	wrote  bool
}

func (b *bufferedResponse) Header() http.Header {
	if b.header == nil {
		b.header = make(http.Header)
	}
	return b.header
}

func (b *bufferedResponse) WriteHeader(status int) {
	if b.wrote {
		return
	}
	b.wrote = true
	b.status = status
}

func (b *bufferedResponse) Write(payload []byte) (int, error) {
	if !b.wrote {
		b.WriteHeader(http.StatusOK)
	}
	return b.body.Write(payload)
}

func serveView(next http.Handler, r *http.Request) *bufferedResponse {
	rec := &bufferedResponse{status: http.StatusOK}
	next.ServeHTTP(rec, r)
	if !rec.wrote {
		rec.WriteHeader(http.StatusOK)
	}
	return rec
}

func isRedirect(status int) bool {
	return status == http.StatusMovedPermanently ||
		status == http.StatusFound ||
		status == http.StatusSeeOther ||
		status == http.StatusTemporaryRedirect ||
		status == http.StatusPermanentRedirect
}

func sameOriginTarget(origin *url.URL, current *http.Request, loc string) (*url.URL, bool) {
	u, err := url.Parse(loc)
	if err != nil {
		return nil, false
	}
	if u.IsAbs() && !sameViewHost(u) && (origin == nil || u.Scheme != origin.Scheme || u.Host != origin.Host) {
		return nil, false
	}
	if !u.IsAbs() && (u.Path == "" || u.Path[0] != '/') && current != nil && current.URL != nil {
		u = current.URL.ResolveReference(u)
	}
	path := u.Path
	if path == "" {
		path = "/"
	}
	return &url.URL{Scheme: "http", Host: "127.0.0.1", Path: path, RawQuery: u.RawQuery}, true
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
