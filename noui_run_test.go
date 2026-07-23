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
)

func TestRun_NoUI_PrintsReadyAndServes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(io.MultiWriter(prev, &buf))
	defer log.SetOutput(prev)

	app := &App{
		ID:    "br.tec.lew.eletrocromo.noui_test",
		NoUI:  true,
		Context: ctx,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("pong"))
		}),
	}

	errCh := make(chan error, 1)
	go func() { errCh <- app.Run() }()

	// Wait for READY line.
	var link string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if i := strings.Index(buf.String(), ReadyLinePrefix); i >= 0 {
			rest := buf.String()[i+len(ReadyLinePrefix):]
			if j := strings.IndexByte(rest, '\n'); j >= 0 {
				link = strings.TrimSpace(rest[:j])
			} else {
				link = strings.TrimSpace(rest)
			}
			if link != "" {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if link == "" {
		cancel()
		t.Fatalf("no READY line in logs:\n%s", buf.String())
	}
	if !strings.HasPrefix(link, "http://127.0.0.1:") && !strings.HasPrefix(link, "http://localhost:") {
		t.Fatalf("unexpected READY url %q", link)
	}
	if !strings.Contains(link, "token=") {
		t.Fatalf("READY url missing token: %q", link)
	}

	resp, err := http.Get(link)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || string(body) != "pong" {
		cancel()
		t.Fatalf("status=%d body=%q", resp.StatusCode, body)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not exit after cancel")
	}
}
