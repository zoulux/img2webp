package report

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

type ProgressStage string

const (
	StageAnalyzing ProgressStage = "analyzing"
	StageEncoding  ProgressStage = "encoding"
	StageScoring   ProgressStage = "scoring"
	StageDone      ProgressStage = "done"
)

type Progress struct {
	FileIndex     int
	TotalFiles    int
	Stage         ProgressStage
	StageProgress float64
	InputPath     string
}

// ProgressWriter displays a unified progress bar that aggregates progress across all files.
type ProgressWriter struct {
	w       io.Writer
	mu      sync.Mutex
	shown   bool
	lastPct int
}

func NewProgressWriter(w io.Writer) *ProgressWriter {
	return &ProgressWriter{w: w, lastPct: -1}
}

func (p *ProgressWriter) Advance(progress Progress) {
	if p == nil || p.w == nil || progress.TotalFiles <= 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	// FileIndex/TotalFiles represents accumulated progress (scaled by 100)
	pct := progress.FileIndex * 100 / progress.TotalFiles
	if pct > 100 {
		pct = 100
	}

	// Only update display when percentage changes
	if pct != p.lastPct {
		p.lastPct = pct
		fmt.Fprintf(p.w, "\r[%s] %3d%%",
			renderProgressBar(pct, 100, 20),
			pct)
		p.shown = true
	}
}

func (p *ProgressWriter) Finish() {
	if p == nil || p.w == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.shown {
		return
	}
	fmt.Fprintln(p.w)
}

func renderProgressBar(completed, total, width int) string {
	if total <= 0 || width <= 0 {
		return ""
	}
	filled := completed * width / total
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("=", filled) + strings.Repeat("-", width-filled)
}
