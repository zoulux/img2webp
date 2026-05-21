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
	// Check the main quality window (now ordered: q, q-3, q+3)
	if candidates[0].Quality != 85 || candidates[1].Quality != 82 || candidates[2].Quality != 88 {
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
	if graphic[0].Quality != 90 {
		t.Fatalf("graphic default quality = %d, want 90", graphic[0].Quality)
	}

	clamped := BuildCandidates(analyze.KindPhoto, analyze.Features{}, 1, ModeAuto)
	if clamped[0].Quality != 2 {
		t.Fatalf("first candidate quality = %d, want 2 (adjusted from 1)", clamped[0].Quality)
	}
	clamped = BuildCandidates(analyze.KindPhoto, analyze.Features{}, 100, ModeAuto)
	if clamped[2].Quality != 100 {
		t.Fatalf("high quality candidate = %d, want 100", clamped[2].Quality)
	}
}

func TestBuildCandidatesKeepsLowQualityCandidatesDistinctAtClampBoundary(t *testing.T) {
	candidates := BuildCandidates(analyze.KindPhoto, analyze.Features{}, 1, ModeAuto)
	// At quality=1, qualityWindow adjusts: mid=2, low=1, high=4
	// Now ordered: q=2, q-3=1, q+3=4
	got := []int{candidates[0].Quality, candidates[1].Quality, candidates[2].Quality}
	want := []int{2, 1, 4}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("qualities[%d] = %d, want %d (all=%v)", i, got[i], want[i], got)
		}
	}
}

func TestBuildCandidatesKeepsHighQualityCandidatesDistinctAtClampBoundary(t *testing.T) {
	candidates := BuildCandidates(analyze.KindPhoto, analyze.Features{}, 100, ModeAuto)
	// At quality=100, qualityWindow gives: mid=99, low=97, high=100
	// Now ordered: q=99, q-3=97, q+3=100
	got := []int{candidates[0].Quality, candidates[1].Quality, candidates[2].Quality}
	want := []int{99, 97, 100}
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

func TestBuildCandidateGroupsFirstIsMiddleQuality(t *testing.T) {
	f := analyze.Features{Width: 1920, Height: 1080}
	groups := BuildCandidateGroups(analyze.KindPhoto, f, 82, ModeAuto)

	// First should be middle quality
	if groups.First.Quality != 82 {
		t.Fatalf("First quality = %d, want 82", groups.First.Quality)
	}
	if groups.First.PassName != "q" {
		t.Fatalf("First PassName = %q, want q", groups.First.PassName)
	}

	// IfPass should have lower quality candidates
	if len(groups.IfPass) != 3 {
		t.Fatalf("len(IfPass) = %d, want 3", len(groups.IfPass))
	}
	for i, c := range groups.IfPass {
		if c.Quality >= groups.First.Quality {
			t.Fatalf("IfPass[%d] quality %d should be < First quality %d", i, c.Quality, groups.First.Quality)
		}
	}

	// IfFail should have higher quality candidates
	if len(groups.IfFail) != 1 {
		t.Fatalf("len(IfFail) = %d, want 1", len(groups.IfFail))
	}
	for i, c := range groups.IfFail {
		if c.Quality <= groups.First.Quality {
			t.Fatalf("IfFail[%d] quality %d should be > First quality %d", i, c.Quality, groups.First.Quality)
		}
	}
}

func TestBuildCandidateGroupsLosslessReturnsOnlyFirst(t *testing.T) {
	groups := BuildCandidateGroups(analyze.KindPhoto, analyze.Features{}, 75, ModeLossless)

	if !groups.First.Lossless {
		t.Fatal("First should be lossless")
	}
	if len(groups.IfPass) != 0 || len(groups.IfFail) != 0 || len(groups.Special) != 0 {
		t.Fatal("Lossless mode should have no other candidates")
	}
}

func TestBuildCandidateGroupsGraphicHasSpecial(t *testing.T) {
	groups := BuildCandidateGroups(analyze.KindGraphic, analyze.Features{Width: 800, Height: 600}, 0, ModeAuto)

	if len(groups.Special) != 1 {
		t.Fatalf("len(Special) = %d, want 1", len(groups.Special))
	}
	if groups.Special[0].PassName != "near-lossless" {
		t.Fatalf("Special PassName = %q, want near-lossless", groups.Special[0].PassName)
	}
}
