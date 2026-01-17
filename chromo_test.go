package eletrocromo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApp_ServeHTTP_Auth(t *testing.T) {
	token := "secret-token-123"
	app := &App{
		AuthToken: token,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "success")
		}),
		Context: context.Background(),
	}

	tests := []struct {
		name           string
		queryToken     string
		cookieToken    string
		wantStatus     int
		wantSetCookie  bool
	}{
		{
			name:       "valid token in query",
			queryToken: token,
			wantStatus: http.StatusOK,
			wantSetCookie: true,
		},
		{
			name:       "invalid token in query",
			queryToken: "wrong-token",
			wantStatus: http.StatusUnauthorized,
			wantSetCookie: false,
		},
		{
			name:       "no token",
			wantStatus: http.StatusUnauthorized,
			wantSetCookie: false,
		},
		{
			name:        "valid token in cookie",
			cookieToken: token,
			wantStatus:  http.StatusOK,
			wantSetCookie: false,
		},
		{
			name:        "invalid token in cookie",
			cookieToken: "wrong-token",
			wantStatus:  http.StatusUnauthorized,
			wantSetCookie: false,
		},
		{
			name:        "valid query overrides invalid cookie", // Logic check: query is checked first?
			queryToken:  token,
			cookieToken: "wrong-token",
			wantStatus:  http.StatusOK, // If query is valid, it sets cookie and proceeds
			wantSetCookie: true,
		},
		{
			name:        "invalid query overrides valid cookie", // If query is present but invalid?
			queryToken:  "wrong-token",
			cookieToken: token,
			// Current logic:
			// if token != "" { if token == AuthToken { setCookie } } else { token = cookie }
			// if token != AuthToken { 401 }
			// So if query is "wrong-token", it checks match (fail), then checks match again (fail).
			// It does NOT fall back to cookie if query is present.
			wantStatus:  http.StatusUnauthorized,
			wantSetCookie: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			q := req.URL.Query()
			if tt.queryToken != "" {
				q.Add("token", tt.queryToken)
			}
			req.URL.RawQuery = q.Encode()

			if tt.cookieToken != "" {
				req.AddCookie(&http.Cookie{Name: AUTH_COOKIE_KEY, Value: tt.cookieToken})
			}

			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status code = %v, want %v", w.Code, tt.wantStatus)
			}

			cookies := w.Result().Cookies()
			foundCookie := false
			for _, c := range cookies {
				if c.Name == AUTH_COOKIE_KEY {
					foundCookie = true
					if c.Value != token {
						t.Errorf("cookie value = %v, want %v", c.Value, token)
					}
				}
			}

			if tt.wantSetCookie && !foundCookie {
				t.Errorf("expected cookie to be set")
			}
			if !tt.wantSetCookie && foundCookie {
				// Note: httptest recorder accumulates headers.
				// In this simple test, we create a new recorder each time, so this is fine.
				t.Errorf("did not expect cookie to be set")
			}
		})
	}
}
