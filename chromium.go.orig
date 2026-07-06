package eletrocromo

import (
	"errors"
	"fmt"
	"net/url"
	"os/exec"

	"github.com/jasonlovesdoggo/gopen"
)

var chromiumLikes = []string{
	"C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe", // we hate it but we can count it's there
	"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
	"/Applications/Chromium.app/Contents/MacOS/Chromium",
	"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
	"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
	"/Applications/Vivaldi.app/Contents/MacOS/Vivaldi",
	"/Applications/Opera.app/Contents/MacOS/Opera",
	"msedge",
	"brave",
	"vivaldi",
	"opera",
	"chromium",
	"chrome",
	"google-chrome",
	"google-chrome-stable",
	"chromium-browser",
}

var ErrNoChromium = errors.New("no chromium detected")

// GetChromium searches for a Chromium-based browser installation on the system.
// It iterates through a predefined list of common browser paths and executable names,
// returning the path of the first one found.
func GetChromium() (string, error) {
	for _, ch := range chromiumLikes {
		path, err := exec.LookPath(ch)
		if errors.Is(err, exec.ErrNotFound) {
			continue
		}
		if err != nil {
			continue
		}
		return path, nil
	}
	return "", ErrNoChromium
}

// LaunchChromium opens the specified URL in a Chromium-based browser in "app mode" (borderless).
// If no suitable browser is found, it falls back to the system's default browser.
//
// This function enforces security by requiring the URL scheme to be either "http" or "https".
func LaunchChromium(u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: %s", u.Scheme)
	}
	chromium, err := GetChromium()
	if errors.Is(err, ErrNoChromium) {
		return gopen.Open(u.String())
	}
	cmd := exec.Command(chromium, "--app", u.String())
	return cmd.Start()
}
