package eletrocromo

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Pinned workspaced release used when bootstrapping the ensure helper.
// Bump intentionally; checksums must match the release assets.
const (
	workspacedBootstrapVersion = "0.12.0"
	workspacedReleaseBase      = "https://github.com/lucasew/workspaced/releases/download/" + workspacedBootstrapVersion
)

// SHA-256 digests for workspaced 0.12.0 release archives (from checksums.txt).
var workspacedAssetSHA256 = map[string]string{
	"workspaced_Darwin_arm64.tar.gz":  "58d6fc78b93a978ac3b65b870191df661cc9aa88e919dada104b087a53f1f9df",
	"workspaced_Darwin_x86_64.tar.gz": "1b043f3c5cc397940f85c588a7ae45ca63d2913f829f7b994067c01ca37912e4",
	"workspaced_Linux_arm64.tar.gz":   "1ac21a0f0946eed0c62b03f7bbe662cfc9298457f6a394862ffdd3437aa715d0",
	"workspaced_Linux_i386.tar.gz":    "496aa4606ff11790b18fe30fe904acb879f745594e2f30a24dd9e0d49f7f3153",
	"workspaced_Linux_x86_64.tar.gz":  "0b8d9110ce44653eb08aa7b0423aa5ca4344f8d3f57e3d1e7732326d193b678b",
	"workspaced_Windows_arm64.zip":    "163791fe707df2f95c89b92b10f858363690c2cf5c9af13b1da4a13c0e45ed88",
	"workspaced_Windows_i386.zip":     "4bfefcf1c2eb40f39e053f047f4a8ab835886bf9615f43b48aea97159422bf2e",
	"workspaced_Windows_x86_64.zip":   "979ff0ab8cdd78f0dcc32838eab3d6c41855ace48edb5f816f5eb59e92a23756",
}

// bootstrapHTTP is used for release downloads. No overall Timeout (large
// assets may take a while); ResponseHeaderTimeout and body idle timeout
// catch stuck connections.
var bootstrapHTTP = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          4,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   30 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

// httpGet is bootstrapHTTP.Do wrapper; tests may override.
var httpGet = func(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return bootstrapHTTP.Do(req)
}

func workspacedAssetName() (string, error) {
	var osPart string
	switch runtime.GOOS {
	case "linux":
		osPart = "Linux"
	case "darwin":
		osPart = "Darwin"
	case "windows":
		osPart = "Windows"
	default:
		return "", fmt.Errorf("bootstrap workspaced: unsupported GOOS %s", runtime.GOOS)
	}
	var archPart string
	switch runtime.GOARCH {
	case "amd64":
		archPart = "x86_64"
	case "arm64":
		archPart = "arm64"
	case "386":
		archPart = "i386"
	default:
		return "", fmt.Errorf("bootstrap workspaced: unsupported GOARCH %s", runtime.GOARCH)
	}
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return "workspaced_" + osPart + "_" + archPart + ext, nil
}

func workspacedCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "eletrocromo", "workspaced", workspacedBootstrapVersion), nil
}

