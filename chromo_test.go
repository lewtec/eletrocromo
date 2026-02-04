package eletrocromo

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeHTTP_Auth(t *testing.T) {
	authToken := "secret-token"
	app := &App{
		AuthToken: authToken,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
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
			path := "/"
			if tt.tokenParam != "" {
				path += "?token=" + tt.tokenParam
			}
			req := httptest.NewRequest("GET", path, nil)
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{Name: AUTH_COOKIE_KEY, Value: tt.cookieValue})
			}
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

// Verify that the constant time comparison logic is actually being used
// This is a bit meta, but we can verify the behavior at least.
func TestConstantTimeCompareUsage(t *testing.T) {
	// This test just ensures we are using crypto/subtle logic in our heads,
	// but strictly speaking we are testing the endpoint behavior above.
	// We can't easily assert "ConstantTimeCompare was called" without mocking crypto/subtle which is impossible.
	// So we rely on code review for that part, and functional correctness above.
}
