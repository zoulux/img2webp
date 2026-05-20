package bin

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
)

func ReleaseCWebP(cacheRoot string) (string, error) {
	bundle, err := embeddedCWebP(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", err
	}

	return releaseExecutable(cacheRoot, "cwebp", bundle)
}

func releaseExecutable(cacheRoot, name string, bundle executableBundle) (string, error) {
	sum := sha256.New()
	sum.Write(bundle.Executable)
	for _, relPath := range sortedSupportPaths(bundle.SupportFiles) {
		sum.Write([]byte(relPath))
		sum.Write(bundle.SupportFiles[relPath])
	}
	dir := filepath.Join(cacheRoot, hex.EncodeToString(sum.Sum(nil)[:8]))
	path := filepath.Join(dir, name)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := syncReleasedFile(path, bundle.Executable); err != nil {
		return "", err
	}
	for _, relPath := range sortedSupportPaths(bundle.SupportFiles) {
		supportPath := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(supportPath), 0o755); err != nil {
			return "", err
		}
		if err := syncReleasedFile(supportPath, bundle.SupportFiles[relPath]); err != nil {
			return "", err
		}
	}

	return path, nil
}

func syncReleasedFile(path string, blob []byte) error {
	if info, err := os.Stat(path); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("cached executable path is not a regular file: %s", path)
		}
		cached, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(cached) != string(blob) {
			if err := os.WriteFile(path, blob, 0o755); err != nil {
				return err
			}
		} else if info.Mode().Perm() != 0o755 {
			if err := os.Chmod(path, 0o755); err != nil {
				return err
			}
			return nil
		} else {
			return nil
		}
		if err := os.Chmod(path, 0o755); err != nil {
			return err
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.WriteFile(path, blob, 0o755); err != nil {
		return err
	}
	return nil
}

func sortedSupportPaths(files map[string][]byte) []string {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	return paths
}
