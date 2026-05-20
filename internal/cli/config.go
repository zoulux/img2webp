package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

type Mode string

const (
	ModeAuto     Mode = "auto"
	ModePhoto    Mode = "photo"
	ModeGraphic  Mode = "graphic"
	ModeLossless Mode = "lossless"
)

type Config struct {
	InputPath    string
	OutputDir    string
	Quality      int
	Overwrite    bool
	Workers      int
	ReencodeWebP bool
	Mode         Mode
	DryRun       bool
}

var errExactlyOneInputPath = fmt.Errorf("exactly one input path is required")
var errHelpRequested = flag.ErrHelp

func ParseConfig(args []string) (Config, error) {
	if wantsHelp(args) {
		return Config{}, errHelpRequested
	}

	fs := flag.NewFlagSet("img2webp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	cfg := Config{OutputDir: "output", Mode: ModeAuto}

	fs.StringVar(&cfg.OutputDir, "o", cfg.OutputDir, "output directory")
	fs.StringVar(&cfg.OutputDir, "output", cfg.OutputDir, "output directory")
	fs.IntVar(&cfg.Quality, "q", 0, "quality 1-100, 0 means auto")
	fs.IntVar(&cfg.Quality, "quality", 0, "quality 1-100, 0 means auto")
	fs.BoolVar(&cfg.Overwrite, "overwrite", false, "overwrite existing outputs")
	fs.IntVar(&cfg.Workers, "workers", 0, "worker count, 0 means cpu count")
	fs.BoolVar(&cfg.ReencodeWebP, "reencode-webp", false, "re-encode input webp files")
	mode := string(cfg.Mode)
	fs.StringVar(&mode, "mode", mode, "auto|photo|graphic|lossless")
	fs.BoolVar(&cfg.DryRun, "dry-run", false, "print plan without writing files")

	parsedArgs, inputPath, err := normalizeArgs(args)
	if err != nil {
		return Config{}, err
	}

	if err := fs.Parse(parsedArgs); err != nil {
		return Config{}, err
	}

	if fs.NArg() != 0 {
		return Config{}, errExactlyOneInputPath
	}
	cfg.InputPath = inputPath

	if cfg.Quality < 0 || cfg.Quality > 100 {
		return Config{}, fmt.Errorf("quality must be between 0 and 100: %d", cfg.Quality)
	}

	cfg.Mode = Mode(mode)
	switch cfg.Mode {
	case ModeAuto, ModePhoto, ModeGraphic, ModeLossless:
	default:
		return Config{}, fmt.Errorf("unsupported mode: %s", cfg.Mode)
	}

	return cfg, nil
}

func normalizeArgs(args []string) ([]string, string, error) {
	var inputPath string
	parsed := make([]string, 0, len(args))
	positionalOnly := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "" {
			continue
		}
		if arg == "--" {
			positionalOnly = true
			continue
		}
		if positionalOnly || !strings.HasPrefix(arg, "-") {
			if inputPath != "" {
				return nil, "", errExactlyOneInputPath
			}
			inputPath = arg
			continue
		}
		parsed = append(parsed, arg)
		if flagTakesValue(arg) && i+1 < len(args) {
			i++
			parsed = append(parsed, args[i])
		}
	}

	if inputPath == "" {
		inputPath = "."
	}

	return parsed, inputPath, nil
}

func wantsHelp(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			return true
		}
	}
	return false
}

func HelpText() string {
	return strings.TrimSpace(`Usage: img2webp [flags] [input]

Arguments:
  input                     input file or directory (default ".")

Flags:
  -o, --output string       output directory (default "output")
  -q, --quality int         quality 1-100, 0 means auto
      --overwrite           overwrite existing outputs
      --workers int         worker count, 0 means cpu count
      --reencode-webp       re-encode input webp files
      --mode string         auto|photo|graphic|lossless (default "auto")
      --dry-run             print plan without writing files
  -h, --help                show help
`)
}

func flagTakesValue(arg string) bool {
	name := strings.TrimLeft(arg, "-")
	if strings.Contains(name, "=") {
		return false
	}
	switch name {
	case "o", "output", "q", "quality", "workers", "mode":
		return true
	default:
		return false
	}
}
