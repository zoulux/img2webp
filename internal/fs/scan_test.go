package fs

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCollectInputsRecursesAndFiltersSupportedExtensions(t *testing.T) {
	tmp := t.TempDir()
	mustWriteFile(t, filepath.Join(tmp, "a.jpg"))
	mustWriteFile(t, filepath.Join(tmp, "b.txt"))
	mustWriteFile(t, filepath.Join(tmp, "nested", "c.PNG"))
	mustWriteFile(t, filepath.Join(tmp, "nested", "d.webp"))

	files, err := CollectInputs(tmp)
	if err != nil {
		t.Fatalf("CollectInputs returned error: %v", err)
	}

	want := []string{
		filepath.Join(tmp, "a.jpg"),
		filepath.Join(tmp, "nested", "c.PNG"),
		filepath.Join(tmp, "nested", "d.webp"),
	}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("files = %#v, want %#v", files, want)
	}
}

func TestCollectInputsReturnsSupportedSingleFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "hero.JPEG")
	mustWriteFile(t, path)

	files, err := CollectInputs(path)
	if err != nil {
		t.Fatalf("CollectInputs returned error: %v", err)
	}

	want := []string{path}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("files = %#v, want %#v", files, want)
	}
}

func TestCollectInputsReturnsEmptyForUnsupportedSingleFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "notes.txt")
	mustWriteFile(t, path)

	files, err := CollectInputs(path)
	if err != nil {
		t.Fatalf("CollectInputs returned error: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("len(files) = %d, want 0", len(files))
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