// bootstrapWorkspaced downloads a pinned workspaced release into the user cache
// (if missing), verifies the archive SHA-256, extracts the binary, and returns
// its path.
func bootstrapWorkspaced(ctx context.Context) (string, error) {
	asset, err := workspacedAssetName()
	if err != nil {
		return "", err
	}
	wantSum, ok := workspacedAssetSHA256[asset]
	if !ok {
		return "", fmt.Errorf("bootstrap workspaced: no pinned checksum for %s", asset)
	}

	dir, err := workspacedCacheDir()
	if err != nil {
		return "", fmt.Errorf("bootstrap workspaced: cache dir: %w", err)
	}
	binName := "workspaced"
	if runtime.GOOS == "windows" {
		binName = "workspaced.exe"
	}
	binPath := filepath.Join(dir, binName)
	if st, err := os.Stat(binPath); err == nil && st.Mode().IsRegular() && st.Size() > 0 {
		return binPath, nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("bootstrap workspaced: mkdir: %w", err)
	}

	url := workspacedReleaseBase + "/" + asset
	resp, err := httpGet(ctx, url)
	if err != nil {
		return "", fmt.Errorf("bootstrap workspaced: download: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return "", fmt.Errorf("bootstrap workspaced: download %s: HTTP %s", url, resp.Status)
	}

	archivePath := filepath.Join(dir, asset)
	f, err := os.OpenFile(archivePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		_ = resp.Body.Close()
		return "", err
	}
	h := sha256.New()
	w := io.MultiWriter(f, h)
	// body.Close closes the HTTP response body (idleTimeoutReader wraps it).
	body := newIdleTimeoutReader(resp.Body, downloadIdleTimeout)
	_, copyErr := io.Copy(w, body)
	bodyCloseErr := body.Close()
	fileCloseErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(archivePath)
		return "", fmt.Errorf("bootstrap workspaced: write archive: %w", copyErr)
	}
	if bodyCloseErr != nil {
		_ = os.Remove(archivePath)
		return "", fmt.Errorf("bootstrap workspaced: close body: %w", bodyCloseErr)
	}
	if fileCloseErr != nil {
		_ = os.Remove(archivePath)
		return "", fileCloseErr
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != wantSum {
		_ = os.Remove(archivePath)
		return "", fmt.Errorf("bootstrap workspaced: checksum mismatch for %s: got %s want %s", asset, got, wantSum)
	}

	if err := extractWorkspacedBinary(archivePath, binPath, binName); err != nil {
		_ = os.Remove(binPath)
		return "", fmt.Errorf("bootstrap workspaced: extract: %w", err)
	}
	if err := os.Chmod(binPath, 0o755); err != nil {
		_ = os.Remove(binPath)
		return "", err
	}
	return binPath, nil
}

func extractWorkspacedBinary(archivePath, destPath, binName string) error {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractZipBinary(archivePath, destPath, binName)
	}
	return extractTarGzBinary(archivePath, destPath, binName)
}

func extractTarGzBinary(archivePath, destPath, binName string) (err error) {
	f, openErr := os.Open(archivePath)
	if openErr != nil {
		return openErr
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	gz, gzErr := gzip.NewReader(f)
	if gzErr != nil {
		return gzErr
	}
	defer func() {
		if cerr := gz.Close(); err == nil {
			err = cerr
		}
	}()
	tr := tar.NewReader(gz)
	for {
		hdr, nextErr := tr.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return nextErr
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		base := filepath.Base(hdr.Name)
		if base != binName && base != "workspaced" {
			continue
		}
		out, outErr := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if outErr != nil {
			return outErr
		}
		_, copyErr := io.Copy(out, tr)
		closeErr := out.Close()
		if copyErr != nil {
			_ = os.Remove(destPath)
			return copyErr
		}
		if closeErr != nil {
			_ = os.Remove(destPath)
			return closeErr
		}
		return nil
	}
	return fmt.Errorf("binary %q not found in archive", binName)
}

func extractZipBinary(archivePath, destPath, binName string) (err error) {
	r, openErr := zip.OpenReader(archivePath)
	if openErr != nil {
		return openErr
	}
	defer func() {
		if cerr := r.Close(); err == nil {
			err = cerr
		}
	}()
	for _, zf := range r.File {
		base := filepath.Base(zf.Name)
		if base != binName && base != "workspaced" && base != "workspaced.exe" {
			continue
		}
		rc, rcErr := zf.Open()
		if rcErr != nil {
			return rcErr
		}
		out, outErr := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if outErr != nil {
			_ = rc.Close()
			return outErr
		}
		_, copyErr := io.Copy(out, rc)
		rcCloseErr := rc.Close()
		closeErr := out.Close()
		if copyErr != nil {
			_ = os.Remove(destPath)
			return copyErr
		}
		if rcCloseErr != nil {
			_ = os.Remove(destPath)
			return rcCloseErr
		}
		if closeErr != nil {
			_ = os.Remove(destPath)
			return closeErr
		}
		return nil
	}
	return fmt.Errorf("binary %q not found in archive", binName)
}
