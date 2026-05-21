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
	"sync/atomic"

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

var cwebpSem = make(chan struct{}, runtime.NumCPU())

func acquireCwebpSlot(ctx context.Context) error {
	select {
	case cwebpSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseCwebpSlot() {
	<-cwebpSem
}

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

type fileJob struct {
	index     int
	inputPath string
}

type analyzedFile struct {
	index      int
	inputPath  string
	outputPath string
	features   analyze.Features
	kind       analyze.Kind
	mode       strategy.Mode
	candidates []strategy.Candidate
	sourceInfo os.FileInfo
	original   image.Image
	result     report.Result
	tempDir    string
}

type encodedFile struct {
	index            int
	inputPath        string
	outputPath       string
	candidates       []strategy.Candidate
	candidateResults []encode.CandidateResult
	sourceInfo       os.FileInfo
	result           report.Result
	tempDir          string
}

type indexedResult struct {
	index  int
	result report.Result
}

type errorRecorder struct {
	mu         sync.Mutex
	firstError error
}

func (e *errorRecorder) record(err error) {
	e.mu.Lock()
	if e.firstError == nil {
		e.firstError = err
	}
	e.mu.Unlock()
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
		workers = int(float64(runtime.NumCPU()) * 1.5)
		if workers < 1 {
			workers = 1
		}
	}

	total := len(inputs)

	jobsCh := make(chan fileJob, workers)
	go func() {
		for i, input := range inputs {
			jobsCh <- fileJob{index: i, inputPath: input}
		}
		close(jobsCh)
	}()

	analyzedCh := make(chan analyzedFile, workers)
	var wgAnalyze sync.WaitGroup
	for w := 0; w < workers; w++ {
		wgAnalyze.Add(1)
		go func() {
			defer wgAnalyze.Done()
			for job := range jobsCh {
				if err := ctx.Err(); err != nil {
					analyzedCh <- analyzedFile{
						index:     job.index,
						inputPath: job.inputPath,
						result:    report.Result{InputPath: job.inputPath, Status: report.StatusFailed, Message: err.Error()},
					}
					continue
				}
				file := a.analyzeFile(rootInput, job.inputPath, cfg)
				analyzedCh <- file
			}
		}()
	}
	go func() {
		wgAnalyze.Wait()
		close(analyzedCh)
	}()

	encodedCh := make(chan encodedFile, workers)
	var wgEncode sync.WaitGroup
	for w := 0; w < workers; w++ {
		wgEncode.Add(1)
		go func() {
			defer wgEncode.Done()
			for file := range analyzedCh {
				if file.result.Status == report.StatusFailed || file.result.Status == report.StatusSkipped {
					encodedCh <- encodedFile{
						index:      file.index,
						inputPath:  file.inputPath,
						sourceInfo: file.sourceInfo,
						result:     file.result,
					}
					continue
				}
				encoded := a.encodeFile(ctx, &binaryPath, file)
				encodedCh <- encoded
			}
		}()
	}
	go func() {
		wgEncode.Wait()
		close(encodedCh)
	}()

	var mu sync.Mutex
	results := make([]indexedResult, 0, total)
	var completedCount int64

	for encoded := range encodedCh {
		fileResult := encoded.result
		if fileResult.Status == report.StatusSuccess && len(encoded.candidateResults) > 0 {
			picked, ok := encode.PickSmallestPassingWithSizeCheck(encoded.candidateResults, encoded.sourceInfo.Size())
			if !ok {
				fileResult.Status = report.StatusFailed
				fileResult.Message = "no candidate met quality thresholds"
			} else if err := fs.CopyFile(picked.Path, fileResult.OutputPath); err != nil {
				fileResult.Status = report.StatusFailed
				fileResult.Message = err.Error()
			} else {
				if outInfo, err := os.Stat(fileResult.OutputPath); err == nil {
					fileResult.OutputBytes = outInfo.Size()
				}
				fileResult.Message = string(encoded.candidates[0].Kind)
			}
		}

		if encoded.tempDir != "" {
			os.RemoveAll(encoded.tempDir)
		}

		mu.Lock()
		results = append(results, indexedResult{index: encoded.index, result: fileResult})
		mu.Unlock()

		if a.onProgress != nil {
			completed := int(atomic.AddInt64(&completedCount, 1))
			a.onProgress(report.Progress{
				FileIndex:  completed * 100,
				TotalFiles: total * 100,
				Stage:      report.StageDone,
			})
		}
	}

	if err := ctx.Err(); err != nil {
		return buildResult(results), err
	}
	return buildResult(results), nil
}

func buildResult(results []indexedResult) Result {
	sort.Slice(results, func(i, j int) bool {
		return results[i].index < results[j].index
	})
	result := Result{Results: make([]report.Result, 0, len(results))}
	for _, r := range results {
		result.Results = append(result.Results, r.result)
		result.Summary.Add(r.result)
	}
	return result
}

func (a *App) analyzeFile(rootInput, input string, cfg cli.Config) analyzedFile {
	sourceInfo, err := os.Stat(input)
	if err != nil {
		return analyzedFile{
			inputPath:  input,
			sourceInfo: sourceInfo,
			result:     report.Result{InputPath: input, Status: report.StatusFailed, Message: err.Error()},
		}
	}

	outputPath, err := fs.BuildOutputPath(rootInput, input, cfg.OutputDir)
	if err != nil {
		return analyzedFile{
			inputPath:  input,
			sourceInfo: sourceInfo,
			result:     report.Result{InputPath: input, Status: report.StatusFailed, SourceBytes: sourceInfo.Size(), Message: err.Error()},
		}
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
			return analyzedFile{
				inputPath:  input,
				outputPath: outputPath,
				sourceInfo: sourceInfo,
				result:     result,
			}
		} else if !os.IsNotExist(err) {
			result.Status = report.StatusFailed
			result.Message = err.Error()
			return analyzedFile{
				inputPath:  input,
				outputPath: outputPath,
				sourceInfo: sourceInfo,
				result:     result,
			}
		}
	}

	if cfg.DryRun {
		result.Status = report.StatusSkipped
		result.Message = "dry-run"
		return analyzedFile{
			inputPath:  input,
			outputPath: outputPath,
			sourceInfo: sourceInfo,
			result:     result,
		}
	}

	if strings.EqualFold(filepath.Ext(input), ".webp") && !cfg.ReencodeWebP {
		if err := fs.CopyFile(input, outputPath); err != nil {
			result.Status = report.StatusFailed
			result.Message = err.Error()
			return analyzedFile{
				inputPath:  input,
				outputPath: outputPath,
				sourceInfo: sourceInfo,
				result:     result,
			}
		}
		outInfo, err := os.Stat(outputPath)
		if err == nil {
			result.OutputBytes = outInfo.Size()
		}
		result.Status = report.StatusSkipped
		result.Message = "copied existing webp"
		return analyzedFile{
			inputPath:  input,
			outputPath: outputPath,
			sourceInfo: sourceInfo,
			result:     result,
		}
	}

	features, original, err := analyze.AnalyzeFile(input)
	if err != nil {
		result.Status = report.StatusFailed
		result.Message = err.Error()
		return analyzedFile{
			inputPath:  input,
			outputPath: outputPath,
			sourceInfo: sourceInfo,
			result:     result,
		}
	}

	kind := analyze.Classify(features)
	mode := strategy.Mode(cfg.Mode)
	candidates := strategy.BuildCandidates(kind, features, cfg.Quality, mode)

	if err := fs.EnsureParentDir(outputPath); err != nil {
		result.Status = report.StatusFailed
		result.Message = err.Error()
		return analyzedFile{
			inputPath:  input,
			outputPath: outputPath,
			sourceInfo: sourceInfo,
			result:     result,
		}
	}

	tempDir, err := os.MkdirTemp("", "img2webp-*")
	if err != nil {
		result.Status = report.StatusFailed
		result.Message = fmt.Sprintf("failed to create temp directory: %s", err)
		return analyzedFile{
			inputPath:  input,
			outputPath: outputPath,
			sourceInfo: sourceInfo,
			result:     result,
		}
	}

	return analyzedFile{
		inputPath:  input,
		outputPath: outputPath,
		features:   features,
		kind:       kind,
		mode:       mode,
		candidates: candidates,
		sourceInfo: sourceInfo,
		original:   original,
		result:     report.Result{InputPath: input, OutputPath: outputPath, SourceBytes: sourceInfo.Size(), Status: report.StatusSuccess},
		tempDir:    tempDir,
	}
}

