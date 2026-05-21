package metric

import (
	"image"
	"runtime"
	"sync"
)

func ssimCPU(a, b image.Image) float64 {
	if a.Bounds() != b.Bounds() {
		return 0
	}

	bounds := a.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Fast path for NRGBA images (most common case)
	if imgA, ok := a.(*image.NRGBA); ok {
		if imgB, ok := b.(*image.NRGBA); ok {
			return ssimNRGBAFast(imgA, imgB, width, height)
		}
	}

	// Fast path for RGBA images
	if imgA, ok := a.(*image.RGBA); ok {
		if imgB, ok := b.(*image.RGBA); ok {
			return ssimRGBAFast(imgA, imgB, width, height)
		}
	}

	// Fast path for YCbCr (JPEG decoded images)
	if imgA, ok := a.(*image.YCbCr); ok {
		if imgB, ok := b.(*image.YCbCr); ok {
			return ssimYCbCrFast(imgA, imgB, width, height)
		}
	}

	// Fast path for NYCbCrA (WebP decoded images)
	if imgA, ok := a.(*image.NYCbCrA); ok {
		if imgB, ok := b.(*image.NYCbCrA); ok {
			return ssimNYCbCrAFast(imgA, imgB, width, height)
		}
	}

	// Fallback: convert to NRGBA first, then compute
	return ssimFallback(a, b, bounds)
}

// ssimNRGBAFast computes SSIM for NRGBA images using direct pixel access
func ssimNRGBAFast(a, b *image.NRGBA, width, height int) float64 {
	pixels := width * height

	// Parallel computation for large images
	if pixels > 250000 { // > 500x500
		return ssimNRGBAParallel(a, b, width, height)
	}

	var diff float64
	stride := a.Stride

	for y := 0; y < height; y++ {
		offset := y * stride
		for x := 0; x < width; x++ {
			i := offset + x*4
			diff += abs64(float64(int(a.Pix[i]) - int(b.Pix[i])))
			diff += abs64(float64(int(a.Pix[i+1]) - int(b.Pix[i+1])))
			diff += abs64(float64(int(a.Pix[i+2]) - int(b.Pix[i+2])))
		}
	}

	total := float64(pixels) * 255 * 3
	return 1 - diff/total
}

// ssimNRGBAParallel computes SSIM using multiple goroutines
func ssimNRGBAParallel(a, b *image.NRGBA, width, height int) float64 {
	numWorkers := runtime.NumCPU()
	if numWorkers > height {
		numWorkers = height
	}

	rowsPerWorker := (height + numWorkers - 1) / numWorkers

	var totalDiff float64
	var mu sync.Mutex
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		startY := w * rowsPerWorker
		endY := startY + rowsPerWorker
		if endY > height {
			endY = height
		}
		if startY >= height {
			break
		}

		wg.Add(1)
		go func(startY, endY int) {
			defer wg.Done()

			var localDiff float64
			stride := a.Stride

			for y := startY; y < endY; y++ {
				offset := y * stride
				for x := 0; x < width; x++ {
					i := offset + x*4
					localDiff += abs64(float64(int(a.Pix[i]) - int(b.Pix[i])))
					localDiff += abs64(float64(int(a.Pix[i+1]) - int(b.Pix[i+1])))
					localDiff += abs64(float64(int(a.Pix[i+2]) - int(b.Pix[i+2])))
				}
			}

			mu.Lock()
			totalDiff += localDiff
			mu.Unlock()
		}(startY, endY)
	}

	wg.Wait()

	pixels := width * height
	total := float64(pixels) * 255 * 3
	return 1 - totalDiff/total
}

// ssimRGBAFast computes SSIM for RGBA images using direct pixel access
func ssimRGBAFast(a, b *image.RGBA, width, height int) float64 {
	pixels := width * height

	// Parallel computation for large images
	if pixels > 250000 {
		return ssimRGBAParallel(a, b, width, height)
	}

	var diff float64
	stride := a.Stride

	for y := 0; y < height; y++ {
		offset := y * stride
		for x := 0; x < width; x++ {
			i := offset + x*4
			diff += abs64(float64(int(a.Pix[i]) - int(b.Pix[i])))
			diff += abs64(float64(int(a.Pix[i+1]) - int(b.Pix[i+1])))
			diff += abs64(float64(int(a.Pix[i+2]) - int(b.Pix[i+2])))
		}
	}

	total := float64(pixels) * 255 * 3
	return 1 - diff/total
}

