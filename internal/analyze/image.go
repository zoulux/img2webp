package analyze

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"

	_ "golang.org/x/image/webp"
)

type Features struct {
	Width              int
	Height             int
	HasAlpha           bool
	UniqueColorDensity float64
	EdgeDensity        float64
	TextureScore       float64
}

type Kind string

const (
	KindPhoto              Kind = "photo"
	KindGraphic            Kind = "graphic"
	KindTransparentGraphic Kind = "transparent-graphic"
)

func AnalyzeFile(path string) (Features, error) {
	file, err := os.Open(path)
	if err != nil {
		return Features{}, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return Features{}, err
	}
	return AnalyzeImage(img), nil
}

func AnalyzeImage(img image.Image) Features {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width == 0 || height == 0 {
		return Features{}
	}

	stepY := max(1, height/64)
	stepX := max(1, width/64)

	hasAlpha := false
	for y := bounds.Min.Y; y < bounds.Max.Y && !hasAlpha; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if uint8(a>>8) < 255 {
				hasAlpha = true
				break
			}
		}
	}

	var sampleCount int
	uniqueBuckets := map[[3]uint8]struct{}{}
	var edgeAccum float64
	var textureAccum float64

	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			r, g, b, _ := img.At(x, y).RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
			uniqueBuckets[[3]uint8{r8 >> 3, g8 >> 3, b8 >> 3}] = struct{}{}
			sampleCount++

			if x+1 < bounds.Max.X && y+1 < bounds.Max.Y {
				r2, g2, b2, _ := img.At(x+1, y).RGBA()
				r3, g3, b3, _ := img.At(x, y+1).RGBA()
				dx := absDiff8(r8, uint8(r2>>8)) + absDiff8(g8, uint8(g2>>8)) + absDiff8(b8, uint8(b2>>8))
				dy := absDiff8(r8, uint8(r3>>8)) + absDiff8(g8, uint8(g3>>8)) + absDiff8(b8, uint8(b3>>8))
				edge := (dx + dy) / (255.0 * 6.0)
				edgeAccum += edge
				textureAccum += math.Min(edge*1.4, 1.0)
			}
		}
	}

	denom := float64(max(sampleCount, 1))
	return Features{
		Width:              width,
		Height:             height,
		HasAlpha:           hasAlpha,
		UniqueColorDensity: float64(len(uniqueBuckets)) / denom,
		EdgeDensity:        edgeAccum / denom,
		TextureScore:       textureAccum / denom,
	}
}

func absDiff8(a, b uint8) float64 {
	if a > b {
		return float64(a - b)
	}
	return float64(b - a)
}

