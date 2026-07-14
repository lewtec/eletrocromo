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
func (a *App) BackgroundRun(task Task) error {
	a.WaitGroup.Add(1)
	go func() {
		defer a.WaitGroup.Done()
		if err := task.Run(a.Context); err != nil {
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
	if subtle.ConstantTimeCompare([]byte(token), []byte(a.AuthToken)) != 1 {
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
//  2. Starts an internal HTTP server (using httptest.Server for simplified port management).
//  3. Launches the Chromium browser pointing to the server's URL with the auth token.
//  4. Blocks until the application context is cancelled, then waits for background
//     tasks and shuts down the server.
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

	if err := a.BackgroundRun(NewBrowserLaunchTask(link)); err != nil {
		return err
	}

	<-ctx.Done()
	a.WaitGroup.Wait()
	return nil
}
