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

const testAppID = "br.tec.lew.test"

// ErrFakeExit stands in for a process exit status in wrapHeliumExit tests.
var ErrFakeExit = errors.New("exit status 1")

func TestRedactSecretsInText_StripsTokenQuery(t *testing.T) {
	secret := "deadbeef-cafe-babe-0000-111122223333"
	in := fmt.Sprintf(
		`[1234:5678:ERROR:launch] failed --app=http://127.0.0.1:3847/?token=%s other=1`,
		secret,
	)
	out := redactSecretsInText(in)
	if strings.Contains(out, secret) {
		t.Fatalf("token still present after redaction: %q", out)
	}
	if strings.Contains(out, "token=") {
		t.Fatalf("token= still present after redaction: %q", out)
	}
	if !strings.Contains(out, "http://127.0.0.1:3847/") {
		t.Fatalf("expected base URL kept, got %q", out)
	}
}

func TestWrapHeliumExit_RedactsTokenFromStderr(t *testing.T) {
	secret := "session-secret-do-not-leak"
	stderr := "Chromium: --app=http://127.0.0.1:9/?token=" + secret + " crashed"
	err := wrapHeliumExit(stderr, ErrFakeExit)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if strings.Contains(msg, secret) {
		t.Fatalf("token leaked into error: %q", msg)
	}
	if strings.Contains(msg, "token=") {
		t.Fatalf("token= leaked into error: %q", msg)
	}
	if !errors.Is(err, ErrHeliumLaunch) {
		t.Fatalf("want ErrHeliumLaunch, got %v", err)
	}
}

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
			err = LaunchChromium(t.Context(), u, testAppID)
			if err == nil {
				t.Fatal("expected error for non-http(s) scheme")
			}
			if !errors.Is(err, ErrInvalidURLScheme) {
				t.Fatalf("expected ErrInvalidURLScheme, got %v", err)
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
	_, err := ResolveBrowserHost(t.Context())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrNoChromium) {
		t.Fatalf("want ErrNoChromium, got %v", err)
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "xdg-open") || strings.Contains(msg, "system browser") {
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

	path, err := ResolveBrowserHost(t.Context())
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
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	u, err := url.Parse("http://127.0.0.1:9/")
	if err != nil {
		t.Fatal(err)
	}
	err = LaunchChromium(t.Context(), u, testAppID)
	if err == nil {
		t.Fatal("expected error without host")
	}
	if !errors.Is(err, ErrNoChromium) {
		t.Fatalf("want ErrNoChromium, got %v", err)
	}
}

func TestLaunchChromium_UsesAppIDProfile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)

	dir, err := ProfileDir(testAppID)
	if err != nil {
		t.Fatal(err)
	}
	wantSuffix := filepath.Join("eletrocromo", "profiles", testAppID)
	if !strings.HasSuffix(dir, wantSuffix) {
		t.Fatalf("profile %q does not end with %q", dir, wantSuffix)
	}
	if !strings.HasPrefix(dir, root) {
		t.Fatalf("profile not under XDG_DATA_HOME: %q", dir)
	}
}

func TestRun_RequiresAppID(t *testing.T) {
	app := App{
		Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		Context: t.Context(),
	}
	if err := app.Run(); err == nil {
		t.Fatal("expected error for missing App.ID")
	}
}

func TestRun_ResolveFailsBeforeServer(t *testing.T) {
	orig := resolveBrowserHost
	t.Cleanup(func() { resolveBrowserHost = orig })
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	resolveBrowserHost = func(context.Context) (string, error) {
		return "", fmt.Errorf("%w: test deny", ErrNoChromium)
	}

	app := App{
		ID:      "br.tec.lew.test.resolve",
		Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		Context: t.Context(),
	}
	err := app.Run()
	if err == nil {
		t.Fatal("expected resolve error from Run")
	}
	if !errors.Is(err, ErrNoChromium) {
		t.Fatalf("want ErrNoChromium, got %v", err)
	}
}

func TestRun_ImmediateHeliumExitIsError(t *testing.T) {
	origResolve := resolveBrowserHost
	origGrace := heliumStartupGrace
	t.Cleanup(func() {
		resolveBrowserHost = origResolve
		heliumStartupGrace = origGrace
	})
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	heliumStartupGrace = 200 * time.Millisecond

	trueBin, err := exec.LookPath("true")
	if err != nil {
		t.Skip("no true binary")
	}
	resolveBrowserHost = func(context.Context) (string, error) {
		return trueBin, nil
	}

	app := App{
		ID:      "br.tec.lew.test.exit",
		Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		Context: t.Context(),
	}
	err = app.Run()
	if err == nil {
		t.Fatal("expected ErrHeliumLaunch when host exits immediately")
	}
	if !errors.Is(err, ErrHeliumLaunch) {
		t.Fatalf("want ErrHeliumLaunch, got %v", err)
	}
}

func TestRun_ResolvesThenLaunches(t *testing.T) {
	origResolve := resolveBrowserHost
	origGrace := heliumStartupGrace
	t.Cleanup(func() {
		resolveBrowserHost = origResolve
		heliumStartupGrace = origGrace
	})
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	heliumStartupGrace = 100 * time.Millisecond

	var resolved atomic.Bool
	script := filepath.Join(t.TempDir(), "fake-helium")
	// Stay up longer than grace + parent timeout window.
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexec sleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	resolveBrowserHost = func(context.Context) (string, error) {
		resolved.Store(true)
		return script, nil
	}

	// Bound Run by context instead of a manual Sleep goroutine.
	ctx, cancel := context.WithTimeout(t.Context(), 400*time.Millisecond)
	defer cancel()

	app := App{
		ID:      "br.tec.lew.test.launch",
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

	path, err := bootstrapWorkspaced(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if path != bin {
		t.Fatalf("got %q want %q", path, bin)
	}
}
