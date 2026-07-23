package apkgen

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lewtec/eletrocromo/internal/version"
)

// BuildOptions drives a full Android APK build from an eletrocromo app.
type BuildOptions struct {
	// Config is package identity + go_main (after LoadConfig/Merge/flags).
	Config Config
	// BaseDir resolves relative Config.GoMain (config file directory or cwd).
	BaseDir string
	// WorkDir holds the generated Gradle project. Empty → temp under os.TempDir.
	WorkDir string
	// KeepWorkDir leaves WorkDir in place (always true when WorkDir is set by caller).
	KeepWorkDir bool
	// OutAPK is the destination .apk path (directories created). Required for full build.
	OutAPK string
	// GoOnly stops after multiarch libeletrocromo.so (no Gradle / no SDK).
	GoOnly bool
	// Stdout/Stderr for subprocess logs (default os.Stdout/Stderr).
	Stdout io.Writer
	Stderr io.Writer
}

// BuildResult is the outcome of Build.
type BuildResult struct {
	// APKPath is the copied/final APK (empty if GoOnly).
	APKPath string
	// WorkDir is the generated Android project path.
	WorkDir string
	// JNILibs lists built native binaries.
	JNILibs []string
}

// Build scaffolds the Android host, cross-compiles the Go app into jniLibs,
// and (unless GoOnly) runs Gradle assembleDebug and copies the APK to OutAPK.
func Build(opts BuildOptions) (*BuildResult, error) {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	cfg, err := opts.Config.withDefaults()
	if err != nil {
		return nil, err
	}
	baseDir := strings.TrimSpace(opts.BaseDir)
	if baseDir == "" {
		baseDir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	goMain, err := ResolveGoMain(cfg.GoMain, baseDir)
	if err != nil {
		return nil, err
	}

	// Stamp version from app tree (git describe) when config omitted version_*.
	vi := version.ResolveDir(goMain)
	if strings.TrimSpace(opts.Config.VersionName) == "" {
		cfg.VersionName = vi.AndroidName()
	}
	if opts.Config.VersionCode <= 0 {
		cfg.VersionCode = version.AndroidCodeFrom(vi.Version, version.GitCommitCount(goMain))
	}
	fmt.Fprintf(stdout, "eletrocromo: version %s (code %d)\n", cfg.VersionName, cfg.VersionCode)

	workDir, ephemeral, err := resolveWorkDir(opts)
	if err != nil {
		return nil, err
	}
	// Ephemeral work dirs: keep on failure (inspect logs), keep for --go-only
	// or KeepWorkDir; delete after a successful full APK copy.
	var buildErr error
	defer func() {
		if ephemeral && buildErr == nil && !opts.KeepWorkDir && !opts.GoOnly {
			_ = os.RemoveAll(workDir)
		}
	}()

	// Host go_main in generated config is absolute so scripts work from workDir.
	genCfg := cfg
	genCfg.GoMain = goMain

	fmt.Fprintf(stdout, "eletrocromo: generating Android host in %s\n", workDir)
	if err := Create(Options{
		OutDir: workDir,
		Force:  true, // work dir is ours or full rebuild
		Config: genCfg,
	}); err != nil {
		buildErr = fmt.Errorf("generate host: %w", err)
		return nil, buildErr
	}

	fmt.Fprintf(stdout, "eletrocromo: building Go binary for %v\n", genCfg.abis())
	libs, err := BuildGoLibs(workDir, goMain, genCfg.abis(), vi, stdout, stderr)
	if err != nil {
		buildErr = err
		return nil, buildErr
	}

	result := &BuildResult{WorkDir: workDir, JNILibs: libs}
	if opts.GoOnly {
		fmt.Fprintf(stdout, "eletrocromo: --go-only: skipped Gradle (jniLibs ready under %s)\n", workDir)
		return result, nil
	}

	outAPK := strings.TrimSpace(opts.OutAPK)
	if outAPK == "" {
		buildErr = fmt.Errorf("out apk path is required (or use --go-only)")
		return nil, buildErr
	}
	outAPK, err = filepath.Abs(outAPK)
	if err != nil {
		buildErr = err
		return nil, buildErr
	}

	fmt.Fprintf(stdout, "eletrocromo: assembling debug APK…\n")
	apk, err := AssembleDebug(workDir, stdout, stderr)
	if err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := os.MkdirAll(filepath.Dir(outAPK), 0o755); err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := copyFile(apk, outAPK); err != nil {
		buildErr = fmt.Errorf("copy apk: %w", err)
		return nil, buildErr
	}
	result.APKPath = outAPK
	fmt.Fprintf(stdout, "eletrocromo: APK → %s\n", outAPK)
	return result, nil
}

func resolveWorkDir(opts BuildOptions) (workDir string, cleanup bool, err error) {
	if strings.TrimSpace(opts.WorkDir) != "" {
		abs, err := filepath.Abs(opts.WorkDir)
		if err != nil {
			return "", false, err
		}
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return "", false, err
		}
		return abs, false, nil
	}
	dir, err := os.MkdirTemp("", "eletrocromo-android-*")
	if err != nil {
		return "", false, err
	}
	return dir, true, nil
}

