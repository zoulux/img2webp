package app

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "golang.org/x/image/webp"

	"github.com/zoulux/img2webp/internal/analyze"
	"github.com/zoulux/img2webp/internal/bin"
	"github.com/zoulux/img2webp/internal/encode"
	"github.com/zoulux/img2webp/internal/metric"
	"github.com/zoulux/img2webp/internal/strategy"
)

// BenchmarkImageProcessingStages measures time spent in each processing stage
func BenchmarkImageProcessingStages(b *testing.B) {
	// Create test images of different sizes
	sizes := []struct {
		name   string
		width  int
		height int
	}{
		{"720p_1280x720", 1280, 720},
		{"1080p_1920x1080", 1920, 1080},
		{"4K_3840x2160", 3840, 2160},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			// Create test image
			tmpDir := b.TempDir()
			inputPath := filepath.Join(tmpDir, "test.jpg")
			createTestImage(b, inputPath, size.width, size.height)

			// Get cwebp binary
			cacheDir, _ := os.UserCacheDir()
			binaryPath, err := bin.ReleaseCWebP(filepath.Join(cacheDir, "img2webp", "bin"))
			if err != nil {
				b.Skipf("cwebp not available: %v", err)
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Stage 1: Analyze (includes decode)
				t1 := time.Now()
				features, original, err := analyze.AnalyzeFile(inputPath)
				if err != nil {
					b.Fatal(err)
				}
				analyzeTime := time.Since(t1)

				// Stage 2: Build candidates
				t2 := time.Now()
				kind := analyze.Classify(features)
				candidates := strategy.BuildCandidates(kind, features, 85, strategy.ModeAuto)
				candidateTime := time.Since(t2)

				// Stage 3: Encode all candidates with cwebp (parallel)
				t3 := time.Now()
				var wg sync.WaitGroup
				var mu sync.Mutex
				outputs := make([]string, len(candidates))
				encodeErrors := make([]error, len(candidates))

				for i, c := range candidates {
					wg.Add(1)
					go func(idx int, candidate strategy.Candidate) {
						defer wg.Done()
						outputPath := filepath.Join(tmpDir, fmt.Sprintf("candidate-%d.webp", idx))
						err := encode.RunCWebP(context.Background(), binaryPath, inputPath, outputPath, candidate)
						mu.Lock()
						outputs[idx] = outputPath
						encodeErrors[idx] = err
						mu.Unlock()
					}(i, c)
				}
				wg.Wait()
				encodeTime := time.Since(t3)

				// Stage 5: Decode WebP (first candidate only for benchmark)
				t5 := time.Now()
				encoded := decodeWebPImage(b, outputs[0])
				webpDecodeTime := time.Since(t5)

				// Stage 6: SSIM calculation
				t6 := time.Now()
				ssimScore := metric.SSIM(original, encoded)
				_ = ssimScore
				ssimTime := time.Since(t6)

				// Stage 7: Edge calculation
				t7 := time.Now()
				edgeScore := metric.EdgeScore(original, encoded)
				_ = edgeScore
				edgeTime := time.Since(t7)

				// Report stage times (only on first iteration)
				if i == 0 {
					total := time.Since(t1)
					b.Logf("\n=== Stage Timing ===")
					b.Logf("Analyze (decode):  %v (%.1f%%)", analyzeTime, float64(analyzeTime)/float64(total)*100)
					b.Logf("Build candidates:  %v (%.1f%%)", candidateTime, float64(candidateTime)/float64(total)*100)
					b.Logf("cwebp encode:      %v (%.1f%%)", encodeTime, float64(encodeTime)/float64(total)*100)
					b.Logf("Decode WebP:       %v (%.1f%%)", webpDecodeTime, float64(webpDecodeTime)/float64(total)*100)
					b.Logf("SSIM calc:         %v (%.1f%%)", ssimTime, float64(ssimTime)/float64(total)*100)
					b.Logf("Edge calc:         %v (%.1f%%)", edgeTime, float64(edgeTime)/float64(total)*100)
					b.Logf("Total:             %v", total)
				}
			}
		})
	}
}

func createTestImage(b *testing.B, path string, width, height int) {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	// Create a gradient pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b_ := uint8(((x + y) * 255) / (width + height))
			img.Set(x, y, color.NRGBA{R: r, G: g, B: b_, A: 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()

	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		b.Fatal(err)
	}
}

func decodeImage(b *testing.B, path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		b.Fatal(err)
	}
	return img
}

func decodeWebPImage(b *testing.B, path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		b.Fatal(err)
	}
	return img
}
