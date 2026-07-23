package eletrocromo

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// App acts as the core controller for the application, managing the lifecycle,
// authentication state, and the internal web server.
// It coordinates background tasks and ensures graceful shutdown.
type App struct {
	// ID is the reverse-domain application identity (e.g. "br.tec.lew.counter").
	// Required. Isolates the Helium --user-data-dir per app and is the intended
	// APK package name when packaging is added later.
	ID string

	Handler   http.Handler
	AuthToken string
	WaitGroup sync.WaitGroup
	Context   context.Context

	// NoUI skips Helium resolve/launch and only serves loopback HTTP.
	// Used by the Android WebView host (and tests). Also enabled when
	// ELETROCROMO_NO_UI is 1/true/yes.
	NoUI bool
}

// ReadyLinePrefix is printed once the loopback server is listening in NoUI mode.
// The Android shell parses the following URL (token included) to open WebView.
//
//	ELETROCROMO_READY http://127.0.0.1:PORT/?token=UUID
const ReadyLinePrefix = "ELETROCROMO_READY "

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
// Startup Sequence (desktop):
//  1. Validates App.ID (reverse-domain) and prepares an isolated Helium profile.
//  2. Generates a new random AuthToken if one is not already set.
//  3. Resolves Helium (local PATH or workspaced ensure) — before binding any port.
//  4. Starts the internal HTTP server (httptest for ephemeral loopback bind).
//  5. Launches Helium with --user-data-dir + --app; fails Run if the process
//     exits during a short startup grace (launch failures are not ignored).
//  6. Blocks until the context is cancelled (including Helium exit), then waits
//     for background tasks and shuts down the server.
//
// NoUI / ELETROCROMO_NO_UI: skip Helium; bind, print ReadyLinePrefix + URL, wait.
func (a *App) Run() error {
	if err := ValidateAppID(a.ID); err != nil {
		return err
	}

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

	noUI := a.NoUI || noUIEnabled()

	var profileDir string
	var bin string
	if !noUI {
		var err error
		profileDir, err = ProfileDir(a.ID)
		if err != nil {
			return err
		}
		// Resolve Helium first: ensure can take a long time (download workspaced +
		// helium-browser). Do not open a listening server until we know we can
		// open a window; failures must not leave a loopback port up with a token.
		log.Printf("resolving Helium host…")
		bin, err = resolveBrowserHost(ctx)
		if err != nil {
			return err
		}
		log.Printf("Helium host: %s (profile %s)", bin, profileDir)
	}

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
	// Prefer 127.0.0.1 host for Android WebView + networkSecurityConfig (not [::1]).
	base := ts.URL
	if u, err := url.Parse(ts.URL); err == nil {
		host := u.Hostname()
		if host == "" || host == "localhost" || host == "::1" {
			u.Host = net.JoinHostPort("127.0.0.1", u.Port())
			base = u.String()
		}
	}
	link := fmt.Sprintf("%s/?token=%s", strings.TrimRight(base, "/"), a.AuthToken)
	log.Printf("webserver started on %s", link)

	if noUI {
		// Machine-parseable line on stdout without log timestamps (Android host).
		// Also log for humans / tests that capture log.Writer().
		fmt.Fprintln(os.Stdout, ReadyLinePrefix+link)
		log.Print(ReadyLinePrefix + link)
		// Optional side channel: write the URL to a file (stdout can block or be
		// lost under ProcessBuilder; Android shell sets ELETROCROMO_READY_FILE).
		if path := strings.TrimSpace(os.Getenv("ELETROCROMO_READY_FILE")); path != "" {
			if err := os.WriteFile(path, []byte(link+"\n"), 0o600); err != nil {
				log.Printf("ELETROCROMO_READY_FILE: %v", err)
			}
		}
		<-ctx.Done()
		a.WaitGroup.Wait()
		return nil
	}

	win, err := startAppWindow(bin, link, profileDir)
	if err != nil {
		cancel()
		a.WaitGroup.Wait()
		return fmt.Errorf("launch Helium: %w", err)
	}
	if err := win.awaitStartup(heliumStartupGrace); err != nil {
		win.stop()
		cancel()
		a.WaitGroup.Wait()
		return err
	}
	// After a healthy start, Helium exit cancels the app (window-owned lite).
	win.watchExit(func(exitErr error) {
		if exitErr != nil {
			log.Printf("Helium exited: %v", exitErr)
		} else {
			log.Printf("Helium exited")
		}
		cancel()
	})

	<-ctx.Done()
	// Ctrl+C / parent cancel: tear down the process group so helpers do not leak.
	win.stop()
	a.WaitGroup.Wait()
	return nil
}

func noUIEnabled() bool {
	v := strings.TrimSpace(os.Getenv("ELETROCROMO_NO_UI"))
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}
