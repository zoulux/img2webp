package strategy

import "github.com/zoulux/img2webp/internal/analyze"

type Mode string

const (
	ModeAuto     Mode = "auto"
	ModePhoto    Mode = "photo"
	ModeGraphic  Mode = "graphic"
	ModeLossless Mode = "lossless"
)

type Candidate struct {
	Kind         analyze.Kind
	Quality      int
	Method       int
	AlphaQuality int
	NearLossless int
	Lossless     bool
	PassName     string
}
