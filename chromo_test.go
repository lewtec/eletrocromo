package eletrocromo

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApp_ServeHTTP(t *testing.T) {
	authToken := "secret-token"
	app := &App{
		AuthToken: authToken,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "success")
		}),
	}

	tests := []struct {
		name           string
		token          string
		cookie         *http.Cookie
		expectedStatus int
	}{
		{
			name:           "No token, no cookie",
			token:          "",
			cookie:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid token",
			token:          "wrong-token",
			cookie:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Valid token in query",
			token:          authToken,
			cookie:         nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid token in cookie",
			token:          "",
			cookie:         &http.Cookie{Name: AUTH_COOKIE_KEY, Value: authToken},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid token in cookie",
			token:          "",
			cookie:         &http.Cookie{Name: AUTH_COOKIE_KEY, Value: "wrong"},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/"
			if tt.token != "" {
				url += "?token=" + tt.token
			}
			req := httptest.NewRequest("GET", url, nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			w := httptest.NewRecorder()

			app.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
