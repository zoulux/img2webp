package encode

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/zoulux/img2webp/internal/strategy"
)

func buildArgs(inputPath, outputPath string, c strategy.Candidate) []string {
	args := []string{"-mt", "-m", strconv.Itoa(c.Method), "-q", strconv.Itoa(c.Quality), "-alpha_q", strconv.Itoa(c.AlphaQuality)}
	if c.NearLossless > 0 {
		args = append(args, "-near_lossless", strconv.Itoa(c.NearLossless))
	}
	if c.Lossless {
		args = append(args, "-lossless")
	}
	args = append(args, inputPath, "-o", outputPath)
	return args
}

func RunCWebP(ctx context.Context, binaryPath, inputPath, outputPath string, c strategy.Candidate) error {
	cmd := exec.CommandContext(ctx, binaryPath, buildArgs(inputPath, outputPath, c)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, msg)
	}
	return nil
}
