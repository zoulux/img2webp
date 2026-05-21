package metric

import (
	"image"
	"runtime"
	"sync"
)

func edgeScoreCPU(a, b image.Image) float64 {
	if a.Bounds() != b.Bounds() {
		return 0
	}

	bounds := a.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Fast path for NRGBA images
	if imgA, ok := a.(*image.NRGBA); ok {
		if imgB, ok := b.(*image.NRGBA); ok {
			return edgeNRGBAFast(imgA, imgB, width, height)
		}
	}

	// Fast path for RGBA images
	if imgA, ok := a.(*image.RGBA); ok {
		if imgB, ok := b.(*image.RGBA); ok {
			return edgeRGBAFast(imgA, imgB, width, height)
		}
	}

	// Fast path for YCbCr (JPEG decoded images)
	if imgA, ok := a.(*image.YCbCr); ok {
		if imgB, ok := b.(*image.YCbCr); ok {
			return edgeYCbCrFast(imgA, imgB, width, height)
		}
	}

	// Fast path for NYCbCrA (WebP decoded images)
	if imgA, ok := a.(*image.NYCbCrA); ok {
		if imgB, ok := b.(*image.NYCbCrA); ok {
			return edgeNYCbCrAFast(imgA, imgB, width, height)
		}
	}

	// Fallback
	return edgeFallback(a, b, bounds)
}

func edgeNRGBAFast(a, b *image.NRGBA, width, height int) float64 {
	pixels := (width - 1) * (height - 1)

	if pixels <= 0 {
		return 1
	}

	// Parallel computation for large images
	if pixels > 250000 {
		return edgeNRGBAParallel(a, b, width, height)
	}

	var diff float64
	stride := a.Stride

	for y := 0; y < height-1; y++ {
		offset := y * stride

		for x := 0; x < width-1; x++ {
			i := offset + x*4

			// Horizontal edge for image A
			edgeA := abs64(float64(int(a.Pix[i]) - int(a.Pix[i+4])))
			edgeA += abs64(float64(int(a.Pix[i+1]) - int(a.Pix[i+5])))
			edgeA += abs64(float64(int(a.Pix[i+2]) - int(a.Pix[i+6])))

			// Horizontal edge for image B
			edgeB := abs64(float64(int(b.Pix[i]) - int(b.Pix[i+4])))
			edgeB += abs64(float64(int(b.Pix[i+1]) - int(b.Pix[i+5])))
			edgeB += abs64(float64(int(b.Pix[i+2]) - int(b.Pix[i+6])))

			diff += abs64(edgeA - edgeB)
		}
	}

	total := float64(pixels) * 255 * 3
	return 1 - diff/total
}

func edgeNRGBAParallel(a, b *image.NRGBA, width, height int) float64 {
	numWorkers := runtime.NumCPU()
	if numWorkers > height-1 {
		numWorkers = height - 1
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	rowsPerWorker := (height - 1 + numWorkers - 1) / numWorkers

	var totalDiff float64
	var mu sync.Mutex
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		startY := w * rowsPerWorker
		endY := startY + rowsPerWorker
		if endY > height-1 {
			endY = height - 1
		}
		if startY >= height-1 {
			break
		}

		wg.Add(1)
		go func(startY, endY int) {
			defer wg.Done()

			var localDiff float64
			stride := a.Stride

			for y := startY; y < endY; y++ {
				offset := y * stride

				for x := 0; x < width-1; x++ {
					i := offset + x*4

					// Horizontal edge for image A
					edgeA := abs64(float64(int(a.Pix[i]) - int(a.Pix[i+4])))
					edgeA += abs64(float64(int(a.Pix[i+1]) - int(a.Pix[i+5])))
					edgeA += abs64(float64(int(a.Pix[i+2]) - int(a.Pix[i+6])))

					// Horizontal edge for image B
					edgeB := abs64(float64(int(b.Pix[i]) - int(b.Pix[i+4])))
					edgeB += abs64(float64(int(b.Pix[i+1]) - int(b.Pix[i+5])))
					edgeB += abs64(float64(int(b.Pix[i+2]) - int(b.Pix[i+6])))

					localDiff += abs64(edgeA - edgeB)
				}
			}

			mu.Lock()
			totalDiff += localDiff
			mu.Unlock()
		}(startY, endY)
	}

	wg.Wait()

	pixels := (width - 1) * (height - 1)
	total := float64(pixels) * 255 * 3
	return 1 - totalDiff/total
}

