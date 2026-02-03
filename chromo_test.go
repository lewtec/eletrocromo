package eletrocromo

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeHTTP(t *testing.T) {
	app := &App{
		AuthToken: "secret",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprint(w, "success")
		}),
	}

	tests := []struct {
		name       string
		token      string
		cookie     string
		wantCode   int
		wantBody   string
	}{
		{
			name:     "Valid token",
			token:    "secret",
			wantCode: http.StatusOK,
			wantBody: "success",
		},
		{
			name:     "Invalid token",
			token:    "wrong",
			wantCode: http.StatusUnauthorized,
			wantBody: "forbidden",
		},
		{
			name:     "No token",
			token:    "",
			wantCode: http.StatusUnauthorized,
			wantBody: "forbidden",
		},
		{
			name:     "Valid cookie",
			cookie:   "secret",
			wantCode: http.StatusOK,
			wantBody: "success",
		},
		{
			name:     "Invalid cookie",
			cookie:   "wrong",
			wantCode: http.StatusUnauthorized,
			wantBody: "forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.token != "" {
				q := req.URL.Query()
				q.Add("token", tt.token)
				req.URL.RawQuery = q.Encode()
			}
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: AUTH_COOKIE_KEY, Value: tt.cookie})
			}

			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("expected status %d, got %d", tt.wantCode, w.Code)
			}
			gotBody := strings.TrimSpace(w.Body.String())
			if gotBody != tt.wantBody {
				t.Errorf("expected body %q, got %q (raw: %q)", tt.wantBody, gotBody, w.Body.String())
			}
		})
	}
}

func TestServeHTTP_NoHandler(t *testing.T) {
	app := &App{
		AuthToken: "secret",
		Handler:   nil,
	}

	req := httptest.NewRequest("GET", "/?token=secret", nil)
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
	gotBody := strings.TrimSpace(w.Body.String())
	if gotBody != "no handler setup" {
		t.Errorf("expected body 'no handler setup', got %q", gotBody)
	}
}
