package eletrocromo

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_NoUI_PrintsReadyAndServes(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(io.MultiWriter(prev, &buf))
	defer log.SetOutput(prev)

	app := &App{
		ID:      "br.tec.lew.eletrocromo.noui_test",
		NoUI:    true,
		Context: ctx,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte("pong")); err != nil {
				return
			}
		}),
	}

	errCh := make(chan error, 1)
	go func() { errCh <- app.Run() }()

	var link string
	require.Eventually(t, func() bool {
		i := strings.Index(buf.String(), ReadyLinePrefix)
		if i < 0 {
			return false
		}
		rest := buf.String()[i+len(ReadyLinePrefix):]
		if j := strings.IndexByte(rest, '\n'); j >= 0 {
			link = strings.TrimSpace(rest[:j])
		} else {
			link = strings.TrimSpace(rest)
		}
		return link != ""
	}, 3*time.Second, 20*time.Millisecond, "no READY line in logs:\n%s", buf.String())
	assert.True(t, strings.HasPrefix(link, "http://127.0.0.1:") || strings.HasPrefix(link, "http://localhost:"), link)
	assert.Contains(t, link, "token=")

	resp, err := http.Get(link)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	if cerr := resp.Body.Close(); err == nil {
		err = cerr
	}
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "pong", string(body))

	cancel()
	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		require.Fail(t, "Run did not exit after cancel")
	}
}
