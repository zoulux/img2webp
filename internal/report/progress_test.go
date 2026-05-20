package report

import (
	"bytes"
	"strings"
	"testing"
)

func TestProgressWriterAdvancePrintsProgressBar(t *testing.T) {
	var buf bytes.Buffer
	pw := NewProgressWriter(&buf)

	pw.Advance(Progress{FileIndex: 25, TotalFiles: 100, Stage: StageDone, StageProgress: 1.0, InputPath: "/tmp/hero.png"})
	output := buf.String()
	if !strings.HasPrefix(output, "\r[") {
		t.Fatalf("output = %q, want \\r[ prefix", output)
	}
	if !strings.Contains(output, "%") {
		t.Fatalf("output = %q, want percentage", output)
	}
	if !strings.Contains(output, "25%") {
		t.Fatalf("output = %q, want '25%%'", output)
	}
}

func TestProgressWriterFinishPrintsNewlineWhenShown(t *testing.T) {
	var buf bytes.Buffer
	pw := NewProgressWriter(&buf)

	pw.Advance(Progress{FileIndex: 0, TotalFiles: 2, Stage: StageDone, StageProgress: 1, InputPath: "a.png"})
	buf.Reset()
	pw.Finish()
	if got := buf.String(); got != "\n" {
		t.Fatalf("Finish() output = %q, want \\n", got)
	}
}

func TestProgressWriterFinishNoopWhenNotShown(t *testing.T) {
	var buf bytes.Buffer
	pw := NewProgressWriter(&buf)

	pw.Finish()
	if got := buf.String(); got != "" {
		t.Fatalf("Finish() output = %q, want empty", got)
	}
}

func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		completed int
		total     int
		width     int
		want      string
	}{
		{0, 100, 20, "--------------------"},
		{100, 100, 20, "===================="},
		{50, 100, 20, "==========----------"},
		{25, 100, 20, "=====---------------"},
		{0, 0, 20, ""},
	}
	for _, tt := range tests {
		got := renderProgressBar(tt.completed, tt.total, tt.width)
		if got != tt.want {
			t.Errorf("renderProgressBar(%d, %d, %d) = %q, want %q", tt.completed, tt.total, tt.width, got, tt.want)
		}
	}
}

func TestProgressWriterAdvanceClampsValues(t *testing.T) {
	var buf bytes.Buffer
	pw := NewProgressWriter(&buf)

	pw.Advance(Progress{FileIndex: 10, TotalFiles: 3, Stage: StageEncoding, StageProgress: 2.0, InputPath: "x.png"})
	output := buf.String()
	if !strings.Contains(output, "100%") {
		t.Fatalf("output = %q, want 100%% (clamped)", output)
	}
}

func TestProgressWriterNilReceiver(t *testing.T) {
	var pw *ProgressWriter
	pw.Advance(Progress{FileIndex: 0, TotalFiles: 2, Stage: StageDone, InputPath: "a.png"})
	pw.Finish()
}

func TestProgressWriterOnlyUpdatesOnPctChange(t *testing.T) {
	var buf bytes.Buffer
	pw := NewProgressWriter(&buf)

	pw.Advance(Progress{FileIndex: 0, TotalFiles: 100, Stage: StageAnalyzing, StageProgress: 0.5, InputPath: "a.png"})
	first := buf.String()
	buf.Reset()

	pw.Advance(Progress{FileIndex: 0, TotalFiles: 100, Stage: StageAnalyzing, StageProgress: 0.51, InputPath: "a.png"})
	second := buf.String()

	if first == "" {
		t.Fatal("first output is empty")
	}
	if second != "" {
		t.Fatalf("second output = %q, want empty (same percentage)", second)
	}
}
