package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zoulux/img2webp/internal/app"
	"github.com/zoulux/img2webp/internal/cli"
	"github.com/zoulux/img2webp/internal/report"
)

type runner interface {
	Run(context.Context, cli.Config) (app.Result, error)
}

var newApp = func(opts *app.Options) runner {
	return app.New(opts)
}

var notifyContext = signal.NotifyContext

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	cfg, err := cli.ParseConfig(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, cli.HelpText())
			return 0
		}
		fmt.Fprintln(stderr, err)
		return 2
	}

	ctx, stop := notifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pw := report.NewProgressWriter(stderr)
	start := time.Now()
	result, err := newApp(&app.Options{OnProgress: pw.Advance}).Run(ctx, cfg)
	pw.Finish()
	elapsed := time.Since(start).Round(time.Millisecond)

	if result.Summary.TotalFiles > 0 {
		report.PrintSummary(stdout, result.Summary)
		fmt.Fprintf(stdout, "elapsed: %v\n", elapsed)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if result.Summary.TotalFiles == 0 {
		report.PrintSummary(stdout, result.Summary)
		fmt.Fprintf(stdout, "elapsed: %v\n", elapsed)
	}
	return 0
}
