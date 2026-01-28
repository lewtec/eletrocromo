package eletrocromo

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
)

type App struct {
	Handler   http.Handler
	AuthToken string
	WaitGroup sync.WaitGroup
	Context   context.Context
}

const AUTH_COOKIE_KEY = "eletrocromo_token"

func (a *App) BackgroundRun(task Task) error {
	a.WaitGroup.Add(1)
	defer a.WaitGroup.Done()
	return task.Run(a.Context)
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token != "" {
		if token == a.AuthToken {
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
	if token != a.AuthToken {
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
		_ = a.BackgroundRun(FunctionTask(func(ctx context.Context) error {
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
			}
			return nil
		},
		))
	}()
	go func() {
		_ = a.BackgroundRun(
			FunctionTask(func(ctx context.Context) error {
				u, err := url.Parse(link)
				if err != nil {
					return err
				}
				return LaunchChromium(u)
			}),
		)
	}()
	time.Sleep(time.Second)
	a.WaitGroup.Wait()
	return nil
}

