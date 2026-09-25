package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lewtec/eletrocromo/internal/gen/apk"
	"github.com/lewtec/lewkit/x/cmd"
)

// errConfigRequired is returned when build or run has no eletrocromo.json path.
var errConfigRequired = errors.New("config path required")

// buildCmd cross-compiles the app in eletrocromo.json for GOOS/GOARCH.
// GOOS and GOARCH default to this machine. GOARCH can also be set with --arch.
type buildCmd struct {
	path cmd.StringArg `help:"path to eletrocromo.json"`
	goos cmd.StringArg `long:"goos" default:"" help:"target GOOS (default: GOOS, else this machine)"`
	arch cmd.StringArg `long:"arch" default:"" help:"target GOARCH (default: GOARCH, else this machine)"`
	sdk  cmd.StringArg `long:"sdk" default:"" help:"iphonesimulator or iphoneos when GOOS=ios"`
	hostFlags
}

func (buildCmd) Description() string {
	return "Build the app in eletrocromo.json for GOOS/GOARCH (darwin, ios, android, linux, windows)."
}

func (c *buildCmd) Run(ctx context.Context) error {
	return c.dispatch(ctx, false)
}

type runCmd struct {
	buildCmd
}

func (runCmd) Description() string {
	return "Build the app in eletrocromo.json for GOOS/GOARCH and launch it."
}

func (c *runCmd) Run(ctx context.Context) error {
	return c.dispatch(ctx, true)
}

func (c *buildCmd) dispatch(ctx context.Context, launch bool) error {
	config := strings.TrimSpace(c.path.Value())
	if config == "" {
		config = strings.TrimSpace(c.config.Value())
	}
	if config == "" {
		return errConfigRequired
	}
	if err := c.config.Parse(config); err != nil {
		return err
	}
	goos := strings.TrimSpace(c.goos.Value())
	if goos == "" {
		goos = strings.TrimSpace(os.Getenv("GOOS"))
	}
	if goos == "" {
		goos = runtime.GOOS
	}
	arch := strings.TrimSpace(c.arch.Value())
	if arch == "" {
		arch = strings.TrimSpace(os.Getenv("GOARCH"))
	}
	if arch == "" {
		arch = runtime.GOARCH
	}
	absConfig, err := filepath.Abs(config)
	if err != nil {
		return err
	}
	slug := filepath.Base(filepath.Dir(absConfig))
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if strings.TrimSpace(c.out.Value()) == "" {
		out, err := defaultArtifact(cwd, goos, slug, arch)
		if err != nil {
			return err
		}
		if err := c.out.Parse(out); err != nil {
			return err
		}
	}
	if strings.TrimSpace(c.workDir.Value()) == "" {
		if err := c.workDir.Parse(filepath.Join(cwd, "dist", goos+"-"+slug)); err != nil {
			return err
		}
	}
	if launch && goos == "android" {
		if err := ensureAndroidSDK(ctx); err != nil {
			return err
		}
	}
	if err := c.buildTarget(ctx, goos, arch); err != nil {
		return err
	}
	if !launch {
		return nil
	}
	return launchTarget(ctx, goos, arch, c.out.Value(), absConfig)
}

func (c *buildCmd) buildTarget(ctx context.Context, goos, arch string) error {
	switch goos {
	case "android":
		return (&buildAndroidCmd{hostFlags: c.hostFlags}).Run(ctx)
	case "darwin":
		return (&macosCmd{hostFlags: c.hostFlags}).Run(ctx)
	case "ios":
		ios := &iosCmd{hostFlags: c.hostFlags}
		if sdk := strings.TrimSpace(c.sdk.Value()); sdk != "" {
			if err := ios.sdk.Parse(sdk); err != nil {
				return err
			}
		}
		return ios.Run(ctx)
	case "linux":
		cmd := &linuxCmd{desktopCmd: desktopCmd{hostFlags: c.hostFlags}}
		if err := cmd.arch.Parse(arch); err != nil {
			return err
		}
		return cmd.Run(ctx)
	case "windows":
		cmd := &windowsCmd{desktopCmd: desktopCmd{hostFlags: c.hostFlags}}
		if err := cmd.arch.Parse(arch); err != nil {
			return err
		}
		return cmd.Run(ctx)
	default:
		return fmt.Errorf("unsupported GOOS %q", goos)
	}
}

