package eletrocromo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestLaunchChromium_RejectsNonHTTPSchemes(t *testing.T) {
	cases := []string{
		"file:///etc/passwd",
		"javascript:alert(1)",
		"ftp://example.com/",
		"data:text/html,hi",
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			u, err := url.Parse(raw)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			err = LaunchChromium(context.Background(), u)
			if err == nil {
				t.Fatal("expected error for non-http(s) scheme")
			}
			if !strings.Contains(err.Error(), "invalid URL scheme") {
				t.Fatalf("expected scheme error, got %v", err)
			}
		})
	}
}

func TestGetChromium_NoPanic(t *testing.T) {
	path, err := GetChromium()
	if err == nil {
		if path == "" {
			t.Fatal("empty path with nil error")
		}
		return
	}
	if !errors.Is(err, ErrNoChromium) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetChromium_OnlyHelium(t *testing.T) {
	orig := lookPath
	t.Cleanup(func() { lookPath = orig })

	lookPath = func(file string) (string, error) {
		switch file {
		case "helium":
			return "/fake/helium", nil
		case "chromium", "chrome", "google-chrome", "msedge", "brave":
			return "/fake/" + file, nil
		default:
			return "", exec.ErrNotFound
		}
	}
	path, err := GetChromium()
	if err != nil {
		t.Fatal(err)
	}
	if path != "/fake/helium" {
		t.Fatalf("want helium, got %q", path)
	}
}

func TestGetChromium_IgnoresOtherBrowsers(t *testing.T) {
	orig := lookPath
	t.Cleanup(func() { lookPath = orig })

	lookPath = func(file string) (string, error) {
		if file == "chromium" || file == "chrome" || file == "google-chrome" {
			return "/usr/bin/" + file, nil
		}
		return "", exec.ErrNotFound
	}
	_, err := GetChromium()
	if !errors.Is(err, ErrNoChromium) {
		t.Fatalf("want ErrNoChromium when only Chrome/Chromium present, got path/err %v", err)
	}
}

func TestResolveBrowserHost_NoEnsureNoHost(t *testing.T) {
	orig := lookPath
	t.Cleanup(func() { lookPath = orig })
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }

	t.Setenv("ELETROCROMO_NO_ENSURE", "1")
	_, err := ResolveBrowserHost(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrNoChromium) {
		t.Fatalf("want ErrNoChromium, got %v", err)
	}
	if strings.Contains(err.Error(), "xdg-open") || strings.Contains(strings.ToLower(err.Error()), "system browser") {
		t.Fatalf("must not mention system browser fallback: %v", err)
	}
}

func TestResolveBrowserHost_EnsureViaWorkspacedWhich(t *testing.T) {
	origLook := lookPath
	origCmd := commandOutput
	t.Cleanup(func() {
		lookPath = origLook
		commandOutput = origCmd
	})

	lookPath = func(file string) (string, error) {
		if file == "workspaced" {
			return "/bin/workspaced", nil
		}
		return "", exec.ErrNotFound
	}
	t.Setenv("ELETROCROMO_NO_ENSURE", "")
	t.Setenv("ELETROCROMO_WORKSPACED", "")

	var gotArgs []string
	commandOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		gotArgs = append([]string{name}, args...)
		return []byte("/cache/tools/helium\n"), nil
	}

	path, err := ResolveBrowserHost(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if path != "/cache/tools/helium" {
		t.Fatalf("got path %q", path)
	}
	want := []string{"/bin/workspaced", "tool", "which", "helium-browser", "helium"}
	if len(gotArgs) != len(want) {
		t.Fatalf("args %v want %v", gotArgs, want)
	}
	for i := range want {
		if gotArgs[i] != want[i] {
			t.Fatalf("args %v want %v", gotArgs, want)
		}
	}
}

func TestLaunchChromium_NoSystemBrowserFallback(t *testing.T) {
	orig := lookPath
	t.Cleanup(func() { lookPath = orig })
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	t.Setenv("ELETROCROMO_NO_ENSURE", "1")

	u, err := url.Parse("http://127.0.0.1:9/")
	if err != nil {
		t.Fatal(err)
	}
	err = LaunchChromium(context.Background(), u)
	if err == nil {
		t.Fatal("expected error without host")
	}
	if !errors.Is(err, ErrNoChromium) {
		t.Fatalf("want ErrNoChromium, got %v", err)
	}
}

func TestRun_ResolveFailsBeforeServer(t *testing.T) {
	orig := resolveBrowserHost
	t.Cleanup(func() { resolveBrowserHost = orig })

	resolveBrowserHost = func(context.Context) (string, error) {
		return "", fmt.Errorf("%w: test deny", ErrNoChromium)
	}

	app := App{
		Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		Context: context.Background(),
	}
	err := app.Run()
	if err == nil {
		t.Fatal("expected resolve error from Run")
	}
	if !errors.Is(err, ErrNoChromium) {
		t.Fatalf("want ErrNoChromium, got %v", err)
	}
}

func TestRun_ResolvesThenLaunches(t *testing.T) {
	orig := resolveBrowserHost
	t.Cleanup(func() { resolveBrowserHost = orig })

	var resolved atomic.Bool
	trueBin, err := exec.LookPath("true")
	if err != nil {
		t.Skip("no true binary")
	}
	resolveBrowserHost = func(context.Context) (string, error) {
		resolved.Store(true)
		return trueBin, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Cancel shortly after Run enters the wait loop (post-launch).
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	app := App{
		Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		Context: ctx,
	}
	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
	if !resolved.Load() {
		t.Fatal("resolveBrowserHost was not called")
	}
}

func TestWorkspacedAssetName_HasPinnedChecksum(t *testing.T) {
	name, err := workspacedAssetName()
	if err != nil {
		t.Skip(err)
	}
	if _, ok := workspacedAssetSHA256[name]; !ok {
		t.Fatalf("asset %q missing from pinned checksums", name)
	}
	if !strings.HasPrefix(name, "workspaced_") {
		t.Fatalf("unexpected name %q", name)
	}
}

func TestBootstrapSkipsDownloadWhenCached(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)
	cache, err := workspacedCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(cache, "workspaced")
	if err := os.WriteFile(bin, []byte("#!/bin/true\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	origGet := httpGet
	t.Cleanup(func() { httpGet = origGet })
	httpGet = func(context.Context, string) (*http.Response, error) {
		t.Fatal("httpGet should not be called when binary is cached")
		return nil, nil
	}

	path, err := bootstrapWorkspaced(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if path != bin {
		t.Fatalf("got %q want %q", path, bin)
	}
}
