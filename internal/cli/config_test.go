package cli

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseConfigDefaultsOutputDir(t *testing.T) {
	cfg, err := ParseConfig([]string{"./assets"})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if cfg.InputPath != "./assets" {
		t.Fatalf("InputPath = %q, want ./assets", cfg.InputPath)
	}

	if cfg.OutputDir != "output" {
		t.Fatalf("OutputDir = %q, want output", cfg.OutputDir)
	}

	if cfg.Mode != ModeAuto {
		t.Fatalf("Mode = %q, want %q", cfg.Mode, ModeAuto)
	}

	if cfg.Quality != 0 {
		t.Fatalf("Quality = %d, want 0", cfg.Quality)
	}
}

func TestNormalizeArgsPreservesFlagsAroundInputPath(t *testing.T) {
	parsed, inputPath, err := normalizeArgs([]string{"--output", "dist", "./assets", "-q", "85", "--dry-run", "--mode=photo"})
	if err != nil {
		t.Fatalf("normalizeArgs returned error: %v", err)
	}

	if inputPath != "./assets" {
		t.Fatalf("inputPath = %q, want ./assets", inputPath)
	}

	want := []string{"--output", "dist", "-q", "85", "--dry-run", "--mode=photo"}
	if !reflect.DeepEqual(parsed, want) {
		t.Fatalf("parsed = %#v, want %#v", parsed, want)
	}
}

func TestNormalizeArgsDefaultsToCurrentDirectory(t *testing.T) {
	parsed, inputPath, err := normalizeArgs([]string{"--output", "dist"})
	if err != nil {
		t.Fatalf("normalizeArgs returned error: %v", err)
	}

	if inputPath != "." {
		t.Fatalf("inputPath = %q, want .", inputPath)
	}

	want := []string{"--output", "dist"}
	if !reflect.DeepEqual(parsed, want) {
		t.Fatalf("parsed = %#v, want %#v", parsed, want)
	}
}

func TestNormalizeArgsRejectsMultipleInputPaths(t *testing.T) {
	_, _, err := normalizeArgs([]string{"./assets", "./more-assets"})
	if !errors.Is(err, errExactlyOneInputPath) {
		t.Fatalf("errors.Is(%v, errExactlyOneInputPath) = false", err)
	}
}

func TestParseConfigAcceptsFlagsBeforeInputPath(t *testing.T) {
	cfg, err := ParseConfig([]string{"-q", "85", "--mode", "photo", "./assets"})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if cfg.Quality != 85 {
		t.Fatalf("Quality = %d, want 85", cfg.Quality)
	}
	if cfg.Mode != ModePhoto {
		t.Fatalf("Mode = %q, want %q", cfg.Mode, ModePhoto)
	}
}

func TestParseConfigAcceptsValuedFlagsAroundInputPath(t *testing.T) {
	cfg, err := ParseConfig([]string{"--output", "dist", "./assets", "--workers", "3"})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if cfg.OutputDir != "dist" {
		t.Fatalf("OutputDir = %q, want dist", cfg.OutputDir)
	}
	if cfg.Workers != 3 {
		t.Fatalf("Workers = %d, want 3", cfg.Workers)
	}
}

func TestParseConfigAcceptsBooleanFlagsAfterInputPath(t *testing.T) {
	cfg, err := ParseConfig([]string{"./assets", "--overwrite", "--dry-run", "--reencode-webp"})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if !cfg.Overwrite {
		t.Fatal("Overwrite = false, want true")
	}
	if !cfg.DryRun {
		t.Fatal("DryRun = false, want true")
	}
	if !cfg.ReencodeWebP {
		t.Fatal("ReencodeWebP = false, want true")
	}
}

func TestParseConfigAcceptsEqualsFormFlagsMixedWithInputPath(t *testing.T) {
	cfg, err := ParseConfig([]string{"--quality=85", "./assets", "--mode=graphic", "--output=dist"})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if cfg.Quality != 85 {
		t.Fatalf("Quality = %d, want 85", cfg.Quality)
	}
	if cfg.Mode != ModeGraphic {
		t.Fatalf("Mode = %q, want %q", cfg.Mode, ModeGraphic)
	}
	if cfg.OutputDir != "dist" {
		t.Fatalf("OutputDir = %q, want dist", cfg.OutputDir)
	}
}

func TestParseConfigDefaultsToCurrentDirectory(t *testing.T) {
	cfg, err := ParseConfig([]string{"--output", "dist"})
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if cfg.InputPath != "." {
		t.Fatalf("InputPath = %q, want .", cfg.InputPath)
	}
	if cfg.OutputDir != "dist" {
		t.Fatalf("OutputDir = %q, want dist", cfg.OutputDir)
	}
}

func TestParseConfigRejectsMultipleInputPaths(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "repeated positional args", args: []string{"./assets", "./more-assets", "--dry-run"}},
		{name: "second positional after double dash", args: []string{"--", "./assets", "./more-assets"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseConfig(tt.args)
			if !errors.Is(err, errExactlyOneInputPath) {
				t.Fatalf("errors.Is(%v, errExactlyOneInputPath) = false", err)
			}
		})
	}
}

func TestParseConfigAcceptsInputPathAfterDoubleDash(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "standard path", args: []string{"--", "./assets"}, want: "./assets"},
		{name: "leading dash path", args: []string{"--", "-leading-dash-dir"}, want: "-leading-dash-dir"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseConfig(tt.args)
			if err != nil {
				t.Fatalf("ParseConfig returned error: %v", err)
			}

			if cfg.InputPath != tt.want {
				t.Fatalf("InputPath = %q, want %q", cfg.InputPath, tt.want)
			}
		})
	}
}

func TestParseConfigReturnsHelpErrorForHelpFlags(t *testing.T) {
	tests := [][]string{{"-h"}, {"--help"}}
	for _, args := range tests {
		_, err := ParseConfig(args)
		if !errors.Is(err, errHelpRequested) {
			t.Fatalf("ParseConfig(%v) error = %v, want help error", args, err)
		}
	}
}

func TestParseConfigRejectsOutOfRangeQuality(t *testing.T) {
	_, err := ParseConfig([]string{"./assets", "-q", "101"})
	if err == nil || err.Error() != "quality must be between 0 and 100: 101" {
		t.Fatalf("expected quality range error, got %v", err)
	}
}

func TestParseConfigRejectsUnknownMode(t *testing.T) {
	_, err := ParseConfig([]string{"./assets", "--mode", "vector"})
	if err == nil || err.Error() != "unsupported mode: vector" {
		t.Fatalf("expected unsupported mode error, got %v", err)
	}
}
