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
	"time"

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

// BackgroundRun executes a given task in a separate goroutine.
// It manages the App's WaitGroup to ensure all background tasks are completed
// before the application shuts down.
func (a *App) BackgroundRun(task Task) error {
	a.WaitGroup.Add(1)
	defer a.WaitGroup.Done()
	return task.Run(a.Context)
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
// 1. Generates a new random AuthToken if one is not already set.
// 2. Starts an internal HTTP server (using httptest.Server for simplified port management).
// 3. Launches the Chromium browser pointing to the server's URL with the auth token.
// 4. Waits for the application context to be cancelled.
//
// It also starts a heartbeat task to keep the process alive/monitored if needed.
func (a *App) Run() error {
	if a.AuthToken == "" {
		a.AuthToken = uuid.New().String()
	}
	if a.Context == nil {
		a.Context = context.Background()
	}
	ctx, cancel := context.WithCancel(a.Context)
	defer cancel()

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

	go func() {
		_ = a.BackgroundRun(NewKeepAliveTask(5 * time.Second))
	}()
	go func() {
		_ = a.BackgroundRun(NewBrowserLaunchTask(link))
	}()
	time.Sleep(time.Second)
	a.WaitGroup.Wait()
	return nil
}
