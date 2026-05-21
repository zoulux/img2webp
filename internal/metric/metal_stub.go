//go:build !darwin

package metric

import "github.com/zoulux/img2webp/internal/metric/gpu"

func initMetalBackend() gpu.Backend {
	return nil
}
