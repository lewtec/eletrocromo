// Package desktop is the unpackaged share fallback (clipboard / open).
package desktop

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/lewtec/eletrocromo/driver"
	"github.com/lewtec/eletrocromo/driver/share"
)

func init() {
	driver.Register[share.Driver](factory{})
}

type factory struct{}

func (factory) ID() string                                { return "desktop" }
func (factory) Priority() int                             { return 0 }
func (factory) CheckCompatibility(context.Context) error  { return nil }
func (factory) New(context.Context) (share.Driver, error) { return impl{}, nil }

type impl struct{}

func (impl) Out(ctx context.Context, item share.Item) error {
	text := strings.TrimSpace(item.Text)
	if text == "" {
		text = strings.TrimSpace(item.URL)
	}
	if len(item.Paths) > 0 {
		return openPath(ctx, item.Paths[0])
	}
	if text == "" {
		return share.ErrEmptyItem
	}
	return clip(ctx, text)
}

func openPath(ctx context.Context, path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", path)
	case "windows":
		cmd = exec.CommandContext(ctx, "cmd", "/c", "start", "", path)
	default:
		cmd = exec.CommandContext(ctx, "xdg-open", path)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("open %s: %w: %s", path, err, out)
	}
	return nil
}

func clip(ctx context.Context, text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "pbcopy")
	case "windows":
		cmd = exec.CommandContext(ctx, "clip")
	default:
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.CommandContext(ctx, "wl-copy")
		} else {
			cmd = exec.CommandContext(ctx, "xclip", "-selection", "clipboard")
		}
	}
	cmd.Stdin = strings.NewReader(text)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("clipboard: %w: %s", err, out)
	}
	return nil
}
