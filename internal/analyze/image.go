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

func AnalyzeFile(path string) (Features, image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return Features{}, nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return Features{}, nil, err
	}
	return AnalyzeImage(img), img, nil
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

	var sampleCount int
	uniqueBuckets := map[[3]uint8]struct{}{}
	var edgeAccum float64
	var textureAccum float64

	// Fast path for NRGBA
	if nrgba, ok := img.(*image.NRGBA); ok {
		// Check alpha in all pixels (not just samples)
		hasAlpha := checkAlphaNRGBA(nrgba)

		for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
			for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
				i := y*nrgba.Stride + x*4
				r8, g8, b8 := nrgba.Pix[i], nrgba.Pix[i+1], nrgba.Pix[i+2]
				uniqueBuckets[[3]uint8{r8 >> 3, g8 >> 3, b8 >> 3}] = struct{}{}
				sampleCount++

				if x+1 < bounds.Max.X && y+1 < bounds.Max.Y {
					i2 := y*nrgba.Stride + (x+1)*4
					i3 := (y+1)*nrgba.Stride + x*4
					dx := absDiff8(r8, nrgba.Pix[i2]) + absDiff8(g8, nrgba.Pix[i2+1]) + absDiff8(b8, nrgba.Pix[i2+2])
					dy := absDiff8(r8, nrgba.Pix[i3]) + absDiff8(g8, nrgba.Pix[i3+1]) + absDiff8(b8, nrgba.Pix[i3+2])
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

	// Fast path for RGBA
	if rgba, ok := img.(*image.RGBA); ok {
		hasAlpha := checkAlphaRGBA(rgba)

		for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
			for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
				i := y*rgba.Stride + x*4
				r8, g8, b8 := rgba.Pix[i], rgba.Pix[i+1], rgba.Pix[i+2]
				uniqueBuckets[[3]uint8{r8 >> 3, g8 >> 3, b8 >> 3}] = struct{}{}
				sampleCount++

				if x+1 < bounds.Max.X && y+1 < bounds.Max.Y {
					i2 := y*rgba.Stride + (x+1)*4
					i3 := (y+1)*rgba.Stride + x*4
					dx := absDiff8(r8, rgba.Pix[i2]) + absDiff8(g8, rgba.Pix[i2+1]) + absDiff8(b8, rgba.Pix[i2+2])
					dy := absDiff8(r8, rgba.Pix[i3]) + absDiff8(g8, rgba.Pix[i3+1]) + absDiff8(b8, rgba.Pix[i3+2])
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

	// Fast path for YCbCr (no alpha)
	if ycbcr, ok := img.(*image.YCbCr); ok {
		for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
			for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
				yi := ycbcr.Y[y*ycbcr.YStride+x]
				uniqueBuckets[[3]uint8{yi >> 3, yi >> 3, yi >> 3}] = struct{}{}
				sampleCount++

				if x+1 < bounds.Max.X && y+1 < bounds.Max.Y {
					yi2 := ycbcr.Y[y*ycbcr.YStride+(x+1)]
					yi3 := ycbcr.Y[(y+1)*ycbcr.YStride+x]
					dx := absDiff8(yi, yi2)
					dy := absDiff8(yi, yi3)
					edge := (dx + dy) / (255.0 * 2.0)
					edgeAccum += edge
					textureAccum += math.Min(edge*1.4, 1.0)
				}
			}
		}

		denom := float64(max(sampleCount, 1))
		return Features{
			Width:              width,
			Height:             height,
			HasAlpha:           false,
			UniqueColorDensity: float64(len(uniqueBuckets)) / denom,
			EdgeDensity:        edgeAccum / denom,
			TextureScore:       textureAccum / denom,
		}
	}

	// Fallback to slow path
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

func checkAlphaNRGBA(img *image.NRGBA) bool {
	for y := 0; y < img.Rect.Dy(); y++ {
		offset := y * img.Stride
		for x := 0; x < img.Rect.Dx(); x++ {
			if img.Pix[offset+x*4+3] < 255 {
				return true
			}
		}
	}
	return false
}

func checkAlphaRGBA(img *image.RGBA) bool {
	for y := 0; y < img.Rect.Dy(); y++ {
		offset := y * img.Stride
		for x := 0; x < img.Rect.Dx(); x++ {
			if img.Pix[offset+x*4+3] < 255 {
				return true
			}
		}
	}
	return false
}

func absDiff8(a, b uint8) float64 {
	if a > b {
		return float64(a - b)
	}
	return float64(b - a)
}

