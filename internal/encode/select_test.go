package encode

import (
	"testing"

	"github.com/zoulux/img2webp/internal/analyze"
)

func TestPickSmallestPassingNoPassingCandidate(t *testing.T) {
	results := []CandidateResult{
		{Path: "a.webp", Size: 1200, Scores: Scores{Pass: false}},
		{Path: "b.webp", Size: 900, Scores: Scores{Pass: false}},
	}

	picked, ok := PickSmallestPassing(results)
	if ok {
		t.Fatalf("picked %+v, want no passing candidate", picked)
	}
}

func TestPickSmallestPassingPrefersFirstOnTie(t *testing.T) {
	results := []CandidateResult{
		{Path: "first.webp", Size: 900, Scores: Scores{Pass: true}},
		{Path: "second.webp", Size: 900, Scores: Scores{Pass: true}},
	}

	picked, ok := PickSmallestPassing(results)
	if !ok {
		t.Fatal("expected a passing candidate")
	}
	if picked.Path != "first.webp" {
		t.Fatalf("picked %q, want first.webp", picked.Path)
	}
}

func TestEvaluatePassUsesThresholdsByKind(t *testing.T) {
	if !EvaluatePass(analyze.KindPhoto, Scores{SSIM: 0.92, Edge: 0.88}) {
		t.Fatal("expected photo scores at threshold to pass")
	}
	if EvaluatePass(analyze.KindGraphic, Scores{SSIM: 0.92, Edge: 0.87}) {
		t.Fatal("expected graphic score below edge threshold to fail")
	}
	if EvaluatePass(analyze.KindTransparentGraphic, Scores{SSIM: 0.97, Edge: 0.95, AlphaEdge: 0.97}) {
		t.Fatal("expected transparent graphic score below alpha threshold to fail")
	}
}

func TestPickSmallestPassingWithSizeCheck(t *testing.T) {
	tests := []struct {
		name       string
		results    []CandidateResult
		sourceSize int64
		wantPath   string
		wantOk     bool
	}{
		{
			name:       "prefers_smaller_than_source",
			results:    []CandidateResult{{Path: "a.webp", Size: 500, Scores: Scores{Pass: true}}, {Path: "b.webp", Size: 1500, Scores: Scores{Pass: true}}},
			sourceSize: 1000,
			wantPath:   "a.webp",
			wantOk:     true,
		},
		{
			name:       "picks_smallest_overall_if_none_smaller_than_source",
			results:    []CandidateResult{{Path: "a.webp", Size: 1500, Scores: Scores{Pass: true}}, {Path: "b.webp", Size: 2000, Scores: Scores{Pass: true}}},
			sourceSize: 1000,
			wantPath:   "a.webp",
			wantOk:     true,
		},
		{
			name:       "no_passing_candidates",
			results:    []CandidateResult{{Path: "a.webp", Size: 500, Scores: Scores{Pass: false}}},
			sourceSize: 1000,
			wantPath:   "",
			wantOk:     false,
		},
		{
			name:       "ignores_non_passing_when_picking_smaller",
			results:    []CandidateResult{{Path: "a.webp", Size: 500, Scores: Scores{Pass: false}}, {Path: "b.webp", Size: 800, Scores: Scores{Pass: true}}},
			sourceSize: 1000,
			wantPath:   "b.webp",
			wantOk:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			picked, ok := PickSmallestPassingWithSizeCheck(tt.results, tt.sourceSize)
			if ok != tt.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOk)
			}
			if ok && picked.Path != tt.wantPath {
				t.Fatalf("picked.Path = %q, want %q", picked.Path, tt.wantPath)
			}
		})
	}
}