func edgeRGBAFast(a, b *image.RGBA, width, height int) float64 {
	pixels := (width - 1) * (height - 1)
	if pixels <= 0 {
		return 1
	}

	if pixels > 250000 {
		return edgeRGBAParallel(a, b, width, height)
	}

	var diff float64
	stride := a.Stride

	for y := 0; y < height-1; y++ {
		offset := y * stride

		for x := 0; x < width-1; x++ {
			i := offset + x*4

			edgeA := abs64(float64(int(a.Pix[i]) - int(a.Pix[i+4])))
			edgeA += abs64(float64(int(a.Pix[i+1]) - int(a.Pix[i+5])))
			edgeA += abs64(float64(int(a.Pix[i+2]) - int(a.Pix[i+6])))

			edgeB := abs64(float64(int(b.Pix[i]) - int(b.Pix[i+4])))
			edgeB += abs64(float64(int(b.Pix[i+1]) - int(b.Pix[i+5])))
			edgeB += abs64(float64(int(b.Pix[i+2]) - int(b.Pix[i+6])))

			diff += abs64(edgeA - edgeB)
		}
	}

	total := float64(pixels) * 255 * 3
	return 1 - diff/total
}

func edgeRGBAParallel(a, b *image.RGBA, width, height int) float64 {
	numWorkers := runtime.NumCPU()
	if numWorkers > height-1 {
		numWorkers = height - 1
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	rowsPerWorker := (height - 1 + numWorkers - 1) / numWorkers

	var totalDiff float64
	var mu sync.Mutex
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		startY := w * rowsPerWorker
		endY := startY + rowsPerWorker
		if endY > height-1 {
			endY = height - 1
		}
		if startY >= height-1 {
			break
		}

		wg.Add(1)
		go func(startY, endY int) {
			defer wg.Done()

			var localDiff float64
			stride := a.Stride

			for y := startY; y < endY; y++ {
				offset := y * stride

				for x := 0; x < width-1; x++ {
					i := offset + x*4

					edgeA := abs64(float64(int(a.Pix[i]) - int(a.Pix[i+4])))
					edgeA += abs64(float64(int(a.Pix[i+1]) - int(a.Pix[i+5])))
					edgeA += abs64(float64(int(a.Pix[i+2]) - int(a.Pix[i+6])))

					edgeB := abs64(float64(int(b.Pix[i]) - int(b.Pix[i+4])))
					edgeB += abs64(float64(int(b.Pix[i+1]) - int(b.Pix[i+5])))
					edgeB += abs64(float64(int(b.Pix[i+2]) - int(b.Pix[i+6])))

					localDiff += abs64(edgeA - edgeB)
				}
			}

			mu.Lock()
			totalDiff += localDiff
			mu.Unlock()
		}(startY, endY)
	}

	wg.Wait()

	pixels := (width - 1) * (height - 1)
	total := float64(pixels) * 255 * 3
	return 1 - totalDiff/total
}

// edgeYCbCrFast computes edge score for YCbCr images (JPEG decoded)
func edgeYCbCrFast(a, b *image.YCbCr, width, height int) float64 {
	pixels := (width - 1) * (height - 1)
	if pixels <= 0 {
		return 1
	}

	var diff float64
	for y := 0; y < height-1; y++ {
		for x := 0; x < width-1; x++ {
			// Y channel edge
			ya1 := int(a.Y[y*a.YStride+x])
			ya2 := int(a.Y[y*a.YStride+x+1])
			edgeA := abs64(float64(ya1 - ya2))

			yb1 := int(b.Y[y*b.YStride+x])
			yb2 := int(b.Y[y*b.YStride+x+1])
			edgeB := abs64(float64(yb1 - yb2))

			diff += abs64(edgeA - edgeB)
		}
	}

	total := float64(pixels) * 255
	return 1 - diff/total
}

// edgeNYCbCrAFast computes edge score for NYCbCrA images (WebP decoded)
func edgeNYCbCrAFast(a, b *image.NYCbCrA, width, height int) float64 {
	pixels := (width - 1) * (height - 1)
	if pixels <= 0 {
		return 1
	}

	var diff float64
	for y := 0; y < height-1; y++ {
		for x := 0; x < width-1; x++ {
			// Y channel edge
			ya1 := int(a.Y[y*a.YStride+x])
			ya2 := int(a.Y[y*a.YStride+x+1])
			edgeA := abs64(float64(ya1 - ya2))

			yb1 := int(b.Y[y*b.YStride+x])
			yb2 := int(b.Y[y*b.YStride+x+1])
			edgeB := abs64(float64(yb1 - yb2))

			diff += abs64(edgeA - edgeB)
		}
	}

	total := float64(pixels) * 255
	return 1 - diff/total
}

func edgeFallback(a, b image.Image, bounds image.Rectangle) float64 {
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
