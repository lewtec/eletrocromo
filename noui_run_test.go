package eletrocromo

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRun_NoUI_PrintsReadyAndServes(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var logBuf bytes.Buffer
	prevLog := log.Writer()
	log.SetOutput(io.MultiWriter(prevLog, &logBuf))
	defer log.SetOutput(prevLog)

	// Capture stdout: READY line (with token) is machine-parseable for Android.
	var stdoutBuf bytes.Buffer
	var stdoutMu sync.Mutex
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	prevStdout := os.Stdout
	os.Stdout = wOut
	defer func() {
		os.Stdout = prevStdout
		_ = wOut.Close()
		_ = rOut.Close()
	}()
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		buf := make([]byte, 4096)
		for {
			n, err := rOut.Read(buf)
			if n > 0 {
				stdoutMu.Lock()
				stdoutBuf.Write(buf[:n])
				stdoutMu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()

	app := &App{
		ID:      "br.tec.lew.eletrocromo.noui_test",
		NoUI:    true,
		Context: ctx,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("pong"))
		}),
	}

	errCh := make(chan error, 1)
	go func() { errCh <- app.Run() }()

	// Wait for READY line on stdout.
	var link string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		stdoutMu.Lock()
		s := stdoutBuf.String()
		stdoutMu.Unlock()
		if i := strings.Index(s, ReadyLinePrefix); i >= 0 {
			rest := s[i+len(ReadyLinePrefix):]
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
		stdoutMu.Lock()
		stdout := stdoutBuf.String()
		stdoutMu.Unlock()
		t.Fatalf("no READY line on stdout:\nstdout=%q\nlogs=%s", stdout, logBuf.String())
	}
	if !strings.HasPrefix(link, "http://127.0.0.1:") && !strings.HasPrefix(link, "http://localhost:") {
		t.Fatalf("unexpected READY url %q", link)
	}
	if !strings.Contains(link, "token=") {
		t.Fatalf("READY url missing token: %q", link)
	}

	// Logs must not contain the session secret (stdout READY / READY file still do).
	token := link[strings.Index(link, "token=")+len("token="):]
	if amp := strings.IndexByte(token, '&'); amp >= 0 {
		token = token[:amp]
	}
	if token == "" {
		t.Fatal("empty token in READY url")
	}
	logs := logBuf.String()
	if strings.Contains(logs, token) {
		t.Fatalf("auth token leaked into logs:\n%s", logs)
	}
	if strings.Contains(logs, "token=") {
		t.Fatalf("logs must not include token= query:\n%s", logs)
	}

	resp, err := http.Get(link)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		cancel()
		t.Fatal(readErr)
	}
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

	// Restore stdout and close writer so the reader exits.
	os.Stdout = prevStdout
	_ = wOut.Close()
	<-readDone
}
