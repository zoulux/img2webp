//go:build !((linux || windows) && cgo)

package metric

import "github.com/zoulux/img2webp/internal/metric/gpu"

func initCUDABackend() gpu.Backend {
	return nil
}
