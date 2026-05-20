package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/zoulux/img2webp/internal/app"
	"github.com/zoulux/img2webp/internal/cli"
	"github.com/zoulux/img2webp/internal/report"
)

type stubRunner struct {
	result app.Result
	err    error
}

func (s stubRunner) Run(context.Context, cli.Config) (app.Result, error) {
	return s.result, s.err
}

func TestRunPrintsPartialSummaryBeforeError(t *testing.T) {
	inputDir := t.TempDir()
	originalNewApp := newApp
	originalNotifyContext := notifyContext
	defer func() {
		newApp = originalNewApp
		notifyContext = originalNotifyContext
	}()

	newApp = func(_ *app.Options) runner {
		return stubRunner{
			result: app.Result{Summary: report.Summary{TotalFiles: 1, Failed: 1, SourceBytes: 10}},
			err:    errors.New("context canceled"),
		}
	}
	notifyContext = func(ctx context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		return ctx, func() {}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{inputDir}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run() code = %d, want 1", code)
	}
	if got, want := stdout.String(), "files=1 success=0 skipped=0 failed=1 saved=10 (100.0%)\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got, want := stderr.String(), "context canceled\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestRunPrintsZeroSummaryOnSuccessWithoutFiles(t *testing.T) {
	inputDir := t.TempDir()
	originalNewApp := newApp
	originalNotifyContext := notifyContext
	defer func() {
		newApp = originalNewApp
		notifyContext = originalNotifyContext
	}()

	newApp = func(_ *app.Options) runner {
		return stubRunner{}
	}
	notifyContext = func(ctx context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		return ctx, func() {}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{inputDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run() code = %d, want 0", code)
	}
	if got, want := stdout.String(), "files=0 success=0 skipped=0 failed=0 saved=0 (0.0%)\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunPrintsHelpToStdout(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"-h"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run() code = %d, want 0", code)
	}
	if stdout.Len() == 0 {
		t.Fatal("stdout is empty, want help text")
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}
