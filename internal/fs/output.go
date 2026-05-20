package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func BuildOutputPath(rootInput, inputPath, outputDir string) (string, error) {
	rel, err := filepath.Rel(rootInput, inputPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("input %q is outside root %q", inputPath, rootInput)
	}

	if rel == "." {
		rel = filepath.Base(inputPath)
	}

	base := strings.TrimSuffix(rel, filepath.Ext(rel)) + ".webp"
	return filepath.Join(outputDir, base), nil
}

func EnsureParentDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

func CopyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := EnsureParentDir(dst); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