// ssimRGBAParallel computes SSIM using multiple goroutines
func ssimRGBAParallel(a, b *image.RGBA, width, height int) float64 {
	numWorkers := runtime.NumCPU()
	if numWorkers > height {
		numWorkers = height
	}

	rowsPerWorker := (height + numWorkers - 1) / numWorkers

	var totalDiff float64
	var mu sync.Mutex
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		startY := w * rowsPerWorker
		endY := startY + rowsPerWorker
		if endY > height {
			endY = height
		}
		if startY >= height {
			break
		}

		wg.Add(1)
		go func(startY, endY int) {
			defer wg.Done()

			var localDiff float64
			stride := a.Stride

			for y := startY; y < endY; y++ {
				offset := y * stride
				for x := 0; x < width; x++ {
					i := offset + x*4
					localDiff += abs64(float64(int(a.Pix[i]) - int(b.Pix[i])))
					localDiff += abs64(float64(int(a.Pix[i+1]) - int(b.Pix[i+1])))
					localDiff += abs64(float64(int(a.Pix[i+2]) - int(b.Pix[i+2])))
				}
			}

			mu.Lock()
			totalDiff += localDiff
			mu.Unlock()
		}(startY, endY)
	}

	wg.Wait()

	pixels := width * height
	total := float64(pixels) * 255 * 3
	return 1 - totalDiff/total
}

// ssimYCbCrFast computes SSIM for YCbCr images (JPEG decoded)
func ssimYCbCrFast(a, b *image.YCbCr, width, height int) float64 {
	pixels := width * height

	if pixels > 250000 {
		return ssimYCbCrParallel(a, b, width, height)
	}

	var diff float64
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ya := int(a.Y[y*a.YStride+x])
			yb := int(b.Y[y*b.YStride+x])
			diff += abs64(float64(ya - yb))
		}
	}

	total := float64(pixels) * 255
	return 1 - diff/total
}

func ssimYCbCrParallel(a, b *image.YCbCr, width, height int) float64 {
	numWorkers := runtime.NumCPU()
	if numWorkers > height {
		numWorkers = height
	}

	rowsPerWorker := (height + numWorkers - 1) / numWorkers

	var totalDiff float64
	var mu sync.Mutex
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		startY := w * rowsPerWorker
		endY := startY + rowsPerWorker
		if endY > height {
			endY = height
		}
		if startY >= height {
			break
		}

		wg.Add(1)
		go func(startY, endY int) {
			defer wg.Done()

			var localDiff float64
			for y := startY; y < endY; y++ {
				for x := 0; x < width; x++ {
					ya := int(a.Y[y*a.YStride+x])
					yb := int(b.Y[y*b.YStride+x])
					localDiff += abs64(float64(ya - yb))
				}
			}

			mu.Lock()
			totalDiff += localDiff
			mu.Unlock()
		}(startY, endY)
	}

	wg.Wait()

	pixels := width * height
	total := float64(pixels) * 255
	return 1 - totalDiff/total
}

// ssimNYCbCrAFast computes SSIM for NYCbCrA images (WebP decoded)
func ssimNYCbCrAFast(a, b *image.NYCbCrA, width, height int) float64 {
	pixels := width * height

	if pixels > 250000 {
		return ssimNYCbCrAParallel(a, b, width, height)
	}

	var diff float64
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ya := int(a.Y[y*a.YStride+x])
			yb := int(b.Y[y*b.YStride+x])
			diff += abs64(float64(ya - yb))
		}
	}

	total := float64(pixels) * 255
	return 1 - diff/total
}

func ssimNYCbCrAParallel(a, b *image.NYCbCrA, width, height int) float64 {
	numWorkers := runtime.NumCPU()
	if numWorkers > height {
		numWorkers = height
	}

	rowsPerWorker := (height + numWorkers - 1) / numWorkers

	var totalDiff float64
	var mu sync.Mutex
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		startY := w * rowsPerWorker
		endY := startY + rowsPerWorker
		if endY > height {
			endY = height
		}
		if startY >= height {
			break
		}

		wg.Add(1)
		go func(startY, endY int) {
			defer wg.Done()

			var localDiff float64
			for y := startY; y < endY; y++ {
				for x := 0; x < width; x++ {
					ya := int(a.Y[y*a.YStride+x])
					yb := int(b.Y[y*b.YStride+x])
					localDiff += abs64(float64(ya - yb))
				}
			}

			mu.Lock()
			totalDiff += localDiff
			mu.Unlock()
		}(startY, endY)
	}

	wg.Wait()

	pixels := width * height
	total := float64(pixels) * 255
	return 1 - totalDiff/total
}

// ssimFallback converts images to NRGBA first, then computes
func ssimFallback(a, b image.Image, bounds image.Rectangle) float64 {
	width := bounds.Dx()
	height := bounds.Dy()

	// Convert to NRGBA
	imgA := image.NewNRGBA(bounds)
	imgB := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			imgA.Set(x, y, a.At(x, y))
			imgB.Set(x, y, b.At(x, y))
		}
	}

	return ssimNRGBAFast(imgA, imgB, width, height)
}

func abs64(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
