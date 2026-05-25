//go:build !((linux || darwin) && cgo)

package metric

import "github.com/zoulux/img2webp/internal/metric/gpu"

func initOpenCLBackend() gpu.Backend {
	return nil
}
