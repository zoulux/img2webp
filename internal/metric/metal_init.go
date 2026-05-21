//go:build darwin

package metric

import (
	"github.com/zoulux/img2webp/internal/metric/gpu"
	"github.com/zoulux/img2webp/internal/metric/gpu/metal"
)

func initMetalBackend() gpu.Backend {
	backend, err := metal.NewMetalBackend()
	if err != nil {
		return nil
	}
	return backend
}
