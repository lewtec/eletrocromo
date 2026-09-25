package eletrocromo

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServeHTTP_NilHandler_NotFound(t *testing.T) {
	app := &App{AuthToken: "secret-token"}
	req := httptest.NewRequest(http.MethodGet, "/?token=secret-token", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "no handler setup", w.Body.String())
}
