//go:build (linux || darwin) && cgo

package metric

import (
	"github.com/zoulux/img2webp/internal/metric/gpu"
	"github.com/zoulux/img2webp/internal/metric/gpu/opencl"
)

func initOpenCLBackend() gpu.Backend {
	backend, err := opencl.NewOpenCLBackend()
	if err != nil {
		return nil
	}
	return backend
}
