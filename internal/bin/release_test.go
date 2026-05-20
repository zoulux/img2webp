package bin

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReleaseExecutableWritesCachedFile(t *testing.T) {
	tmp := t.TempDir()
	released, err := releaseExecutable(tmp, "cwebp", executableBundle{Executable: []byte("#!/bin/sh\nexit 0\n")})
	if err != nil {
		t.Fatalf("releaseExecutable returned error: %v", err)
	}

	info, err := os.Stat(released)
	if err != nil {
		t.Fatalf("Stat(%q): %v", released, err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("perm = %#o, want %#o", info.Mode().Perm(), 0o755)
	}
	if filepath.Base(released) != "cwebp" {
		t.Fatalf("base = %q, want cwebp", filepath.Base(released))
	}
}

func TestReleaseExecutableCacheHitReturnsExistingFile(t *testing.T) {
	tmp := t.TempDir()
	blob := []byte("#!/bin/sh\nexit 0\n")

	first, err := releaseExecutable(tmp, "cwebp", executableBundle{Executable: blob})
	if err != nil {
		t.Fatalf("first releaseExecutable returned error: %v", err)
	}
	second, err := releaseExecutable(tmp, "cwebp", executableBundle{Executable: blob})
	if err != nil {
		t.Fatalf("second releaseExecutable returned error: %v", err)
	}
	if second != first {
		t.Fatalf("second path = %q, want %q", second, first)
	}

	info, err := os.Stat(second)
	if err != nil {
		t.Fatalf("Stat(%q): %v", second, err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("mode = %v, want regular file", info.Mode())
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("perm = %#o, want %#o", info.Mode().Perm(), 0o755)
	}
}

func TestReleaseExecutableRepairsPermissionsOnCacheHit(t *testing.T) {
	tmp := t.TempDir()
	blob := []byte("#!/bin/sh\nexit 0\n")
	path := releasePath(tmp, "cwebp", executableBundle{Executable: blob})

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, blob, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	released, err := releaseExecutable(tmp, "cwebp", executableBundle{Executable: blob})
	if err != nil {
		t.Fatalf("releaseExecutable returned error: %v", err)
	}
	if released != path {
		t.Fatalf("released path = %q, want %q", released, path)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q): %v", path, err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("perm = %#o, want %#o", info.Mode().Perm(), 0o755)
	}
}

func TestReleaseExecutableReplacesCorruptedCacheHit(t *testing.T) {
	tmp := t.TempDir()
	blob := []byte("#!/bin/sh\nexit 0\n")
	path := releasePath(tmp, "cwebp", executableBundle{Executable: blob})

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("corrupted\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	released, err := releaseExecutable(tmp, "cwebp", executableBundle{Executable: blob})
	if err != nil {
		t.Fatalf("releaseExecutable returned error: %v", err)
	}
	if released != path {
		t.Fatalf("released path = %q, want %q", released, path)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if string(got) != string(blob) {
		t.Fatalf("contents = %q, want %q", got, blob)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q): %v", path, err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("perm = %#o, want %#o", info.Mode().Perm(), 0o755)
	}
}

func TestReleaseExecutableRejectsExistingDirectory(t *testing.T) {
	tmp := t.TempDir()
	blob := []byte("#!/bin/sh\nexit 0\n")
	path := releasePath(tmp, "cwebp", executableBundle{Executable: blob})

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}

	_, err := releaseExecutable(tmp, "cwebp", executableBundle{Executable: blob})
	if err == nil {
		t.Fatal("releaseExecutable returned nil error, want error")
	}
}

func TestReleaseCWebPSignsExecutable(t *testing.T) {
	requireDarwin(t)

	tmp := t.TempDir()

	released, err := ReleaseCWebP(tmp)
	if err != nil {
		t.Fatalf("ReleaseCWebP returned error: %v", err)
	}
	verifyCodeSignature(t, released)
}

func TestEmbeddedCWebPUnsupportedPlatform(t *testing.T) {
	_, err := embeddedCWebP("freebsd", "amd64")
	if err == nil {
		t.Fatal("embeddedCWebP returned nil error, want error")
	}

	var unsupported ErrUnsupportedPlatform
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %T, want ErrUnsupportedPlatform", err)
	}
	if unsupported.GOOS != "freebsd" || unsupported.GOARCH != "amd64" {
		t.Fatalf("error = %+v, want freebsd/amd64", unsupported)
	}
}

func TestEmbeddedCWebPSupportedPlatforms(t *testing.T) {
	_, err := embeddedCWebP(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatalf("embeddedCWebP(%s, %s) returned error: %v", runtime.GOOS, runtime.GOARCH, err)
	}
}

func requireDarwin(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only test")
	}
}

func verifyCodeSignature(t *testing.T, path string) {
	t.Helper()
	cmd := exec.Command("codesign", "--verify", "--verbose=4", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("codesign verify %q: %v\n%s", path, err, output)
	}
}

func releasePath(cacheRoot, name string, bundle executableBundle) string {
	sum := sha256.New()
	sum.Write(bundle.Executable)
	for _, relPath := range sortedSupportPaths(bundle.SupportFiles) {
		sum.Write([]byte(relPath))
		sum.Write(bundle.SupportFiles[relPath])
	}
	return filepath.Join(cacheRoot, hex.EncodeToString(sum.Sum(nil)[:8]), name)
}
