package analyze

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyTransparentGraphic(t *testing.T) {
	f := Features{Width: 128, Height: 128, HasAlpha: true, EdgeDensity: 0.40, UniqueColorDensity: 0.10}
	if got := Classify(f); got != KindTransparentGraphic {
		t.Fatalf("Classify() = %q, want %q", got, KindTransparentGraphic)
	}
}

func TestClassifyGraphic(t *testing.T) {
	f := Features{Width: 1920, Height: 1080, HasAlpha: false, EdgeDensity: 0.32, UniqueColorDensity: 0.08, TextureScore: 0.10}
	if got := Classify(f); got != KindGraphic {
		t.Fatalf("Classify() = %q, want %q", got, KindGraphic)
	}
}

func TestClassifyPhoto(t *testing.T) {
	f := Features{Width: 4032, Height: 3024, HasAlpha: false, EdgeDensity: 0.18, UniqueColorDensity: 0.70, TextureScore: 0.66}
	if got := Classify(f); got != KindPhoto {
		t.Fatalf("Classify() = %q, want %q", got, KindPhoto)
	}
}

func TestClassifyUsesStrictThresholdBoundaries(t *testing.T) {
	tests := []struct {
		name string
		f    Features
		want Kind
	}{
		{
			name: "below both thresholds is graphic",
			f:    Features{UniqueColorDensity: 0.19, TextureScore: 0.24},
			want: KindGraphic,
		},
		{
			name: "unique color density at threshold is photo",
			f:    Features{UniqueColorDensity: 0.20, TextureScore: 0.24},
			want: KindPhoto,
		},
		{
			name: "texture score at threshold is photo",
			f:    Features{UniqueColorDensity: 0.19, TextureScore: 0.25},
			want: KindPhoto,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Classify(tt.f); got != tt.want {
				t.Fatalf("Classify() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAnalyzeImageDetectsAlphaOutsideSampleGrid(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 130, 130))
	for y := 0; y < 130; y++ {
		for x := 0; x < 130; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	img.SetNRGBA(1, 1, color.NRGBA{R: 255, G: 0, B: 0, A: 0})

	got := AnalyzeImage(img)

	if !got.HasAlpha {
		t.Fatal("HasAlpha = false, want true")
	}
	if kind := Classify(got); kind != KindTransparentGraphic {
		t.Fatalf("Classify(AnalyzeImage()) = %q, want %q", kind, KindTransparentGraphic)
	}
}

func TestAnalyzeImageOpaqueImageHasNoAlpha(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 130, 130))
	for y := 0; y < 130; y++ {
		for x := 0; x < 130; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 12, G: 34, B: 56, A: 255})
		}
	}

	got := AnalyzeImage(img)

	if got.HasAlpha {
		t.Fatal("HasAlpha = true, want false")
	}
}

func TestAnalyzeFileDecodesPNGAndMatchesAnalyzeImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(20 + x*30), G: uint8(40 + y*40), B: 90, A: 255})
		}
	}
	img.SetNRGBA(2, 1, color.NRGBA{R: 250, G: 250, B: 250, A: 128})

	path := filepath.Join(t.TempDir(), "sample.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := png.Encode(file, img); err != nil {
		file.Close()
		t.Fatalf("png.Encode returned error: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	got, err := AnalyzeFile(path)
	if err != nil {
		t.Fatalf("AnalyzeFile returned error: %v", err)
	}

	want := AnalyzeImage(img)
	if got != want {
		t.Fatalf("AnalyzeFile() = %+v, want %+v", got, want)
	}
}
