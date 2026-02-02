package eletrocromo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAppServeHTTP(t *testing.T) {
	authToken := "secret-token"
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "success")
	})

	app := &App{
		Handler:   mockHandler,
		AuthToken: authToken,
		Context:   context.Background(),
	}

	tests := []struct {
		name           string
		url            string
		cookie         *http.Cookie
		expectedStatus int
		expectCookie   bool
	}{
		{
			name:           "Valid Token",
			url:            "/?token=" + authToken,
			expectedStatus: http.StatusOK,
			expectCookie:   true,
		},
		{
			name:           "Valid Cookie",
			url:            "/",
			cookie:         &http.Cookie{Name: AUTH_COOKIE_KEY, Value: authToken},
			expectedStatus: http.StatusOK,
			expectCookie:   false, // Cookie already set, but logic might set it again? Current logic: if token in query matches, set cookie. If not in query, check cookie.
		},
		{
			name:           "No Token No Cookie",
			url:            "/",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Token",
			url:            "/?token=wrong",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Cookie",
			url:            "/",
			cookie:         &http.Cookie{Name: AUTH_COOKIE_KEY, Value: "wrong"},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			w := httptest.NewRecorder()

			app.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectCookie {
				found := false
				for _, c := range resp.Cookies() {
					if c.Name == AUTH_COOKIE_KEY && c.Value == authToken {
						found = true
						break
					}
				}
				if !found {
					t.Error("expected auth cookie to be set")
				}
			}
		})
	}
}
