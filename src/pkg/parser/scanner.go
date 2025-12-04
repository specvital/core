// Package parser provides test file scanning and parsing capabilities.
// It supports multiple test frameworks (Jest, Vitest, Playwright, Go testing)
// and enables parallel processing for efficient scanning of large codebases.
package parser

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/specvital/core/src/pkg/domain"
	"github.com/specvital/core/src/pkg/parser/strategies"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

const (
	// DefaultWorkers indicates that the scanner should use GOMAXPROCS as the worker count.
	DefaultWorkers = 0
	// DefaultTimeout is the default scan timeout duration.
	DefaultTimeout = 5 * time.Minute
	// MaxWorkers is the maximum number of concurrent workers allowed.
	MaxWorkers = 1024
)

var (
	// ErrScanCancelled is returned when scanning is cancelled via context.
	ErrScanCancelled = errors.New("scanner: scan cancelled")
	// ErrScanTimeout is returned when scanning exceeds the timeout duration.
	ErrScanTimeout = errors.New("scanner: scan timeout")
)

// ScanResult contains the outcome of a scan operation.
type ScanResult struct {
	// Errors contains non-fatal errors encountered during scanning.
	Errors []ScanError
	// Inventory contains all parsed test files.
	Inventory *domain.Inventory
}

// ScanError represents an error that occurred during a specific phase of scanning.
type ScanError struct {
	// Err is the underlying error.
	Err error
	// Path is the file path where the error occurred (may be empty for non-file errors).
	Path string
	// Phase indicates which phase the error occurred in ("detection" or "parsing").
	Phase string
}

// Error implements the error interface.
func (e ScanError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("[%s] %v", e.Phase, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %v", e.Phase, e.Path, e.Err)
}

// Scan scans the given directory for test files and parses them.
// It uses parallel processing with configurable worker count and timeout.
// The function continues even when individual files fail to parse,
// collecting errors in [ScanResult.Errors].
func Scan(ctx context.Context, rootPath string, opts ...ScanOption) (*ScanResult, error) {
	options := &ScanOptions{
		ExcludePatterns: nil,
		MaxFileSize:     DefaultMaxFileSize,
		Patterns:        nil,
		Timeout:         DefaultTimeout,
		Workers:         DefaultWorkers,
	}

	for _, opt := range opts {
		opt(options)
	}

	workers := options.Workers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	if workers > MaxWorkers {
		workers = MaxWorkers
	}

	timeout := options.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	detectorOpts := buildDetectorOpts(options)
	detectionResult, err := DetectTestFiles(ctx, rootPath, detectorOpts...)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, ErrScanTimeout
		}
		if errors.Is(err, context.Canceled) {
			return nil, ErrScanCancelled
		}
		return nil, fmt.Errorf("scanner: detection failed: %w", err)
	}

	scanResult := &ScanResult{
		Errors: make([]ScanError, 0),
		Inventory: &domain.Inventory{
			Files:    make([]domain.TestFile, 0),
			RootPath: rootPath,
		},
	}

	for _, detErr := range detectionResult.Errors {
		scanResult.Errors = append(scanResult.Errors, ScanError{
			Err:   detErr,
			Path:  "",
			Phase: "detection",
		})
	}

	if len(detectionResult.Files) == 0 {
		return scanResult, nil
	}

	files, errs := parseFilesParallel(ctx, detectionResult.Files, workers)

	scanResult.Inventory.Files = files
	scanResult.Errors = append(scanResult.Errors, errs...)

	return scanResult, nil
}

func buildDetectorOpts(options *ScanOptions) []DetectorOption {
	var detectorOpts []DetectorOption

	if len(options.ExcludePatterns) > 0 {
		merged := make([]string, 0, len(DefaultSkipPatterns)+len(options.ExcludePatterns))
		merged = append(merged, DefaultSkipPatterns...)
		merged = append(merged, options.ExcludePatterns...)
		detectorOpts = append(detectorOpts, WithSkipPatterns(merged))
	}

	if len(options.Patterns) > 0 {
		detectorOpts = append(detectorOpts, WithPatterns(options.Patterns))
	}

	if options.MaxFileSize > 0 {
		detectorOpts = append(detectorOpts, WithMaxFileSize(options.MaxFileSize))
	}

	return detectorOpts
}

func parseFilesParallel(ctx context.Context, files []string, workers int) ([]domain.TestFile, []ScanError) {
	sem := semaphore.NewWeighted(int64(workers))
	g, gCtx := errgroup.WithContext(ctx)

	var (
		mu         sync.Mutex
		results    = make([]domain.TestFile, 0, len(files))
		scanErrors = make([]ScanError, 0)
	)

	for _, file := range files {
		g.Go(func() error {
			if err := sem.Acquire(gCtx, 1); err != nil {
				return nil // Context cancelled
			}
			defer sem.Release(1)

			testFile, err := parseFile(gCtx, file)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				scanErrors = append(scanErrors, ScanError{
					Err:   err,
					Path:  file,
					Phase: "parsing",
				})
				return nil // Continue with other files
			}

			if testFile != nil {
				results = append(results, *testFile)
			}

			return nil
		})
	}

	_ = g.Wait() // Errors are collected in scanErrors

	return results, scanErrors
}

func parseFile(ctx context.Context, path string) (*domain.TestFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}

	strategy := strategies.FindStrategy(path, content)
	if strategy == nil {
		return nil, nil // No matching strategy
	}

	testFile, err := strategy.Parse(ctx, content, path)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return testFile, nil
}
