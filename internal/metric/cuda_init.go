//go:build (linux || windows) && cgo

package metric

import (
	"github.com/zoulux/img2webp/internal/metric/gpu"
	"github.com/zoulux/img2webp/internal/metric/gpu/cuda"
)

func initCUDABackend() gpu.Backend {
	backend, err := cuda.NewCUDABackend()
	if err != nil {
		return nil
	}
	return backend
}
