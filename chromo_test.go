package eletrocromo

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthRequired_Unconfigured(t *testing.T) {
	app := &App{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		AuthToken: "", // explicitly empty
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
	}
}
