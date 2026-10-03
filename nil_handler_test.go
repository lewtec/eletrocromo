package eletrocromo

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeHTTP_NilHandler_NotFound(t *testing.T) {
	app := &App{AuthToken: "secret-token"}
	req := httptest.NewRequest(http.MethodGet, "/?token=secret-token", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
	if body := w.Body.String(); body != "no handler setup" {
		t.Fatalf("unexpected body %q", body)
	}
}