func defaultArtifact(cwd, goos, slug, arch string) (string, error) {
	dist := filepath.Join(cwd, "dist")
	switch goos {
	case "android":
		return filepath.Join(dist, slug+"-debug.apk"), nil
	case "darwin":
		return filepath.Join(dist, slug+".app"), nil
	case "ios":
		return filepath.Join(dist, slug+"-ios.app"), nil
	case "linux":
		return filepath.Join(dist, slug+"-linux-"+arch), nil
	case "windows":
		return filepath.Join(dist, slug+"-windows-"+arch+".exe"), nil
	default:
		return "", fmt.Errorf("unsupported GOOS %q", goos)
	}
}

func ensureAndroidSDK(ctx context.Context) error {
	if _, err := exec.LookPath("sdkmanager"); err != nil {
		return nil
	}
	lic := exec.CommandContext(ctx, "sh", "-c", "yes | sdkmanager --licenses >/dev/null || true")
	lic.Stdout, lic.Stderr = os.Stdout, os.Stderr
	if err := lic.Run(); err != nil {
		return err
	}
	inst := exec.CommandContext(ctx, "sdkmanager", "--install", "platforms;android-35", "build-tools;35.0.0")
	inst.Stdout, inst.Stderr = os.Stdout, os.Stderr
	return inst.Run()
}

func launchTarget(ctx context.Context, goos, arch, artifact, config string) error {
	switch goos {
	case "darwin":
		return runCmdOut(ctx, "open", artifact)
	case "ios":
		return launchIOS(ctx, artifact, config)
	case "android":
		return launchAndroid(ctx, artifact, config)
	case "linux", "windows":
		if runtime.GOOS != goos || runtime.GOARCH != arch {
			_, err := fmt.Fprintf(os.Stdout, "built %s\n", artifact)
			return err
		}
		cmd := exec.CommandContext(ctx, artifact)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		return cmd.Run()
	default:
		return fmt.Errorf("unsupported GOOS %q", goos)
	}
}

func launchIOS(ctx context.Context, app, config string) error {
	id, err := packageID(config)
	if err != nil {
		return err
	}
	if err := bootIOSSimulator(ctx); err != nil {
		return err
	}
	if err := runCmdOut(ctx, "xcrun", "simctl", "install", "booted", app); err != nil {
		return err
	}
	return runCmdOut(ctx, "xcrun", "simctl", "launch", "booted", id)
}

func bootIOSSimulator(ctx context.Context) error {
	out, err := exec.CommandContext(ctx, "xcrun", "simctl", "list", "devices", "booted").Output()
	if err != nil {
		return err
	}
	if strings.Contains(string(out), "iPhone") {
		return nil
	}
	list, err := exec.CommandContext(ctx, "xcrun", "simctl", "list", "devices", "available").Output()
	if err != nil {
		return err
	}
	var udid string
	for _, line := range strings.Split(string(list), "\n") {
		if !strings.Contains(line, "iPhone") {
			continue
		}
		start := strings.Index(line, "(")
		end := strings.Index(line, ")")
		if start >= 0 && end > start {
			udid = line[start+1 : end]
			break
		}
	}
	if udid == "" {
		return fmt.Errorf("no available iPhone simulator")
	}
	if err := runCmdOut(ctx, "xcrun", "simctl", "boot", udid); err != nil {
		return err
	}
	if err := runCmdOut(ctx, "open", "-a", "Simulator"); err != nil {
		return err
	}
	return runCmdOut(ctx, "xcrun", "simctl", "bootstatus", udid, "-b")
}

func launchAndroid(ctx context.Context, apk, config string) error {
	id, err := packageID(config)
	if err != nil {
		return err
	}
	if err := runCmdOut(ctx, "adb", "install", "-r", apk); err != nil {
		return err
	}
	return runCmdOut(ctx, "adb", "shell", "am", "start", "-n", id+"/.MainActivity")
}

func packageID(config string) (string, error) {
	cfg, _, err := apk.LoadConfig(config)
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(cfg.PackageID)
	if id == "" {
		return "", fmt.Errorf("%w: %s", ErrMissingPackageID, config)
	}
	return id, nil
}

func runCmdOut(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}
