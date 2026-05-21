package strategy

import "github.com/zoulux/img2webp/internal/analyze"

// CandidateGroup organizes candidates for staged encoding
type CandidateGroup struct {
	First     Candidate   // Test this first (usually mid quality)
	IfPass    []Candidate // If first passes, test these to find smaller
	IfFail    []Candidate // If first fails, test these for better quality
	Special   []Candidate // Special candidates (near-lossless), always test if first fails
}

func BuildCandidates(kind analyze.Kind, f analyze.Features, requestedQuality int, mode Mode) []Candidate {
	kind = selectKind(kind, mode)
	if mode == ModeLossless {
		return []Candidate{{Kind: kind, Quality: 100, Method: 6, AlphaQuality: 100, Lossless: true, PassName: "lossless"}}
	}

	quality := requestedQuality
	if quality == 0 {
		quality = defaultQuality(kind, f)
	}
	quality = clampQuality(quality)

	lowQuality, midQuality, highQuality := qualityWindow(quality)

	base := []Candidate{
		{Kind: kind, Quality: midQuality, Method: 6, AlphaQuality: 90, PassName: "q"},
		{Kind: kind, Quality: lowQuality, Method: 6, AlphaQuality: 90, PassName: "q-3"},
		{Kind: kind, Quality: highQuality, Method: 6, AlphaQuality: 90, PassName: "q+3"},
	}

	lowerQuality := clampQuality(quality - 6)
	lowestQuality := clampQuality(quality - 12)
	base = append(base,
		Candidate{Kind: kind, Quality: lowerQuality, Method: 6, AlphaQuality: 90, PassName: "q-6"},
		Candidate{Kind: kind, Quality: lowestQuality, Method: 6, AlphaQuality: 90, PassName: "q-12"},
	)

	switch kind {
	case analyze.KindGraphic:
		base = append(base, Candidate{Kind: kind, Quality: quality, Method: 6, AlphaQuality: 90, NearLossless: 80, PassName: "near-lossless"})
	case analyze.KindTransparentGraphic:
		base = append(base, Candidate{Kind: kind, Quality: quality, Method: 6, AlphaQuality: 100, NearLossless: 85, PassName: "alpha-near-lossless"})
	}

	return base
}

// BuildAllCandidates returns all candidates for parallel encoding
func BuildAllCandidates(kind analyze.Kind, f analyze.Features, requestedQuality int, mode Mode) []Candidate {
	groups := BuildCandidateGroups(kind, f, requestedQuality, mode)

	candidates := []Candidate{groups.First}
	candidates = append(candidates, groups.IfPass...)
	candidates = append(candidates, groups.IfFail...)
	candidates = append(candidates, groups.Special...)

	return candidates
}

// BuildCandidateGroups organizes candidates for staged encoding optimization
func BuildCandidateGroups(kind analyze.Kind, f analyze.Features, requestedQuality int, mode Mode) CandidateGroup {
	kind = selectKind(kind, mode)
	if mode == ModeLossless {
		return CandidateGroup{
			First:  Candidate{Kind: kind, Quality: 100, Method: 6, AlphaQuality: 100, Lossless: true, PassName: "lossless"},
			IfPass: nil,
			IfFail: nil,
		}
	}

	quality := requestedQuality
	if quality == 0 {
		quality = defaultQuality(kind, f)
	}
	quality = clampQuality(quality)

	lowQuality, midQuality, highQuality := qualityWindow(quality)
	lowerQuality := clampQuality(quality - 6)
	lowestQuality := clampQuality(quality - 12)

	// First: test middle quality
	first := Candidate{Kind: kind, Quality: midQuality, Method: 6, AlphaQuality: 90, PassName: "q"}

	// IfPass: lower quality candidates (to find smaller file)
	ifPass := []Candidate{
		{Kind: kind, Quality: lowQuality, Method: 6, AlphaQuality: 90, PassName: "q-3"},
		{Kind: kind, Quality: lowerQuality, Method: 6, AlphaQuality: 90, PassName: "q-6"},
		{Kind: kind, Quality: lowestQuality, Method: 6, AlphaQuality: 90, PassName: "q-12"},
	}

	// IfFail: higher quality candidates
	ifFail := []Candidate{
		{Kind: kind, Quality: highQuality, Method: 6, AlphaQuality: 90, PassName: "q+3"},
	}

	// Special: near-lossless variants
	var special []Candidate
	switch kind {
	case analyze.KindGraphic:
		special = []Candidate{{Kind: kind, Quality: quality, Method: 6, AlphaQuality: 90, NearLossless: 80, PassName: "near-lossless"}}
	case analyze.KindTransparentGraphic:
		special = []Candidate{{Kind: kind, Quality: quality, Method: 6, AlphaQuality: 100, NearLossless: 85, PassName: "alpha-near-lossless"}}
	}

	return CandidateGroup{
		First:  first,
		IfPass: ifPass,
		IfFail: ifFail,
		Special: special,
	}
}

func defaultQuality(kind analyze.Kind, f analyze.Features) int {
	pixels := f.Width * f.Height
	switch kind {
	case analyze.KindTransparentGraphic:
		if pixels <= 512*512 {
			return 94
		}
		return 90
	case analyze.KindGraphic:
		if pixels >= 1920*1080 {
			return 85
		}
		return 90
	default:
		if pixels >= 1920*1080 {
			return 75
		}
		return 82
	}
}

func clampQuality(q int) int {
	if q < 1 {
		return 1
	}
	if q > 100 {
		return 100
	}
	return q
}

func qualityWindow(quality int) (low, mid, high int) {
	mid = clampQuality(quality)
	low = clampQuality(mid - 3)
	high = clampQuality(mid + 3)

	if low == mid {
		mid = clampQuality(low + 1)
	}
	if high == mid {
		mid = clampQuality(high - 1)
	}

	return low, mid, high
}

func selectKind(kind analyze.Kind, mode Mode) analyze.Kind {
	switch mode {
	case ModePhoto:
		return analyze.KindPhoto
	case ModeGraphic:
		if kind == analyze.KindTransparentGraphic {
			return analyze.KindTransparentGraphic
		}
		return analyze.KindGraphic
	default:
		return kind
	}
}
