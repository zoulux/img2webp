package metric

import (
	"image"
	"image/color"
	"testing"
)

func TestGPUMetalSSIM(t *testing.T) {
	if !HasGPU() {
		t.Skip("No GPU available")
	}

	// Create a large test image (1920x1080 to trigger GPU)
	width, height := 1920, 1080
	img1 := image.NewNRGBA(image.Rect(0, 0, width, height))
	img2 := image.NewNRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img1.Set(x, y, color.NRGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
			img2.Set(x, y, color.NRGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}

	// GPU result
	gpuResult := SSIM(img1, img2)

	// CPU result
	cpuResult := ssimCPU(img1, img2)

	// Results should be very close
	if diff := abs(gpuResult - cpuResult); diff > 0.001 {
		t.Errorf("GPU SSIM = %v, CPU SSIM = %v, diff = %v", gpuResult, cpuResult, diff)
	}

	t.Logf("GPU backend: %s", GPUName())
	t.Logf("GPU SSIM: %.6f, CPU SSIM: %.6f", gpuResult, cpuResult)
}

func TestGPUMetalEdgeScore(t *testing.T) {
	if !HasGPU() {
		t.Skip("No GPU available")
	}

	width, height := 1920, 1080
	img1 := image.NewNRGBA(image.Rect(0, 0, width, height))
	img2 := image.NewNRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img1.Set(x, y, color.NRGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
			img2.Set(x, y, color.NRGBA{R: uint8((x + 10) % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}

	gpuResult := EdgeScore(img1, img2)
	cpuResult := edgeScoreCPU(img1, img2)

	// Allow small tolerance for floating point differences
	if diff := abs(gpuResult - cpuResult); diff > 0.01 {
		t.Errorf("GPU EdgeScore = %v, CPU EdgeScore = %v, diff = %v", gpuResult, cpuResult, diff)
	}

	t.Logf("GPU EdgeScore: %.6f, CPU EdgeScore: %.6f", gpuResult, cpuResult)
}

func BenchmarkSSIMCPUvsGPU(b *testing.B) {
	if !HasGPU() {
		b.Skip("No GPU available")
	}

	sizes := []struct {
		name   string
		width  int
		height int
	}{
		{"small_100x100", 100, 100},
		{"medium_500x500", 500, 500},
		{"large_1000x1000", 1000, 1000},
		{"4K_3840x2160", 3840, 2160},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			img1 := image.NewNRGBA(image.Rect(0, 0, size.width, size.height))
			img2 := image.NewNRGBA(image.Rect(0, 0, size.width, size.height))

			for y := 0; y < size.height; y++ {
				for x := 0; x < size.width; x++ {
					img1.Set(x, y, color.NRGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
					img2.Set(x, y, color.NRGBA{R: uint8((x + 10) % 256), G: uint8(y % 256), B: 128, A: 255})
				}
			}

			b.Run("CPU", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					ssimCPU(img1, img2)
				}
			})

			b.Run("GPU", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					SSIM(img1, img2)
				}
			})
		})
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
