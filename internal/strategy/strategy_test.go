package strategy

import (
	"testing"

	"github.com/zoulux/img2webp/internal/analyze"
)

func TestBuildCandidatesForPhotoIncludesAroundTargetQuality(t *testing.T) {
	f := analyze.Features{Width: 3840, Height: 2160}
	candidates := BuildCandidates(analyze.KindPhoto, f, 85, ModeAuto)
	if len(candidates) != 5 {
		t.Fatalf("len(candidates) = %d, want 5", len(candidates))
	}
	// Check the main quality window
	if candidates[0].Quality != 82 || candidates[1].Quality != 85 || candidates[2].Quality != 88 {
		t.Fatalf("unexpected main candidate qualities: %+v", candidates[:3])
	}
	// Check the additional lower quality candidates
	if candidates[3].Quality != 79 || candidates[4].Quality != 73 {
		t.Fatalf("unexpected lower quality candidates: %+v", candidates[3:])
	}
}

func TestBuildCandidatesForTransparentGraphicAddsAlphaCandidate(t *testing.T) {
	f := analyze.Features{Width: 128, Height: 128, HasAlpha: true}
	candidates := BuildCandidates(analyze.KindTransparentGraphic, f, 92, ModeAuto)
	foundHighAlpha := false
	for _, c := range candidates {
		if c.AlphaQuality == 100 {
			foundHighAlpha = true
		}
	}
	if !foundHighAlpha {
		t.Fatal("expected transparent graphic candidates to include alpha_q=100")
	}
}

func TestBuildCandidatesModeLosslessReturnsSingleLosslessCandidate(t *testing.T) {
	candidates := BuildCandidates(analyze.KindPhoto, analyze.Features{}, 75, ModeLossless)
	if len(candidates) != 1 {
		t.Fatalf("len(candidates) = %d, want 1", len(candidates))
	}
	candidate := candidates[0]
	if !candidate.Lossless || candidate.Quality != 100 || candidate.AlphaQuality != 100 {
		t.Fatalf("unexpected lossless candidate: %+v", candidate)
	}
}

func TestBuildCandidatesModeGraphicOverridesDetectedKind(t *testing.T) {
	candidates := BuildCandidates(analyze.KindPhoto, analyze.Features{Width: 800, Height: 600}, 0, ModeGraphic)
	if got := candidates[len(candidates)-1].PassName; got != "near-lossless" {
		t.Fatalf("last candidate PassName = %q, want near-lossless", got)
	}
	for _, candidate := range candidates {
		if candidate.Kind != analyze.KindGraphic {
			t.Fatalf("candidate Kind = %q, want %q", candidate.Kind, analyze.KindGraphic)
		}
	}
}

func TestBuildCandidatesUsesDefaultQualityAndClampsExtremes(t *testing.T) {
	graphic := BuildCandidates(analyze.KindGraphic, analyze.Features{Width: 640, Height: 480}, 0, ModeAuto)
	if graphic[1].Quality != 90 {
		t.Fatalf("graphic default quality = %d, want 90", graphic[1].Quality)
	}

	clamped := BuildCandidates(analyze.KindPhoto, analyze.Features{}, 1, ModeAuto)
	if clamped[0].Quality != 1 {
		t.Fatalf("low quality candidate = %d, want 1", clamped[0].Quality)
	}
	clamped = BuildCandidates(analyze.KindPhoto, analyze.Features{}, 100, ModeAuto)
	if clamped[2].Quality != 100 {
		t.Fatalf("high quality candidate = %d, want 100", clamped[2].Quality)
	}
}

func TestBuildCandidatesKeepsLowQualityCandidatesDistinctAtClampBoundary(t *testing.T) {
	candidates := BuildCandidates(analyze.KindPhoto, analyze.Features{}, 1, ModeAuto)
	// At quality=1, we get: q-3=1(clamped), q=1(clamped)->adjusted to 2, q+3=4, q-6=1(clamped), q-12=1(clamped)
	// The qualityWindow function adjusts mid when low==mid
	got := []int{candidates[0].Quality, candidates[1].Quality, candidates[2].Quality}
	want := []int{1, 2, 4}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("qualities[%d] = %d, want %d (all=%v)", i, got[i], want[i], got)
		}
	}
}

func TestBuildCandidatesKeepsHighQualityCandidatesDistinctAtClampBoundary(t *testing.T) {
	candidates := BuildCandidates(analyze.KindPhoto, analyze.Features{}, 100, ModeAuto)
	// At quality=100, the main window is: q-3=97, q=99, q+3=100(clamped)->adjusted to 99->98
	got := []int{candidates[0].Quality, candidates[1].Quality, candidates[2].Quality}
	want := []int{97, 99, 100}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("qualities[%d] = %d, want %d (all=%v)", i, got[i], want[i], got)
		}
	}
}

func TestBuildCandidatesModeGraphicPreservesTransparentGraphicKind(t *testing.T) {
	candidates := BuildCandidates(analyze.KindTransparentGraphic, analyze.Features{Width: 800, Height: 600, HasAlpha: true}, 0, ModeGraphic)
	for _, candidate := range candidates {
		if candidate.Kind != analyze.KindTransparentGraphic {
			t.Fatalf("candidate Kind = %q, want %q", candidate.Kind, analyze.KindTransparentGraphic)
		}
	}
	if got := candidates[len(candidates)-1].PassName; got != "alpha-near-lossless" {
		t.Fatalf("last candidate PassName = %q, want alpha-near-lossless", got)
	}
}
