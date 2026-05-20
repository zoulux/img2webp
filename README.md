# img2webp

A high-performance image-to-WebP converter with automatic quality optimization.

## Features

- **Automatic Quality Selection** — Analyzes image content and selects optimal encoding parameters
- **Multi-candidate Parallel Encoding** — Tests multiple quality candidates in parallel, picks the smallest passing quality thresholds
- **Content-Aware Processing** — Detects photos, graphics, and alpha channels for appropriate encoding strategy
- **Embedded Binary** — No external dependencies; `cwebp` is embedded for all major platforms
- **Cross-Platform** — Supports macOS (ARM64/x64), Linux (ARM64/x64), and Windows (x64)

## Installation

### Go Install

```bash
go install github.com/zoulux/img2webp@latest
```

### Download Binary

Download the latest release for your platform from the [Releases page](https://github.com/zoulux/img2webp/releases).

After downloading, grant execute permission:

```bash
# macOS / Linux
chmod +x img2webp-*

# Then run
./img2webp-darwin-arm64  # example for macOS Apple Silicon
```

> **Note for macOS users**: On first run, you may see a security warning. Go to `System Preferences > Privacy & Security` and click "Open Anyway", or run:
> ```bash
> xattr -d com.apple.quarantine img2webp-darwin-arm64
> ```

| Platform | Architecture | Binary |
|----------|-------------|--------|
| macOS | Apple Silicon (M1/M2/M3) | `img2webp-darwin-arm64` |
| macOS | Intel | `img2webp-darwin-amd64` |
| Linux | ARM64 | `img2webp-linux-arm64` |
| Linux | x86_64 | `img2webp-linux-amd64` |
| Windows | x86_64 | `img2webp-windows-amd64.exe` |

### Build from Source

```bash
git clone https://github.com/zoulux/img2webp.git
cd img2webp
go build -o img2webp .
```

## Usage

### Basic Usage

```bash
# Convert all images in current directory
img2webp

# Convert a specific file
img2webp photo.jpg

# Convert all images in a directory
img2webp /path/to/images
```

### Output Directory

```bash
# Specify output directory (default: "output")
img2webp -o /path/to/output photos/
img2webp --output ./webp-images .
```

### Quality Control

```bash
# Auto quality (default, recommended)
img2webp -q 0 images/

# Fixed quality
img2webp -q 85 photo.png
img2webp --quality 90 *.jpg
```

### Encoding Modes

```bash
# Auto-detect (default) — automatically determines best mode
img2webp --mode auto images/

# Photo mode — optimized for photographs
img2webp --mode photo photos/

# Graphic mode — optimized for screenshots, UI elements
img2webp --mode graphic screenshots/

# Lossless mode — for exact pixel reproduction
img2webp --mode lossless icons/
```

### Advanced Options

```bash
# Overwrite existing output files
img2webp --overwrite images/

# Parallel workers (default: CPU count)
img2webp --workers 8 large-batch/

# Re-encode existing WebP files
img2webp --reencode-webp images/

# Dry run — preview without writing files
img2webp --dry-run images/
```

## Command Reference

```
Usage: img2webp [flags] [input]

Arguments:
  input                     input file or directory (default ".")

Flags:
  -o, --output string       output directory (default "output")
  -q, --quality int         quality 1-100, 0 means auto (default 0)
      --overwrite           overwrite existing outputs
      --workers int         worker count, 0 means CPU count (default 0)
      --reencode-webp       re-encode input webp files
      --mode string         auto|photo|graphic|lossless (default "auto")
      --dry-run             print plan without writing files
  -h, --help                show help
```

## How It Works

1. **Input Collection** — Scans the input path for supported image formats (JPEG, PNG, WebP)

2. **Image Analysis** — Detects:
   - Image dimensions and aspect ratio
   - Alpha channel presence
   - Content type (photo vs. graphic)
   - Color complexity

3. **Candidate Generation** — Builds encoding candidates based on:
   - Content classification
   - Quality requirements
   - Alpha handling needs

4. **Parallel Encoding** — Encodes multiple quality candidates simultaneously using worker pools

5. **Quality Scoring** — Evaluates each candidate using:
   - SSIM (Structural Similarity Index)
   - Edge preservation metrics
   - Alpha edge quality (for transparent images)

6. **Selection** — Picks the smallest file that meets quality thresholds

7. **Output** — Writes the optimized WebP to the output directory, preserving the original directory structure

## Supported Formats

### Input
- JPEG (.jpg, .jpeg)
- PNG (.png)
- WebP (.webp)

### Output
- WebP (.webp)

## Performance

The tool uses parallel processing at multiple levels:
- **File-level parallelism** — Processes multiple images concurrently
- **Candidate-level parallelism** — Encodes multiple quality candidates in parallel per image

On a typical modern machine with 8+ cores, you can expect significant speedup for batch conversions.

## Dependencies

- **Embedded `cwebp` binary** — The Google libwebp encoder is embedded for all supported platforms. No external installation required.
- **Go runtime** — Only required if building from source or using `go install`

## License

MIT License
