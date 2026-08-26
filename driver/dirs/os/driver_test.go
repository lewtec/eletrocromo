package os

import (
	"path/filepath"
	"testing"
)

func TestResolve_EnvWins(t *testing.T) {
	got, err := paths(
		"br.tec.lew.counter",
		"linux",
		func(k string) string {
			switch k {
			case envData:
				return "/data"
			case envCache:
				return "/cache"
			case envConfig:
				return "/config"
			default:
				return ""
			}
		},
		func() (string, error) { return "/home/u", nil },
		func() (string, error) { return "/unused-cache", nil },
		func() (string, error) { return "/unused-config", nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Data != "/data" || got.Cache != "/cache" || got.Config != "/config" {
		t.Fatalf("got %+v", got)
	}
	if got.Inbox != filepath.Join("/cache", "inbox") {
		t.Fatalf("Inbox = %q", got.Inbox)
	}
}

func TestDataHome_ByGOOS(t *testing.T) {
	env := map[string]string{}
	getenv := func(k string) string { return env[k] }
	home := func() (string, error) { return "/home/u", nil }

	got, err := dataHome("linux", getenv, home)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/home/u", ".local", "share")
	if got != want {
		t.Fatalf("linux: got %q want %q", got, want)
	}

	got, err = dataHome("darwin", getenv, home)
	if err != nil {
		t.Fatal(err)
	}
	want = filepath.Join("/home/u", "Library", "Application Support")
	if got != want {
		t.Fatalf("darwin: got %q want %q", got, want)
	}

	env["LOCALAPPDATA"] = filepath.Join("C:", "Users", "u", "AppData", "Local")
	got, err = dataHome("windows", getenv, home)
	if err != nil {
		t.Fatal(err)
	}
	if got != env["LOCALAPPDATA"] {
		t.Fatalf("windows: got %q want %q", got, env["LOCALAPPDATA"])
	}
}
