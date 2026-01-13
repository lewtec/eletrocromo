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
	"chromium",
	"chromium-browser",
}

var ErrNoChromium = errors.New("no chromium detected")

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

func LaunchChromium(appUrl string) error {
	parsedUrl, err := url.Parse(appUrl)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if parsedUrl.Scheme != "http" && parsedUrl.Scheme != "https" {
		return fmt.Errorf("invalid url scheme: %s", parsedUrl.Scheme)
	}
	if parsedUrl.Host == "" {
		return fmt.Errorf("invalid url host: %s", parsedUrl.Host)
	}
	chromium, err := GetChromium()
	if errors.Is(err, ErrNoChromium) {
		return gopen.Open(appUrl)
	}
	cmd := exec.Command(chromium, "--app", appUrl)
	return cmd.Start()
}
