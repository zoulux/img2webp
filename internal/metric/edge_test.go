package metric

import (
	"image"
	"image/color"
	"testing"
)

func TestEdgeScoreIdenticalImagesAreOne(t *testing.T) {
	img := steppedImage(8, 8)

	if got := EdgeScore(img, img); got != 1 {
		t.Fatalf("EdgeScore = %v, want 1", got)
	}
}

func TestEdgeScoreDifferentImagesRemainBounded(t *testing.T) {
	base := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	base.Set(0, 0, color.NRGBA{A: 255})
	base.Set(1, 0, color.NRGBA{A: 255})
	base.Set(0, 1, color.NRGBA{A: 255})
	base.Set(1, 1, color.NRGBA{A: 255})

	different := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	different.Set(0, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	different.Set(1, 0, color.NRGBA{A: 255})
	different.Set(0, 1, color.NRGBA{A: 255})
	different.Set(1, 1, color.NRGBA{A: 255})

	got := EdgeScore(base, different)
	if got < 0 || got > 1 {
		t.Fatalf("EdgeScore = %v, want within [0, 1]", got)
	}
	if got >= 1 {
		t.Fatalf("EdgeScore = %v, want < 1 for different images", got)
	}
}

func TestAlphaEdgeScoreIdenticalImagesAreOne(t *testing.T) {
	img := alphaSteppedImage(8, 8)

	if got := AlphaEdgeScore(img, img); got != 1 {
		t.Fatalf("AlphaEdgeScore = %v, want 1", got)
	}
}

func TestAlphaEdgeScoreDifferentImagesRemainBounded(t *testing.T) {
	base := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	base.Set(0, 0, color.NRGBA{A: 255})
	base.Set(1, 0, color.NRGBA{A: 255})
	base.Set(0, 1, color.NRGBA{A: 255})
	base.Set(1, 1, color.NRGBA{A: 255})

	different := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	different.Set(0, 0, color.NRGBA{A: 255})
	different.Set(1, 0, color.NRGBA{A: 0})
	different.Set(0, 1, color.NRGBA{A: 255})
	different.Set(1, 1, color.NRGBA{A: 255})

	got := AlphaEdgeScore(base, different)
	if got < 0 || got > 1 {
		t.Fatalf("AlphaEdgeScore = %v, want within [0, 1]", got)
	}
	if got >= 1 {
		t.Fatalf("AlphaEdgeScore = %v, want < 1 for different images", got)
	}
}

func steppedImage(width, height int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			v := uint8(0)
			if x >= width/2 {
				v = 255
			}
			img.Set(x, y, color.NRGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return img
}

func alphaSteppedImage(width, height int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			a := uint8(0)
			if x >= width/2 {
				a = 255
			}
			img.Set(x, y, color.NRGBA{A: a})
		}
	}
	return img
}
