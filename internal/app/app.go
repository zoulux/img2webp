package app

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	_ "golang.org/x/image/webp"
	decoderwebp "golang.org/x/image/webp"

	"github.com/zoulux/img2webp/internal/analyze"
	"github.com/zoulux/img2webp/internal/bin"
	"github.com/zoulux/img2webp/internal/cli"
	"github.com/zoulux/img2webp/internal/encode"
	"github.com/zoulux/img2webp/internal/fs"
	"github.com/zoulux/img2webp/internal/metric"
	"github.com/zoulux/img2webp/internal/report"
	"github.com/zoulux/img2webp/internal/strategy"
)

type Result struct {
	Summary report.Summary
	Results []report.Result
}

type Options struct {
	OnProgress func(report.Progress)
}

type App struct {
	onProgress func(report.Progress)
}

var releaseCWebP = bin.ReleaseCWebP

func New(opts *Options) *App {
	app := &App{}
	if opts != nil {
		app.onProgress = opts.OnProgress
	}
	return app
}

func (a *App) Run(ctx context.Context, cfg cli.Config) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	inputs, err := fs.CollectInputs(cfg.InputPath)
	if err != nil {
		return Result{}, err
	}

	rootInput, err := rootInputPath(cfg.InputPath)
	if err != nil {
		return Result{}, err
	}

	var binaryPath string
	if _, err := ensureBinaryPath(&binaryPath); err != nil {
		return Result{}, err
	}

	workers := cfg.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if workers > len(inputs) {
		workers = len(inputs)
	}

	type indexedResult struct {
		index  int
		result report.Result
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]indexedResult, 0, len(inputs))
	total := len(inputs)

	// Track progress per file for unified progress display
	fileProgress := make([]float64, total) // 0.0 to 1.0 per file

	for i, input := range inputs {
		wg.Add(1)
		go func(idx int, in string) {
			defer wg.Done()

			progressCb := func(stage report.ProgressStage, stageProgress float64) {
				if a.onProgress == nil {
					return
				}

				// Calculate this file's progress (each stage is ~1/3 of file progress)
				var filePct float64
				switch stage {
				case report.StageAnalyzing:
					filePct = stageProgress * 0.1 // Analyzing is 10% of file work
				case report.StageEncoding:
					filePct = 0.1 + stageProgress*0.8 // Encoding is 80% of file work
				case report.StageScoring:
					filePct = 0.9 + stageProgress*0.1 // Scoring is 10% of file work
				case report.StageDone:
					filePct = 1.0
				}

				mu.Lock()
				fileProgress[idx] = filePct
				// Sum all file progress for unified progress
				var totalProgress float64
				for _, p := range fileProgress {
					totalProgress += p
				}
				mu.Unlock()

				// Report unified progress
				a.onProgress(report.Progress{
					FileIndex:     int(totalProgress * 100),
					TotalFiles:    total * 100,
					Stage:         stage,
					StageProgress: stageProgress,
					InputPath:     in,
				})
			}

			fileResult := a.processFile(ctx, &binaryPath, rootInput, in, cfg, progressCb)

			mu.Lock()
			results = append(results, indexedResult{index: idx, result: fileResult})
			mu.Unlock()
		}(i, input)
	}

	wg.Wait()

	if err := ctx.Err(); err != nil {
		sort.Slice(results, func(i, j int) bool {
			return results[i].index < results[j].index
		})

		result := Result{Results: make([]report.Result, 0, len(results))}
		for _, r := range results {
			result.Results = append(result.Results, r.result)
			result.Summary.Add(r.result)
		}
		return result, err
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].index < results[j].index
	})

	result := Result{Results: make([]report.Result, 0, len(inputs))}
	for _, r := range results {
		result.Results = append(result.Results, r.result)
		result.Summary.Add(r.result)
	}

	return result, nil
}

