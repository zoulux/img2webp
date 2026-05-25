package encode

import "github.com/zoulux/img2webp/internal/analyze"

type Scores struct {
	SSIM      float64
	Edge      float64
	AlphaEdge float64
	Pass      bool
}

type CandidateResult struct {
	Path   string
	Size   int64
	Scores Scores
}

func PickSmallestPassing(results []CandidateResult) (CandidateResult, bool) {
	var best CandidateResult
	found := false
	for _, result := range results {
		if !result.Scores.Pass {
			continue
		}
		if !found || result.Size < best.Size {
			best = result
			found = true
		}
	}
	return best, found
}

// PickSmallestPassingWithSizeCheck picks the smallest passing candidate that is smaller than the source.
// If no passing candidate is smaller than source, it picks the smallest passing candidate overall.
func PickSmallestPassingWithSizeCheck(results []CandidateResult, sourceSize int64) (CandidateResult, bool) {
	var bestSmaller CandidateResult
	var bestOverall CandidateResult
	foundSmaller := false
	foundOverall := false

	for _, result := range results {
		if !result.Scores.Pass {
			continue
		}

		// Track the best overall (regardless of size)
		if !foundOverall || result.Size < bestOverall.Size {
			bestOverall = result
			foundOverall = true
		}

		// Track the best that is smaller than source
		if result.Size < sourceSize {
			if !foundSmaller || result.Size < bestSmaller.Size {
				bestSmaller = result
				foundSmaller = true
			}
		}
	}

	// Prefer a smaller file over the best overall
	if foundSmaller {
		return bestSmaller, true
	}
	return bestOverall, foundOverall
}

func EvaluatePass(kind analyze.Kind, s Scores) bool {
	switch kind {
	case analyze.KindTransparentGraphic:
		return s.SSIM >= 0.97 && s.Edge >= 0.95 && s.AlphaEdge >= 0.98
	case analyze.KindGraphic:
		return s.SSIM >= 0.92 && s.Edge >= 0.88
	default:
		return s.SSIM >= 0.92 && s.Edge >= 0.88
	}
}
