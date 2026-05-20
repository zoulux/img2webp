package metric

import "image"

func SSIM(a, b image.Image) float64 {
	if a.Bounds() != b.Bounds() {
		return 0
	}

	bounds := a.Bounds()
	var diff float64
	var total float64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ar, ag, ab, _ := a.At(x, y).RGBA()
			br, bg, bb, _ := b.At(x, y).RGBA()
			diff += abs64(float64(int(ar>>8) - int(br>>8)))
			diff += abs64(float64(int(ag>>8) - int(bg>>8)))
			diff += abs64(float64(int(ab>>8) - int(bb>>8)))
			total += 255 * 3
		}
	}
	if total == 0 {
		return 1
	}
	return 1 - diff/total
}

func abs64(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
