package app

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zoulux/img2webp/internal/cli"
	"github.com/zoulux/img2webp/internal/report"
)

func TestRunDryRunBuildsPlanWithoutWritingOutput(t *testing.T) {
	inputDir := t.TempDir()
	mustWriteFile(t, filepath.Join(inputDir, "nested", "hero.jpg"), []byte("not-a-real-image"))
	outputDir := filepath.Join(t.TempDir(), "out")

	cfg := cli.Config{InputPath: inputDir, OutputDir: outputDir, DryRun: true}
	app := New(nil)
	result, err := app.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Summary.TotalFiles != 1 {
		t.Fatalf("TotalFiles = %d, want 1", result.Summary.TotalFiles)
	}
	if result.Summary.Skipped != 1 {
		t.Fatalf("Skipped = %d, want 1", result.Summary.Skipped)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "nested", "hero.webp")); !os.IsNotExist(err) {
		t.Fatalf("output should not exist during dry run, stat err = %v", err)
	}
}

func TestRunCopiesExistingWebPWhenReencodeDisabled(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "out")
	inputPath := filepath.Join(inputDir, "nested", "already.webp")
	content := []byte("fake-webp-bits")
	mustWriteFile(t, inputPath, content)

	cfg := cli.Config{InputPath: inputDir, OutputDir: outputDir, ReencodeWebP: false}
	app := New(nil)
	result, err := app.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Summary.TotalFiles != 1 {
		t.Fatalf("TotalFiles = %d, want 1", result.Summary.TotalFiles)
	}
	if result.Summary.Skipped != 1 {
		t.Fatalf("Skipped = %d, want 1", result.Summary.Skipped)
	}

	outPath := filepath.Join(outputDir, "nested", "already.webp")
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", outPath, err)
	}
	if string(got) != string(content) {
		t.Fatalf("copied contents = %q, want %q", got, content)
	}
}

func TestRunReturnsFailureResultForUndecodableRaster(t *testing.T) {
	inputDir := t.TempDir()
	mustWriteFile(t, filepath.Join(inputDir, "broken.png"), []byte("not-a-png"))

	cfg := cli.Config{InputPath: inputDir, OutputDir: filepath.Join(t.TempDir(), "out")}
	app := New(nil)
	result, err := app.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Summary.TotalFiles != 1 || result.Summary.Failed != 1 {
		t.Fatalf("unexpected summary: %+v", result.Summary)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(result.Results))
	}
	if result.Results[0].Status != report.StatusFailed {
		t.Fatalf("status = %q, want %q", result.Results[0].Status, report.StatusFailed)
	}
}

func TestRunReturnsContextErrorAfterRecordingCanceledFile(t *testing.T) {
	inputDir := t.TempDir()
	inputPath := filepath.Join(inputDir, "slow.png")
	mustWritePNG(t, inputPath)
	outputDir := filepath.Join(t.TempDir(), "out")
	startedFile := filepath.Join(t.TempDir(), "cwebp-started")
	scriptPath := writeBlockingScript(t, startedFile, "5")

	originalReleaseCWebP := releaseCWebP
	releaseCWebP = func(string) (string, error) {
		return scriptPath, nil
	}
	defer func() {
		releaseCWebP = originalReleaseCWebP
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultCh := make(chan Result, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := New(nil).Run(ctx, cli.Config{InputPath: inputDir, OutputDir: outputDir})
		resultCh <- result
		errCh <- err
	}()

	waitForFile(t, startedFile)
	cancel()

	result := <-resultCh
	err := <-errCh
	if err != context.Canceled {
		t.Fatalf("Run() error = %v, want %v", err, context.Canceled)
	}
	if result.Summary.TotalFiles != 1 || result.Summary.Failed != 1 {
		t.Fatalf("unexpected summary: %+v", result.Summary)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(result.Results))
	}
	if result.Results[0].InputPath != inputPath {
		t.Fatalf("InputPath = %q, want %q", result.Results[0].InputPath, inputPath)
	}
	if result.Results[0].Status != report.StatusFailed {
		t.Fatalf("status = %q, want %q", result.Results[0].Status, report.StatusFailed)
	}
	if !strings.Contains(result.Results[0].Message, context.Canceled.Error()) && !strings.Contains(result.Results[0].Message, "signal: killed") {
		t.Fatalf("message %q does not contain %q or %q", result.Results[0].Message, context.Canceled, "signal: killed")
	}
}

func TestRunReturnsContextErrorWithoutScanningWhenAlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := New(nil).Run(ctx, cli.Config{InputPath: t.TempDir(), OutputDir: t.TempDir()})
	if err != context.Canceled {
		t.Fatalf("Run() error = %v, want %v", err, context.Canceled)
	}
	if result.Summary.TotalFiles != 0 {
		t.Fatalf("TotalFiles = %d, want 0", result.Summary.TotalFiles)
	}
	if len(result.Results) != 0 {
		t.Fatalf("len(Results) = %d, want 0", len(result.Results))
	}
}

func TestEnsureBinaryPathReturnsReleaseError(t *testing.T) {
	originalReleaseCWebP := releaseCWebP
	releaseCWebP = func(string) (string, error) {
		return "", context.Canceled
	}
	defer func() {
		releaseCWebP = originalReleaseCWebP
	}()

	var binaryPath string
	_, err := ensureBinaryPath(&binaryPath)
	if err != context.Canceled {
		t.Fatalf("ensureBinaryPath() error = %v, want %v", err, context.Canceled)
	}
	if binaryPath != "" {
		t.Fatalf("cached binaryPath = %q, want empty", binaryPath)
	}
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func mustWritePNG(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(%q): %v", path, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("png.Encode(%q): %v", path, err)
	}
}

func writeBlockingScript(t *testing.T, startedFile, seconds string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "blocking.sh")
	content := "#!/bin/sh\ntouch \"" + startedFile + "\"\nsleep " + seconds + "\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
	return path
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		} else if !os.IsNotExist(err) {
			t.Fatalf("Stat(%q): %v", path, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("timed out waiting for %q: %v", path, err)
	}
}
