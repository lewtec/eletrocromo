package eletrocromo

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
			if _, err := w.Write([]byte("ok")); err != nil {
				t.Errorf("write body: %v", err)
			}
		}),
	}

	tests := []struct {
		name           string
		tokenParam     string
		cookieValue    string
		expectedStatus int
	}{
		{
			name:           "Valid token in query",
			tokenParam:     authToken,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid token in query",
			tokenParam:     "wrong-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "No token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Valid token in cookie",
			cookieValue:    authToken,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid token in cookie",
			cookieValue:    "wrong-token",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newAuthRequest(http.MethodGet, "/", tt.tokenParam, tt.cookieValue)
			w := httptest.NewRecorder()

			app.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Verify cookie is set when valid token provided in query
			if tt.tokenParam == authToken {
				cookies := w.Result().Cookies()
				found := false
				for _, c := range cookies {
					if c.Name == AUTH_COOKIE_KEY && c.Value == authToken {
						found = true
						if !c.HttpOnly {
							t.Error("cookie should be HttpOnly")
						}
						if c.SameSite != http.SameSiteStrictMode {
							t.Error("cookie should be SameSiteStrictMode")
						}
					}
				}
				if !found {
					t.Error("auth cookie not set on valid login")
				}
			}
		})
	}
}
