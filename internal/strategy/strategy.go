package strategy

import "github.com/zoulux/img2webp/internal/analyze"

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
		{Kind: kind, Quality: lowQuality, Method: 6, AlphaQuality: 90, PassName: "q-3"},
		{Kind: kind, Quality: midQuality, Method: 6, AlphaQuality: 90, PassName: "q"},
		{Kind: kind, Quality: highQuality, Method: 6, AlphaQuality: 90, PassName: "q+3"},
	}

	// Add lower quality candidates to ensure size reduction
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
