package eletrocromo

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

type App struct {
	Handler   http.Handler
	AuthToken string
}

const AUTH_COOKIE_KEY = "eletrocromo_token"

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
	a.ServeHTTP(w, r)
}

func (a *App) Run(ctx context.Context) error {
	if a.AuthToken == "" {
		a.AuthToken = uuid.New()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// TODO: setup webserver
	// TODO: launch chromium to the webserver
	return nil
}

func LaunchChromium(url string) error {
	// TODO: implement --app
	// TODO: implement xdg-open like as fallback
	return nil
}