func (a *App) encodeFile(ctx context.Context, binaryPath *string, file analyzedFile) encodedFile {
	resolvedBinaryPath, err := ensureBinaryPath(binaryPath)
	if err != nil {
		return encodedFile{
			index:      file.index,
			inputPath:  file.inputPath,
			outputPath: file.outputPath,
			sourceInfo: file.sourceInfo,
			result:     report.Result{InputPath: file.inputPath, Status: report.StatusFailed, Message: err.Error()},
			tempDir:    file.tempDir,
		}
	}

	// Build staged candidates
	groups := strategy.BuildCandidateGroups(file.kind, file.features, 0, file.mode)

	candidateResults := make([]encode.CandidateResult, 0, 5)
	var mu sync.Mutex
	var errRec errorRecorder

	// Stage 1: Test first candidate (middle quality)
	firstResult := a.encodeCandidate(ctx, resolvedBinaryPath, file, groups.First, 0, &mu, &candidateResults, &errRec)
	if errRec.firstError != nil && len(candidateResults) == 0 {
		return encodedFile{
			index:      file.index,
			inputPath:  file.inputPath,
			outputPath: file.outputPath,
			sourceInfo: file.sourceInfo,
			result:     report.Result{InputPath: file.inputPath, Status: report.StatusFailed, Message: errRec.firstError.Error(), SourceBytes: file.sourceInfo.Size()},
			tempDir:    file.tempDir,
		}
	}

	// Stage 2: Based on first result, decide next candidates (parallel)
	if firstResult != nil && firstResult.Scores.Pass {
		// First passed - test lower quality candidates in parallel
		a.encodeCandidatesParallel(ctx, resolvedBinaryPath, file, groups.IfPass, 1, &mu, &candidateResults, &errRec)
	} else {
		// First failed - test higher quality + special candidates in parallel
		allCandidates := append(groups.IfFail, groups.Special...)
		a.encodeCandidatesParallel(ctx, resolvedBinaryPath, file, allCandidates, 1, &mu, &candidateResults, &errRec)
	}

	if len(candidateResults) == 0 && errRec.firstError != nil {
		return encodedFile{
			index:      file.index,
			inputPath:  file.inputPath,
			outputPath: file.outputPath,
			sourceInfo: file.sourceInfo,
			result:     report.Result{InputPath: file.inputPath, Status: report.StatusFailed, Message: errRec.firstError.Error(), SourceBytes: file.sourceInfo.Size()},
			tempDir:    file.tempDir,
		}
	}

	allCandidates := []strategy.Candidate{groups.First}
	allCandidates = append(allCandidates, groups.IfPass...)
	allCandidates = append(allCandidates, groups.IfFail...)
	allCandidates = append(allCandidates, groups.Special...)

	return encodedFile{
		index:            file.index,
		inputPath:        file.inputPath,
		outputPath:       file.outputPath,
		candidates:       allCandidates,
		candidateResults: candidateResults,
		sourceInfo:       file.sourceInfo,
		result:           file.result,
		tempDir:          file.tempDir,
	}
}

