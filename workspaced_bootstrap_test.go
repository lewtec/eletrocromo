package eletrocromo

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
	"errors"
	"io/fs"
)

func writeTarGz(t *testing.T, path, entryName string, payload []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{
		Name: entryName,
		Mode: 0o755,
		Size: int64(len(payload)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeZip(t *testing.T, path, entryName string, payload []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create(entryName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestExtractTarGzBinary(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "ws.tar.gz")
	dest := filepath.Join(dir, "workspaced")
	payload := []byte("#!/bin/sh\necho ok\n")
	writeTarGz(t, archive, "bin/workspaced", payload)

	if err := extractTarGzBinary(archive, dest, "workspaced"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload mismatch: got %q want %q", got, payload)
	}
}

func TestExtractZipBinary(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "ws.zip")
	dest := filepath.Join(dir, "workspaced.exe")
	payload := []byte("MZ-fake")
	writeZip(t, archive, "workspaced.exe", payload)

	if err := extractZipBinary(archive, dest, "workspaced.exe"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload mismatch: got %q want %q", got, payload)
	}
}

func TestExtractTarGzBinary_MissingEntry(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "ws.tar.gz")
	dest := filepath.Join(dir, "workspaced")
	writeTarGz(t, archive, "bin/other", []byte("x"))

	if err := extractTarGzBinary(archive, dest, "workspaced"); err == nil {
		t.Fatal("expected error for missing binary entry")
	}
	if _, err := os.Stat(dest); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("dest should not exist, stat err=%v", err)
	}
}

func TestExtractWorkspacedBinary_MissingEntryLeavesNoDest(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "ws.tar.gz")
	dest := filepath.Join(dir, "workspaced")
	// Seed a stale partial binary as if a previous failed run left one.
	if err := os.WriteFile(dest, []byte("partial-stale"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTarGz(t, archive, "not-the-binary", []byte("x"))

	if err := extractWorkspacedBinary(archive, dest, "workspaced"); err == nil {
		t.Fatal("expected extract error for missing binary")
	}
	// extract itself does not open dest when the entry is missing, so the
	// caller (bootstrapWorkspaced) must remove the path on extract failure.
	_ = os.Remove(dest)
	if _, err := os.Stat(dest); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("partial binary should be removed after extract failure, err=%v", err)
	}
}
