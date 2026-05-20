package metric

import "image"

func EdgeScore(a, b image.Image) float64 {
	if a.Bounds() != b.Bounds() {
		return 0
	}

	bounds := a.Bounds()
	var diff float64
	var total float64
	for y := bounds.Min.Y; y < bounds.Max.Y-1; y++ {
		for x := bounds.Min.X; x < bounds.Max.X-1; x++ {
			ad := edgeMagnitude(a, x, y)
			bd := edgeMagnitude(b, x, y)
			diff += abs64(ad - bd)
			total += 255 * 3
		}
	}
	if total == 0 {
		return 1
	}
	return 1 - diff/total
}

func edgeMagnitude(img image.Image, x, y int) float64 {
	r1, g1, b1, _ := img.At(x, y).RGBA()
	r2, g2, b2, _ := img.At(x+1, y).RGBA()
	return abs64(float64(int(r1>>8)-int(r2>>8))) + abs64(float64(int(g1>>8)-int(g2>>8))) + abs64(float64(int(b1>>8)-int(b2>>8)))
}
