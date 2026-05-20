package metric

import (
	"image"
	"image/color"
	"testing"
)

func TestSSIMIdenticalImagesAreOne(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	fill(img, color.NRGBA{R: 10, G: 20, B: 30, A: 255})

	if got := SSIM(img, img); got != 1 {
		t.Fatalf("SSIM = %v, want 1", got)
	}
}

func fill(img *image.NRGBA, c color.NRGBA) {
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}
