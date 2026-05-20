package metric

import "image"

func AlphaEdgeScore(a, b image.Image) float64 {
	if a.Bounds() != b.Bounds() {
		return 0
	}

	bounds := a.Bounds()
	var diff float64
	var total float64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, aa := a.At(x, y).RGBA()
			_, _, _, ba := b.At(x, y).RGBA()
			diff += abs64(float64(int(aa>>8) - int(ba>>8)))
			total += 255
		}
	}
	if total == 0 {
		return 1
	}
	return 1 - diff/total
}
