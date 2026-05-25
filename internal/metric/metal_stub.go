//go:build !(darwin && arm64)

package metric

import "github.com/zoulux/img2webp/internal/metric/gpu"

func initMetalBackend() gpu.Backend {
	return nil
}
