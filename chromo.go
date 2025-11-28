package eletrocromo

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
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
		fmt.Fprintf(w, "forbidden")
		return
	}
	if a.Handler == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "no handler setup")
		return
	}
	a.Handler.ServeHTTP(w, r)
}

func (a *App) Run() error {
	if a.AuthToken == "" {
		a.AuthToken = uuid.New().String()
	}
	ctx, cancel := context.WithCancel(a.Context)
	defer cancel()
	freePort, err := findFreePort()
	addr := fmt.Sprintf("127.0.0.1:%d", freePort)
	link := fmt.Sprintf("http://%s/?token=%s", addr, a.AuthToken)
	if err != nil {
		return err
	}
	server := http.Server{
		Handler: a,
		Addr:    addr,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
	go a.BackgroundRun(FunctionTask(func(ctx context.Context) error {
		time.Sleep(5 * time.Second)
		return nil
	},
	))
	go a.BackgroundRun(
		FunctionTask(func(ctx context.Context) error {
			log.Printf("webserver started on %s", link)
			err := server.ListenAndServe()
			if err != nil {
				panic(err)
			}
			return server.Close()
		},
		),
	)
	go a.BackgroundRun(
		FunctionTask(func(ctx context.Context) error {
			return LaunchChromium(link)
		}),
	)
	time.Sleep(time.Second)
	a.WaitGroup.Wait()
	return nil
}

func findFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}