func (a *App) processFile(ctx context.Context, binaryPath *string, rootInput, input string, cfg cli.Config, onProgress func(report.ProgressStage, float64)) report.Result {
	if onProgress != nil {
		onProgress(report.StageAnalyzing, 0.0)
	}

	sourceInfo, err := os.Stat(input)
	if err != nil {
		return report.Result{InputPath: input, Status: report.StatusFailed, Message: err.Error()}
	}

	if err := ctx.Err(); err != nil {
		return report.Result{InputPath: input, Status: report.StatusFailed, SourceBytes: sourceInfo.Size(), Message: err.Error()}
	}

	outputPath, err := fs.BuildOutputPath(rootInput, input, cfg.OutputDir)
	if err != nil {
		return report.Result{InputPath: input, Status: report.StatusFailed, SourceBytes: sourceInfo.Size(), Message: err.Error()}
	}

	result := report.Result{
		InputPath:   input,
		OutputPath:  outputPath,
		SourceBytes: sourceInfo.Size(),
	}

	if !cfg.Overwrite {
		if _, err := os.Stat(outputPath); err == nil {
			result.Status = report.StatusSkipped
			result.Message = "output already exists"
			if outInfo, statErr := os.Stat(outputPath); statErr == nil {
				result.OutputBytes = outInfo.Size()
			}
			return result
		} else if !os.IsNotExist(err) {
			result.Status = report.StatusFailed
			result.Message = err.Error()
			return result
		}
	}

	if cfg.DryRun {
		result.Status = report.StatusSkipped
		result.Message = "dry-run"
		return result
	}

	if strings.EqualFold(filepath.Ext(input), ".webp") && !cfg.ReencodeWebP {
		if err := fs.CopyFile(input, outputPath); err != nil {
			result.Status = report.StatusFailed
			result.Message = err.Error()
			return result
		}
		outInfo, err := os.Stat(outputPath)
		if err == nil {
			result.OutputBytes = outInfo.Size()
		}
		result.Status = report.StatusSkipped
		result.Message = "copied existing webp"
		return result
	}

	if err := ctx.Err(); err != nil {
		result.Status = report.StatusFailed
		result.Message = err.Error()
		return result
	}

	resolvedBinaryPath, err := ensureBinaryPath(binaryPath)
	if err != nil {
		result.Status = report.StatusFailed
		result.Message = err.Error()
		return result
	}

	if err := ctx.Err(); err != nil {
		result.Status = report.StatusFailed
		result.Message = err.Error()
		return result
	}

	features, err := analyze.AnalyzeFile(input)
	if err != nil {
		result.Status = report.StatusFailed
		result.Message = err.Error()
		return result
	}
	kind := analyze.Classify(features)
	candidates := strategy.BuildCandidates(kind, features, cfg.Quality, strategy.Mode(cfg.Mode))
	if onProgress != nil {
		onProgress(report.StageAnalyzing, 1.0)
	}

	if err := ctx.Err(); err != nil {
		result.Status = report.StatusFailed
		result.Message = err.Error()
		return result
	}

	if err := fs.EnsureParentDir(outputPath); err != nil {
		result.Status = report.StatusFailed
		result.Message = err.Error()
		return result
	}

	if err := ctx.Err(); err != nil {
		result.Status = report.StatusFailed
		result.Message = err.Error()
		return result
	}

	original, err := decodeGeneric(input)
	if err != nil {
		result.Status = report.StatusFailed
		result.Message = err.Error()
		return result
	}

	candidateResults := make([]encode.CandidateResult, 0, len(candidates))
	totalCandidates := len(candidates)

	// Create a temp directory in the system temp location
	tempDir, err := os.MkdirTemp("", "img2webp-*")
	if err != nil {
		result.Status = report.StatusFailed
		result.Message = fmt.Sprintf("failed to create temp directory: %s", err)
		return result
	}
	defer os.RemoveAll(tempDir)

	// Process candidates in parallel
	type candidateJob struct {
		index     int
		candidate strategy.Candidate
	}
	type candidateOutcome struct {
		index  int
		result encode.CandidateResult
		err    error
	}

	jobs := make(chan candidateJob, len(candidates))
	outcomes := make(chan candidateOutcome, len(candidates))

	// Start workers for parallel encoding (use min of candidates count and CPU count)
	numWorkers := min(len(candidates), runtime.NumCPU())
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				tmpOut := filepath.Join(tempDir, fmt.Sprintf("candidate-%d.webp", job.index))

				if err := encode.RunCWebP(ctx, resolvedBinaryPath, input, tmpOut, job.candidate); err != nil {
					outcomes <- candidateOutcome{index: job.index, err: err}
					continue
				}

				candidateImg, err := decodeWebP(tmpOut)
				if err != nil {
					outcomes <- candidateOutcome{index: job.index, err: err}
					continue
				}

				stats, err := os.Stat(tmpOut)
				if err != nil {
					outcomes <- candidateOutcome{index: job.index, err: err}
					continue
				}

				scores := encode.Scores{
					SSIM:      metric.SSIM(original, candidateImg),
					Edge:      metric.EdgeScore(original, candidateImg),
					AlphaEdge: metric.AlphaEdgeScore(original, candidateImg),
				}
				scores.Pass = encode.EvaluatePass(kind, scores)
				outcomes <- candidateOutcome{
					index: job.index,
					result: encode.CandidateResult{Path: tmpOut, Size: stats.Size(), Scores: scores},
				}
			}
		}()
	}

	// Send all jobs in a separate goroutine so workers can start immediately
	go func() {
		for i, candidate := range candidates {
			jobs <- candidateJob{index: i, candidate: candidate}
		}
		close(jobs)
	}()

	// Wait for workers to finish and close outcomes
	go func() {
		wg.Wait()
		close(outcomes)
	}()

	// Collect results and report progress
	for outcome := range outcomes {
		if err := ctx.Err(); err != nil {
			result.Status = report.StatusFailed
			result.Message = err.Error()
			return result
		}

		if outcome.err == nil {
			candidateResults = append(candidateResults, outcome.result)
		}

		// Report progress
		completed := len(candidateResults)
		stageProgress := float64(completed) / float64(totalCandidates)
		if onProgress != nil {
			onProgress(report.StageEncoding, stageProgress)
		}
	}

	if onProgress != nil {
		onProgress(report.StageScoring, 1.0)
	}

	picked, ok := encode.PickSmallestPassingWithSizeCheck(candidateResults, sourceInfo.Size())
	if !ok {
		result.Status = report.StatusFailed
		result.Message = "no candidate met quality thresholds"
		return result
	}

	// Copy the picked file to the output path (since temp dir may be on different filesystem)
	if err := fs.CopyFile(picked.Path, outputPath); err != nil {
		result.Status = report.StatusFailed
		result.Message = err.Error()
		return result
	}

	if outInfo, err := os.Stat(outputPath); err == nil {
		result.OutputBytes = outInfo.Size()
	}
	result.Status = report.StatusSuccess
	result.Message = string(kind)
	return result
}

func rootInputPath(inputPath string) (string, error) {
	info, err := os.Stat(inputPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return inputPath, nil
	}
	return filepath.Dir(inputPath), nil
}

func ensureBinaryPath(binaryPath *string) (string, error) {
	if *binaryPath != "" {
		return *binaryPath, nil
	}

	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	path, err := releaseCWebP(filepath.Join(cacheRoot, "img2webp", "bin"))
	if err != nil {
		return "", err
	}
	*binaryPath = path
	return path, nil
}

func decodeGeneric(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	return img, err
}

func decodeWebP(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := decoderwebp.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}
