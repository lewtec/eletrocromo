package eletrocromo

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAuthRequest(method, path, tokenParam, cookieValue string) *http.Request {
	if tokenParam != "" {
		path += "?token=" + tokenParam
	}
	req := httptest.NewRequest(method, path, nil)
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: AUTH_COOKIE_KEY, Value: cookieValue})
	}
	return req
}

func TestServeHTTP_Auth(t *testing.T) {
	authToken := "secret-token"
	app := &App{
		AuthToken: authToken,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("ok"))
			require.NoError(t, err)
		}),
	}

	tests := []struct {
		name           string
		tokenParam     string
		cookieValue    string
		expectedStatus int
	}{
		{name: "Valid token in query", tokenParam: authToken, expectedStatus: http.StatusOK},
		{name: "Invalid token in query", tokenParam: "wrong-token", expectedStatus: http.StatusUnauthorized},
		{name: "No token", expectedStatus: http.StatusUnauthorized},
		{name: "Valid token in cookie", cookieValue: authToken, expectedStatus: http.StatusOK},
		{name: "Invalid token in cookie", cookieValue: "wrong-token", expectedStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newAuthRequest(http.MethodGet, "/", tt.tokenParam, tt.cookieValue)
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.tokenParam != authToken {
				return
			}
			var cookie *http.Cookie
			for _, c := range w.Result().Cookies() {
				if c.Name == AUTH_COOKIE_KEY && c.Value == authToken {
					cookie = c
				}
			}
			require.NotNil(t, cookie)
			assert.True(t, cookie.HttpOnly)
			assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
		})
	}
}
