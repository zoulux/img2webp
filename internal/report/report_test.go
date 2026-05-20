package report

import (
	"bytes"
	"testing"
)

func TestSummaryAccumulatesSizes(t *testing.T) {
	s := Summary{}
	s.Add(Result{SourceBytes: 2000, OutputBytes: 1000, Status: StatusSuccess})
	s.Add(Result{SourceBytes: 500, OutputBytes: 500, Status: StatusSkipped})

	if s.TotalFiles != 2 || s.Success != 1 || s.Skipped != 1 || s.Failed != 0 {
		t.Fatalf("unexpected summary counts: %+v", s)
	}
	if s.SourceBytes != 2500 || s.OutputBytes != 1500 {
		t.Fatalf("unexpected summary sizes: %+v", s)
	}
}

func TestSummaryCountsFailures(t *testing.T) {
	s := Summary{}
	s.Add(Result{Status: StatusFailed, SourceBytes: 10})

	if s.TotalFiles != 1 || s.Failed != 1 {
		t.Fatalf("unexpected summary after failure: %+v", s)
	}
}

func TestPrintSummaryFormatsCountsAndSizes(t *testing.T) {
	var buf bytes.Buffer
	PrintSummary(&buf, Summary{
		TotalFiles:  3,
		Success:     1,
		Skipped:     1,
		Failed:      1,
		SourceBytes: 2500,
		OutputBytes: 1500,
	})

	if got, want := buf.String(), "files=3 success=1 skipped=1 failed=1 saved=1000 (40.0%)\n"; got != want {
		t.Fatalf("PrintSummary() = %q, want %q", got, want)
	}
}