// BuildGoLibs cross-compiles the app into workDir/app/src/main/jniLibs/<abi>/libeletrocromo.so.
// stamp is injected via -ldflags -X (goreleaser-style) when apps import internal/version.
func BuildGoLibs(workDir, goMainDir string, abis []string, stamp version.Info, stdout, stderr io.Writer) ([]string, error) {
	if len(abis) == 0 {
		abis = DefaultABIs
	}
	ldflags := stamp.GoBuildLdflags()
	var out []string
	for _, abi := range abis {
		goarch, ok := abiToGOARCH[abi]
		if !ok {
			return nil, fmt.Errorf("unsupported abi %q", abi)
		}
		destDir := filepath.Join(workDir, "app", "src", "main", "jniLibs", abi)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return nil, err
		}
		dest := filepath.Join(destDir, "libeletrocromo.so")
		fmt.Fprintf(stdout, "  → %s (GOARCH=%s)\n", abi, goarch)
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags", ldflags, "-o", dest, ".")
		cmd.Dir = goMainDir
		cmd.Env = append(os.Environ(),
			"CGO_ENABLED=0",
			"GOOS=android",
			"GOARCH="+goarch,
		)
		if goarch == "arm" {
			cmd.Env = append(cmd.Env, "GOARM=7")
		}
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("go build %s (GOARCH=%s CGO_ENABLED=0): %w\nnote: pure Go android builds typically only support arm64-v8a without an NDK; set abis in eletrocromo.json", abi, goarch, err)
		}
		out = append(out, dest)
	}
	return out, nil
}

// AssembleDebug runs Gradle assembleDebug in workDir and returns the debug APK path.
func AssembleDebug(workDir string, stdout, stderr io.Writer) (string, error) {
	if err := requireJDK(); err != nil {
		return "", err
	}
	sdk, err := androidSDK()
	if err != nil {
		return "", err
	}
	if err := writeLocalProperties(workDir, sdk); err != nil {
		return "", err
	}

	gradle, err := resolveGradle(workDir)
	if err != nil {
		return "", err
	}
	args := append(gradle[1:], "assembleDebug", "--stacktrace")
	cmd := exec.Command(gradle[0], args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(),
		"ANDROID_HOME="+sdk,
		"ANDROID_SDK_ROOT="+sdk,
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gradle assembleDebug: %w\n(work dir left at %s)", err, workDir)
	}

	// Standard AGP debug output.
	candidates := []string{
		filepath.Join(workDir, "app", "build", "outputs", "apk", "debug", "app-debug.apk"),
	}
	matches, _ := filepath.Glob(filepath.Join(workDir, "app", "build", "outputs", "apk", "debug", "*.apk"))
	candidates = append(candidates, matches...)
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
			return p, nil
		}
	}
	return "", fmt.Errorf("gradle succeeded but no debug APK under app/build/outputs/apk/debug (work dir %s)", workDir)
}

func requireJDK() error {
	if _, err := exec.LookPath("java"); err != nil {
		return fmt.Errorf("java not found on PATH (need JDK 17+ for Gradle): %w", err)
	}
	return nil
}

func androidSDK() (string, error) {
	for _, key := range []string{"ANDROID_HOME", "ANDROID_SDK_ROOT"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			if st, err := os.Stat(v); err == nil && st.IsDir() {
				return v, nil
			}
			return "", fmt.Errorf("%s=%q is not a directory", key, v)
		}
	}
	// Common user installs.
	home, err := os.UserHomeDir()
	if err == nil {
		for _, rel := range []string{
			"Android/Sdk",
			"Library/Android/sdk",
			"AppData/Local/Android/Sdk",
		} {
			p := filepath.Join(home, rel)
			if st, err := os.Stat(p); err == nil && st.IsDir() {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("Android SDK not found: set ANDROID_HOME (or ANDROID_SDK_ROOT) to the SDK root")
}

func writeLocalProperties(workDir, sdk string) error {
	// Gradle local.properties wants forward slashes / escaped backslashes.
	sdkProp := filepath.ToSlash(sdk)
	body := fmt.Sprintf("## Generated by eletrocromo android build\nsdk.dir=%s\n", sdkProp)
	return os.WriteFile(filepath.Join(workDir, "local.properties"), []byte(body), 0o644)
}

func resolveGradle(workDir string) ([]string, error) {
	wrapper := filepath.Join(workDir, "gradlew")
	if st, err := os.Stat(wrapper); err == nil && !st.IsDir() {
		return []string{wrapper}, nil
	}
	// Bootstrap wrapper if system gradle exists.
	if g, err := exec.LookPath("gradle"); err == nil {
		// Prefer system gradle directly (no wrapper jar in template).
		return []string{g}, nil
	}
	return nil, fmt.Errorf("neither ./gradlew nor gradle on PATH; install Gradle 8.9+ (or Android Studio) and JDK 17")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// DefaultOutAPK suggests dist/<last-label>-debug.apk under cwd.
func DefaultOutAPK(packageID, cwd string) string {
	label := packageID
	if i := strings.LastIndex(packageID, "."); i >= 0 {
		label = packageID[i+1:]
	}
	if label == "" {
		label = "app"
	}
	return filepath.Join(cwd, "dist", label+"-debug.apk")
}
