package eletrocromo

import (
	"errors"
	"fmt"
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

func LaunchChromium(url string) error {
	chromium, err := GetChromium()
	if errors.Is(err, ErrNoChromium) {
		return gopen.Open(url)
	}
	cmd := exec.Command(chromium, fmt.Sprintf("--app=%s", url))
	return cmd.Start()
}
