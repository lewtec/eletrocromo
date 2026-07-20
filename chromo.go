package eletrocromo

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/google/uuid"
)

// App acts as the core controller for the application, managing the lifecycle,
// authentication state, and the internal web server.
// It coordinates background tasks and ensures graceful shutdown.
type App struct {
	Handler   http.Handler
	AuthToken string
	WaitGroup sync.WaitGroup
	Context   context.Context
}

const AUTH_COOKIE_KEY = "eletrocromo_token"

// BackgroundRun starts task in a new goroutine and tracks it on WaitGroup.
// It returns immediately after scheduling; task errors are logged.
// Callers must not wrap BackgroundRun in another goroutine — Add runs
// synchronously so WaitGroup.Wait is race-free with respect to this call.
// A nil App.Context is treated as context.Background(), matching Run.
func (a *App) BackgroundRun(task Task) error {
	ctx := a.Context
	if ctx == nil {
		ctx = context.Background()
	}
	a.WaitGroup.Add(1)
	go func() {
		defer a.WaitGroup.Done()
		if err := task.Run(ctx); err != nil {
			log.Printf("background task: %v", err)
		}
	}()
	return nil
}

// ServeHTTP handles incoming HTTP requests with authentication enforcement.
//
// Authentication Flow:
// 1. Checks for an authentication token in the URL query parameters (used for the initial handshake).
// 2. If present and valid, sets a strict, HttpOnly cookie with the token.
// 3. If no query token is present, falls back to checking the cookie.
//
// Security Policy:
// - Fail Closed: If the token is invalid or missing, returns 401 Unauthorized.
// - If no internal Handler is configured, returns 404 Not Found.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token != "" {
		if subtle.ConstantTimeCompare([]byte(token), []byte(a.AuthToken)) == 1 {
			http.SetCookie(w, &http.Cookie{
				Name:     AUTH_COOKIE_KEY,
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})
		}
	} else {
		cookie, _ := r.Cookie(AUTH_COOKIE_KEY)
		if cookie != nil {
			token = cookie.Value
		}

	}
	// Fail closed when AuthToken is unset: ConstantTimeCompare("", "") would
	// otherwise accept unauthenticated requests (ServeHTTP without Run).
	if a.AuthToken == "" || subtle.ConstantTimeCompare([]byte(token), []byte(a.AuthToken)) != 1 {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprintf(w, "forbidden")
		return
	}
	if a.Handler == nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprintf(w, "no handler setup")
		return
	}
	a.Handler.ServeHTTP(w, r)
}

// Run starts the application and blocks until the context is cancelled.
//
// Startup Sequence:
//  1. Generates a new random AuthToken if one is not already set.
//  2. Resolves Helium (local PATH or workspaced ensure) — before binding any port.
//  3. Starts the internal HTTP server (httptest for ephemeral loopback bind).
//  4. Launches Helium with --app pointing at the server URL + auth token.
//  5. Blocks until the application context is cancelled, then waits for background
//     tasks and shuts down the server.
//
// Ensure runs synchronously on the startup path so a missing/unresolvable host
// fails Run before the webserver is advertised. Launch still uses Start() so
// the browser process is not waited on here (window-owned lifetime is later).
func (a *App) Run() error {
	if a.AuthToken == "" {
		a.AuthToken = uuid.New().String()
	}
	if a.Context == nil {
		a.Context = context.Background()
	}
	ctx, cancel := context.WithCancel(a.Context)
	defer cancel()

	// Background tasks and the HTTP server share the derived context so they
	// stop together when the parent context is cancelled.
	prevCtx := a.Context
	a.Context = ctx
	defer func() { a.Context = prevCtx }()

	// Resolve Helium first: ensure can take a long time (download workspaced +
	// helium-browser). Do not open a listening server until we know we can
	// open a window; failures must not leave a loopback port up with a token.
	log.Printf("resolving Helium host…")
	bin, err := resolveBrowserHost(ctx)
	if err != nil {
		return err
	}
	log.Printf("Helium host: %s", bin)

	ts := httptest.NewUnstartedServer(a)
	ts.Config.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}

	started := make(chan struct{})
	go func() {
		ts.Start()
		close(started)
		<-ctx.Done()
		ts.Close()
	}()

	<-started
	link := fmt.Sprintf("%s/?token=%s", ts.URL, a.AuthToken)
	log.Printf("webserver started on %s", link)

	if err := launchAppWindow(bin, link); err != nil {
		cancel()
		a.WaitGroup.Wait()
		return fmt.Errorf("launch Helium: %w", err)
	}

	<-ctx.Done()
	a.WaitGroup.Wait()
	return nil
}
