// Package gpu provides GPU-accelerated metric computation backends.
package gpu

import "image"

// Backend defines the interface for GPU-accelerated metric computation.
type Backend interface {
	// Available returns true if the GPU backend is ready for use.
	Available() bool

	// ComputeSSIM computes the SSIM-like score between two images.
	ComputeSSIM(a, b image.Image) float64

	// ComputeEdgeScore computes the edge similarity score between two images.
	ComputeEdgeScore(a, b image.Image) float64

	// Name returns the backend name (e.g., "metal", "opencl", "cuda").
	Name() string
}

// ShouldUseGPU determines if GPU acceleration would be beneficial for the given image size.
// Small images may be slower on GPU due to data transfer overhead.
// Note: On Apple Silicon, CPU parallel is often faster than GPU for this workload.
// Set a very high threshold to effectively disable GPU unless explicitly requested.
func ShouldUseGPU(bounds image.Rectangle) bool {
	pixels := bounds.Dx() * bounds.Dy()
	// Only use GPU for very large images (> 8K)
	// On Apple Silicon, CPU parallel is faster for SSIM/Edge calculations
	return pixels > 7680*4320
}
