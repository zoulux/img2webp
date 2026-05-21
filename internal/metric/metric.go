package metric

import (
	"image"
	"runtime"

	"github.com/zoulux/img2webp/internal/metric/gpu"
)

var gpuBackend gpu.Backend

func init() {
	// Initialize GPU backend on macOS
	if runtime.GOOS == "darwin" {
		gpuBackend = initMetalBackend()
	}
}

// InitGPU initializes GPU acceleration. Returns true if GPU is available.
func InitGPU() bool {
	return gpuBackend != nil && gpuBackend.Available()
}

// SSIM computes the structural similarity between two images.
// Automatically uses GPU acceleration if available and beneficial.
func SSIM(a, b image.Image) float64 {
	if gpuBackend != nil && gpuBackend.Available() && gpu.ShouldUseGPU(a.Bounds()) {
		return gpuBackend.ComputeSSIM(a, b)
	}
	return ssimCPU(a, b)
}

// EdgeScore computes the edge similarity between two images.
// Automatically uses GPU acceleration if available and beneficial.
func EdgeScore(a, b image.Image) float64 {
	if gpuBackend != nil && gpuBackend.Available() && gpu.ShouldUseGPU(a.Bounds()) {
		return gpuBackend.ComputeEdgeScore(a, b)
	}
	return edgeScoreCPU(a, b)
}

// HasGPU returns true if a GPU backend is available.
func HasGPU() bool {
	return gpuBackend != nil && gpuBackend.Available()
}

// GPUName returns the name of the active GPU backend, or empty string if none.
func GPUName() string {
	if gpuBackend == nil {
		return ""
	}
	return gpuBackend.Name()
}