func (a *App) encodeCandidate(ctx context.Context, binaryPath string, file analyzedFile, c strategy.Candidate, idx int, mu *sync.Mutex, results *[]encode.CandidateResult, errRec *errorRecorder) *encode.CandidateResult {
	if ctx.Err() != nil {
		errRec.record(ctx.Err())
		return nil
	}

	tmpOut := filepath.Join(file.tempDir, fmt.Sprintf("candidate-%d.webp", idx))

	if err := acquireCwebpSlot(ctx); err != nil {
		errRec.record(err)
		return nil
	}
	defer releaseCwebpSlot()

	if err := encode.RunCWebP(ctx, binaryPath, file.inputPath, tmpOut, c); err != nil {
		errRec.record(err)
		return nil
	}

	candidateImg, err := decodeWebP(tmpOut)
	if err != nil {
		errRec.record(err)
		return nil
	}

	stats, err := os.Stat(tmpOut)
	if err != nil {
		errRec.record(err)
		return nil
	}

	scores := encode.Scores{
		SSIM:      metric.SSIM(file.original, candidateImg),
		Edge:      metric.EdgeScore(file.original, candidateImg),
		AlphaEdge: metric.AlphaEdgeScore(file.original, candidateImg),
	}
	scores.Pass = encode.EvaluatePass(file.kind, scores)

	result := encode.CandidateResult{
		Path:   tmpOut,
		Size:   stats.Size(),
		Scores: scores,
	}

	mu.Lock()
	*results = append(*results, result)
	mu.Unlock()

	return &result
}

func (a *App) encodeCandidatesParallel(ctx context.Context, binaryPath string, file analyzedFile, candidates []strategy.Candidate, startIdx int, mu *sync.Mutex, results *[]encode.CandidateResult, errRec *errorRecorder) {
	var wg sync.WaitGroup
	for i, c := range candidates {
		wg.Add(1)
		go func(idx int, candidate strategy.Candidate) {
			defer wg.Done()
			a.encodeCandidate(ctx, binaryPath, file, candidate, startIdx+idx, mu, results, errRec)
		}(i, c)
	}
	wg.Wait()
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
