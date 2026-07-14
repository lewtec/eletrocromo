package eletrocromo

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Empty AuthToken must not authenticate anyone. ConstantTimeCompare("", "") is
// 1, so ServeHTTP used without Run (or with a blank token) would otherwise
// treat missing credentials as valid.
func TestServeHTTP_EmptyAuthToken_FailClosed(t *testing.T) {
	app := &App{
		AuthToken: "",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}),
	}

	cases := []struct {
		name        string
		tokenParam  string
		cookieValue string
	}{
		{name: "no credentials"},
		{name: "empty query token", tokenParam: ""},
		{name: "empty cookie", cookieValue: ""},
		{name: "non-empty query still rejected", tokenParam: "anything"},
		{name: "non-empty cookie still rejected", cookieValue: "anything"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			path := "/"
			if tt.tokenParam != "" {
				path += "?token=" + tt.tokenParam
			}
			req := httptest.NewRequest(http.MethodGet, path, nil)
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{Name: AUTH_COOKIE_KEY, Value: tt.cookieValue})
			}
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
			}
		})
	}
}
