package fs

import (
	"path/filepath"
	"testing"
)

func TestBuildOutputPathPreservesRelativeStructure(t *testing.T) {
	root := filepath.Clean("/tmp/in")
	input := filepath.Join(root, "nested", "hero.png")
	output := filepath.Clean("/tmp/out")

	got, err := BuildOutputPath(root, input, output)
	if err != nil {
		t.Fatalf("BuildOutputPath returned error: %v", err)
	}

	want := filepath.Join(output, "nested", "hero.webp")
	if got != want {
		t.Fatalf("output path = %q, want %q", got, want)
	}
}

func TestBuildOutputPathRejectsInputOutsideRoot(t *testing.T) {
	root := filepath.Clean("/tmp/in")
	input := filepath.Clean("/tmp/other/hero.png")
	output := filepath.Clean("/tmp/out")

	_, err := BuildOutputPath(root, input, output)
	if err == nil {
		t.Fatal("expected error for input outside root")
	}
}

func TestBuildOutputPathUsesInputBasenameForSingleFileRoot(t *testing.T) {
	root := filepath.Clean("/tmp/in/hero.png")
	input := root
	output := filepath.Clean("/tmp/out")

	got, err := BuildOutputPath(root, input, output)
	if err != nil {
		t.Fatalf("BuildOutputPath returned error: %v", err)
	}

	want := filepath.Join(output, "hero.webp")
	if got != want {
		t.Fatalf("output path = %q, want %q", got, want)
	}
}
