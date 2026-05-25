//go:build !(((linux && amd64) || windows || darwin) && cgo)

package metric

import "github.com/zoulux/img2webp/internal/metric/gpu"

func initOpenCLBackend() gpu.Backend {
	return nil
}
