//go:build (linux || windows) && cgo

package cuda

import (
	"image"
	"image/color"
	"testing"
)

func TestNewCUDABackend(t *testing.T) {
	backend, err := NewCUDABackend()

	// On systems without CUDA, this is expected to fail
	if err != nil {
		t.Logf("CUDA not available: %v", err)
		return
	}

	if backend == nil {
		t.Fatal("backend is nil without error")
	}

	if backend.Name() != "cuda" {
		t.Errorf("expected name 'cuda', got %s", backend.Name())
	}
}

func TestCUDABackend_ComputeSSIM(t *testing.T) {
	backend, err := NewCUDABackend()
	if err != nil {
		t.Skip("CUDA not available")
	}

	// Create two identical images
	img1 := image.NewNRGBA(image.Rect(0, 0, 100, 100))
	img2 := image.NewNRGBA(image.Rect(0, 0, 100, 100))

	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			c := color.NRGBA{R: uint8(x), G: uint8(y), B: 128, A: 255}
			img1.Set(x, y, c)
			img2.Set(x, y, c)
		}
	}

	// Identical images should have score of 1.0
	score := backend.ComputeSSIM(img1, img2)
	if score != 1.0 {
		t.Errorf("expected SSIM 1.0 for identical images, got %f", score)
	}

	// Different images should have lower score
	img3 := image.NewNRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			c := color.NRGBA{R: uint8(255 - x), G: uint8(255 - y), B: 128, A: 255}
			img3.Set(x, y, c)
		}
	}

	score2 := backend.ComputeSSIM(img1, img3)
	if score2 >= 1.0 {
		t.Errorf("expected SSIM < 1.0 for different images, got %f", score2)
	}
}

func TestCUDABackend_ComputeEdgeScore(t *testing.T) {
	backend, err := NewCUDABackend()
	if err != nil {
		t.Skip("CUDA not available")
	}

	img1 := image.NewNRGBA(image.Rect(0, 0, 100, 100))
	img2 := image.NewNRGBA(image.Rect(0, 0, 100, 100))

	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			c := color.NRGBA{R: uint8(x), G: uint8(y), B: 128, A: 255}
			img1.Set(x, y, c)
			img2.Set(x, y, c)
		}
	}

	score := backend.ComputeEdgeScore(img1, img2)
	if score != 1.0 {
		t.Errorf("expected edge score 1.0 for identical images, got %f", score)
	}
}

func TestCUDABackend_SmallImage(t *testing.T) {
	backend, err := NewCUDABackend()
	if err != nil {
		t.Skip("CUDA not available")
	}

	// Small image should use CPU fallback
	img1 := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	img2 := image.NewNRGBA(image.Rect(0, 0, 10, 10))

	score := backend.ComputeSSIM(img1, img2)
	if score != 1.0 {
		t.Errorf("expected SSIM 1.0 for small identical images, got %f", score)
	}
}
