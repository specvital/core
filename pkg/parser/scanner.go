package parser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/detection"
	"github.com/specvital/core/pkg/parser/framework"
	"github.com/specvital/core/pkg/parser/strategies/shared/dotnetast"
	"github.com/specvital/core/pkg/parser/strategies/shared/kotlinast"
	"github.com/specvital/core/pkg/parser/strategies/shared/swiftast"
	"github.com/specvital/core/pkg/source"
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
	// DefaultMaxFileSize is the default maximum file size for scanning (10MB).
	DefaultMaxFileSize = 10 * 1024 * 1024
)

// DefaultSkipPatterns contains directory names that are skipped by default during scanning.
var DefaultSkipPatterns = []string{
	"node_modules",
	".git",
	"vendor",
	"dist",
	".next",
	"__pycache__",
	"coverage",
	".cache",
}

var (
	// ErrScanCancelled is returned when scanning is cancelled via context.
	ErrScanCancelled = errors.New("scanner: scan cancelled")
	// ErrScanTimeout is returned when scanning exceeds the timeout duration.
	ErrScanTimeout = errors.New("scanner: scan timeout")
)

// Scanner performs framework detection and test file parsing.
// It integrates framework.Registry and detection.Detector for improved accuracy.
type Scanner struct {
	registry     *framework.Registry
	detector     *detection.Detector
	projectScope *framework.AggregatedProjectScope
	options      *ScanOptions
}

// ScanResult contains the outcome of a scan operation.
type ScanResult struct {
	// Inventory contains all parsed test files.
	Inventory *domain.Inventory

	// Errors contains non-fatal errors encountered during scanning.
	Errors []ScanError

	// Stats provides scan statistics including confidence distribution.
	Stats ScanStats
}

// ScanError represents an error that occurred during a specific phase of scanning.
type ScanError struct {
	// Err is the underlying error.
	Err error

	// Path is the file path where the error occurred (may be empty for non-file errors).
	Path string

	// Phase indicates which phase the error occurred in.
	// Values: "discovery", "config-parse", "detection", "parsing"
	Phase string
}

// Error implements the error interface.
func (e ScanError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("[%s] %v", e.Phase, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %v", e.Phase, e.Path, e.Err)
}

// ScanStats provides statistics about the scan operation.
type ScanStats struct {
	// FilesScanned is the total number of test file candidates discovered.
	FilesScanned int

	// FilesMatched is the number of files that were successfully parsed.
	FilesMatched int

	// FilesFailed is the number of files that failed to parse.
	FilesFailed int

	// FilesSkipped is the number of files skipped due to low confidence or other reasons.
	FilesSkipped int

	// ConfidenceDist tracks detection confidence distribution.
	// Keys: "definite", "moderate", "weak", "unknown"
	ConfidenceDist map[string]int

	// ConfigsFound is the number of config files discovered and parsed.
	ConfigsFound int

	// Duration is the total scan duration.
	Duration time.Duration
}

// NewScanner creates a new scanner with the given options.
func NewScanner(opts ...ScanOption) *Scanner {
	options := &ScanOptions{}
	for _, opt := range opts {
		opt(options)
	}
	applyDefaults(options)

	detector := detection.NewDetector(options.Registry)

	return &Scanner{
		registry:     options.Registry,
		detector:     detector,
		projectScope: nil,
		options:      options,
	}
}

// SetProjectScope sets pre-parsed config information for remote sources.
// This is useful when scanning from sources without filesystem access (e.g., GitHub API).
func (s *Scanner) SetProjectScope(scope *framework.AggregatedProjectScope) {
	s.projectScope = scope
	s.detector.SetProjectScope(scope)
}

// Scan performs the complete scanning process:
//  1. Discover and parse config files
//  2. Build project scope
//  3. Discover test files
//  4. Detect framework for each file
//  5. Parse test files in parallel
//
// The caller is responsible for calling src.Close() when done.
// For GitSource, failure to close will leak temporary directories.
func (s *Scanner) Scan(ctx context.Context, src source.Source) (*ScanResult, error) {
	startTime := time.Now()

	// Set up timeout
	ctx, cancel := context.WithTimeout(ctx, s.options.Timeout)
	defer cancel()

	rootPath := src.Root()

	result := &ScanResult{
		Inventory: &domain.Inventory{
			RootPath: rootPath,
			Files:    []domain.TestFile{},
		},
		Errors: []ScanError{},
		Stats: ScanStats{
			ConfidenceDist: make(map[string]int),
		},
	}

	if s.projectScope == nil {
		configFiles := s.discoverConfigFiles(ctx, src)
		s.projectScope = s.parseConfigFiles(ctx, src, configFiles, &result.Errors)
		s.detector.SetProjectScope(s.projectScope)
		result.Stats.ConfigsFound = len(s.projectScope.Configs)
	}

	testFiles, errs := s.discoverTestFiles(ctx, src)
	for _, err := range errs {
		result.Errors = append(result.Errors, ScanError{
			Err:   err,
			Phase: "discovery",
		})
	}
	result.Stats.FilesScanned = len(testFiles)

	if len(testFiles) == 0 {
		result.Stats.Duration = time.Since(startTime)
		return result, nil
	}

	files, scanErrors := s.parseFilesParallel(ctx, src, testFiles, result)
	result.Inventory.Files = files
	result.Errors = append(result.Errors, scanErrors...)

	result.Stats.FilesMatched = len(files)
	result.Stats.FilesFailed = len(scanErrors)
	result.Stats.FilesSkipped = result.Stats.FilesScanned - result.Stats.FilesMatched - result.Stats.FilesFailed
	result.Stats.Duration = time.Since(startTime)

	// Check for timeout or cancellation
	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return result, ErrScanTimeout
		}
		if errors.Is(err, context.Canceled) {
			return result, ErrScanCancelled
		}
	}

	return result, nil
}

// ScanFiles scans specific files (for incremental/watch mode).
// This bypasses file discovery and directly scans the provided file paths.
//
// The caller is responsible for calling src.Close() when done.
func (s *Scanner) ScanFiles(ctx context.Context, src source.Source, files []string) (*ScanResult, error) {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(ctx, s.options.Timeout)
	defer cancel()

	result := &ScanResult{
		Inventory: &domain.Inventory{
			RootPath: src.Root(),
			Files:    []domain.TestFile{},
		},
		Errors: []ScanError{},
		Stats: ScanStats{
			FilesScanned:   len(files),
			ConfidenceDist: make(map[string]int),
		},
	}

	if len(files) == 0 {
		result.Stats.Duration = time.Since(startTime)
		return result, nil
	}

	parsedFiles, scanErrors := s.parseFilesParallel(ctx, src, files, result)
	result.Inventory.Files = parsedFiles
	result.Errors = append(result.Errors, scanErrors...)

	result.Stats.FilesMatched = len(parsedFiles)
	result.Stats.FilesFailed = len(scanErrors)
	result.Stats.FilesSkipped = result.Stats.FilesScanned - result.Stats.FilesMatched - result.Stats.FilesFailed
	result.Stats.Duration = time.Since(startTime)

	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return result, ErrScanTimeout
		}
		if errors.Is(err, context.Canceled) {
			return result, ErrScanCancelled
		}
	}

	return result, nil
}

// discoverConfigFiles walks the source root to find framework config files.
// Returns relative paths from the source root for consistent Source.Open() usage.
func (s *Scanner) discoverConfigFiles(ctx context.Context, src source.Source) []string {
	patterns := []string{
		"jest.config.js",
		"jest.config.ts",
		"jest.config.mjs",
		"jest.config.cjs",
		"jest.config.json",
		"vitest.config.js",
		"vitest.config.ts",
		"vitest.config.mjs",
		"vitest.config.cjs",
		"playwright.config.js",
		"playwright.config.ts",
		"cypress.config.cjs",
		"cypress.config.js",
		"cypress.config.mjs",
		"cypress.config.mts",
		"cypress.config.ts",
		"pytest.ini",
		"pyproject.toml",
		"conftest.py",
		".rspec",
		"spec_helper.rb",
		"rails_helper.rb",
		"phpunit.xml",
		"phpunit.xml.dist",
		"phpunit.dist.xml",
		".mocharc.cjs",
		".mocharc.js",
		".mocharc.json",
		".mocharc.jsonc",
		".mocharc.mjs",
		".mocharc.yaml",
		".mocharc.yml",
		"mocha.opts",
	}

	rootPath := src.Root()
	skipSet := buildSkipSet(s.options.ExcludePatterns)
	var configFiles []string

	_ = filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if walkErr != nil {
			return nil
		}

		if d.IsDir() {
			if shouldSkipDir(path, rootPath, skipSet) {
				return filepath.SkipDir
			}
			return nil
		}

		filename := filepath.Base(path)
		for _, pattern := range patterns {
			if filename == pattern {
				relPath, err := filepath.Rel(rootPath, path)
				if err == nil {
					configFiles = append(configFiles, relPath)
				}
				break
			}
		}

		return nil
	})

	return configFiles
}

func (s *Scanner) parseConfigFiles(ctx context.Context, src source.Source, files []string, errors *[]ScanError) *framework.AggregatedProjectScope {
	scope := framework.NewProjectScope()

	for _, file := range files {
		if ctx.Err() != nil {
			break
		}

		content, err := readFileFromSource(ctx, src, file)
		if err != nil {
			*errors = append(*errors, ScanError{
				Err:   err,
				Path:  file,
				Phase: "config-parse",
			})
			continue
		}

		filename := filepath.Base(file)
		parsed := false

		for _, def := range s.registry.All() {
			if def.ConfigParser == nil {
				continue
			}

			signal := framework.Signal{
				Type:  framework.SignalConfigFile,
				Value: filename,
			}

			matched := false
			for _, matcher := range def.Matchers {
				result := matcher.Match(ctx, signal)
				if result.Confidence > 0 {
					matched = true
					break
				}
			}

			if matched {
				// Use absolute path for config parsing to ensure correct BaseDir resolution
				absConfigPath := filepath.Join(src.Root(), file)
				configScope, err := def.ConfigParser.Parse(ctx, absConfigPath, content)
				if err != nil {
					*errors = append(*errors, ScanError{
						Err:   err,
						Path:  file,
						Phase: "config-parse",
					})
				} else {
					scope.AddConfig(absConfigPath, configScope)
					parsed = true
				}
				break
			}
		}

		if !parsed {
			*errors = append(*errors, ScanError{
				Err:   fmt.Errorf("no matching framework config parser"),
				Path:  file,
				Phase: "config-parse",
			})
		}
	}

	return scope
}

// discoverTestFiles walks the source root to find test file candidates.
// Returns relative paths from the source root for consistent Source.Open() usage.
func (s *Scanner) discoverTestFiles(ctx context.Context, src source.Source) ([]string, []error) {
	rootPath := src.Root()
	skipSet := buildSkipSet(append(DefaultSkipPatterns, s.options.ExcludePatterns...))

	var (
		files []string
		errs  []error
		mu    sync.Mutex
	)

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if walkErr != nil {
			mu.Lock()
			errs = append(errs, fmt.Errorf("access error at %s: %w", path, walkErr))
			mu.Unlock()
			return nil
		}

		if d.IsDir() {
			if shouldSkipDir(path, rootPath, skipSet) {
				return filepath.SkipDir
			}
			return nil
		}

		if !isTestFileCandidate(path) {
			return nil
		}

		if len(s.options.Patterns) > 0 {
			if !matchesAnyPattern(path, rootPath, s.options.Patterns) {
				return nil
			}
		}

		if s.options.MaxFileSize > 0 {
			info, err := d.Info()
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("failed to get file info for %s: %w", path, err))
				mu.Unlock()
				return nil
			}
			if info.Size() > s.options.MaxFileSize {
				return nil
			}
		}

		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			mu.Lock()
			errs = append(errs, fmt.Errorf("compute relative path for %s: %w", path, err))
			mu.Unlock()
			return nil
		}

		mu.Lock()
		files = append(files, relPath)
		mu.Unlock()

		return nil
	})

	if err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			errs = append(errs, err)
		}
	}

	return files, errs
}

func (s *Scanner) parseFilesParallel(ctx context.Context, src source.Source, files []string, result *ScanResult) ([]domain.TestFile, []ScanError) {
	workers := s.options.Workers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	if workers > MaxWorkers {
		workers = MaxWorkers
	}

	sem := semaphore.NewWeighted(int64(workers))
	g, gCtx := errgroup.WithContext(ctx)

	var (
		mu         sync.Mutex
		testFiles  = make([]domain.TestFile, 0, len(files))
		scanErrors = make([]ScanError, 0)
	)

	for _, file := range files {
		file := file // Capture loop variable

		g.Go(func() error {
			if err := sem.Acquire(gCtx, 1); err != nil {
				return nil
			}
			defer sem.Release(1)

			testFile, scanErr, confidence := s.parseFile(gCtx, src, file)

			mu.Lock()
			defer mu.Unlock()

			if confidence != "" {
				result.Stats.ConfidenceDist[confidence]++
			}

			if scanErr != nil {
				scanErrors = append(scanErrors, *scanErr)
				return nil
			}

			if testFile != nil {
				testFiles = append(testFiles, *testFile)
			}

			return nil
		})
	}

	_ = g.Wait()

	// Sort by path for deterministic output order.
	// Parallel goroutines complete in variable order based on file size and parsing complexity.
	sort.Slice(testFiles, func(i, j int) bool {
		return testFiles[i].Path < testFiles[j].Path
	})

	return testFiles, scanErrors
}

func (s *Scanner) parseFile(ctx context.Context, src source.Source, path string) (*domain.TestFile, *ScanError, string) {
	if err := ctx.Err(); err != nil {
		return nil, &ScanError{
			Err:   err,
			Path:  path,
			Phase: "parsing",
		}, ""
	}

	content, err := readFileFromSource(ctx, src, path)
	if err != nil {
		return nil, &ScanError{
			Err:   err,
			Path:  path,
			Phase: "parsing",
		}, ""
	}

	// Use absolute path for detection to match config scope paths
	absPath := filepath.Join(src.Root(), path)
	detectionResult := s.detector.Detect(ctx, absPath, content)

	if !detectionResult.IsDetected() {
		return nil, nil, "unknown"
	}

	def := s.registry.Find(detectionResult.Framework)
	if def == nil || def.Parser == nil {
		return nil, &ScanError{
			Err:   fmt.Errorf("no parser for framework %s", detectionResult.Framework),
			Path:  path,
			Phase: "detection",
		}, string(detectionResult.Source)
	}

	testFile, err := def.Parser.Parse(ctx, content, path)
	if err != nil {
		return nil, &ScanError{
			Err:   fmt.Errorf("parse: %w", err),
			Path:  path,
			Phase: "parsing",
		}, string(detectionResult.Source)
	}

	return testFile, nil, string(detectionResult.Source)
}

// readFileFromSource reads a file from source using relative path.
// The relPath must be relative to src.Root().
func readFileFromSource(ctx context.Context, src source.Source, relPath string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	reader, err := src.Open(ctx, relPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", relPath, err)
	}

	return content, nil
}

func buildSkipSet(patterns []string) map[string]bool {
	skipSet := make(map[string]bool, len(patterns))
	for _, p := range patterns {
		skipSet[p] = true
	}
	return skipSet
}

func shouldSkipDir(path, rootPath string, skipSet map[string]bool) bool {
	if path == rootPath {
		return false
	}

	base := filepath.Base(path)
	return skipSet[base]
}

func isTestFileCandidate(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
		return isJSTestFile(path)
	case ".go":
		return isGoTestFile(path)
	case ".java":
		return isJavaTestFile(path)
	case ".kt", ".kts":
		return isKotlinTestFile(path)
	case ".py":
		return isPythonTestFile(path)
	case ".cs":
		return isCSharpTestFile(path)
	case ".rb":
		return isRubyTestFile(path)
	case ".rs":
		return isRustTestFile(path)
	case ".cc", ".cpp", ".cxx":
		return isCppTestFile(path)
	case ".php":
		return isPHPTestFile(path)
	case ".swift":
		return isSwiftTestFile(path)
	default:
		return false
	}
}

func isGoTestFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, "_test.go")
}

func isJavaTestFile(path string) bool {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, ".java")

	// JUnit conventions: *Test.java, *Tests.java, Test*.java
	if strings.HasSuffix(name, "Test") || strings.HasSuffix(name, "Tests") {
		return true
	}
	if strings.HasPrefix(name, "Test") {
		return true
	}

	// Files in test/ or tests/ directory
	normalizedPath := filepath.ToSlash(path)
	if strings.Contains(normalizedPath, "/test/") || strings.Contains(normalizedPath, "/tests/") {
		return true
	}
	if strings.Contains(normalizedPath, "/src/test/") {
		return true
	}

	return false
}

func isKotlinTestFile(path string) bool {
	return kotlinast.IsKotlinTestFile(path)
}

func isJSTestFile(path string) bool {
	base := filepath.Base(path)
	lowerBase := strings.ToLower(base)

	if strings.Contains(lowerBase, ".test.") || strings.Contains(lowerBase, ".spec.") {
		return true
	}

	// Playwright setup/teardown files: *.setup.{js,ts,jsx,tsx}
	ext := filepath.Ext(lowerBase)
	if ext == ".js" || ext == ".ts" || ext == ".jsx" || ext == ".tsx" {
		nameWithoutExt := strings.TrimSuffix(lowerBase, ext)
		if strings.HasSuffix(nameWithoutExt, ".setup") || strings.HasSuffix(nameWithoutExt, ".teardown") {
			return true
		}
	}

	// Cypress E2E test files: *.cy.{js,ts,jsx,tsx}
	if strings.Contains(lowerBase, ".cy.") {
		return true
	}

	normalizedPath := filepath.ToSlash(path)

	// Exclude fixture and mock directories (not actual test files)
	if strings.Contains(normalizedPath, "/__fixtures__/") || strings.HasPrefix(normalizedPath, "__fixtures__/") ||
		strings.Contains(normalizedPath, "/__mocks__/") || strings.HasPrefix(normalizedPath, "__mocks__/") {
		return false
	}

	if strings.Contains(normalizedPath, "/__tests__/") || strings.HasPrefix(normalizedPath, "__tests__/") {
		return true
	}

	// Cypress e2e/ and component/ directories
	if strings.Contains(normalizedPath, "/cypress/e2e/") || strings.Contains(normalizedPath, "/cypress/component/") {
		return true
	}

	return false
}

func isPythonTestFile(path string) bool {
	base := filepath.Base(path)

	// pytest conventions: test_*.py or *_test.py
	if strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py") {
		return true
	}
	if strings.HasSuffix(base, "_test.py") {
		return true
	}

	// conftest.py is a pytest configuration/fixture file, not a test file.
	// It's discovered as a config file but doesn't contain tests.
	if base == "conftest.py" {
		return false
	}

	// Files in tests/ directory
	normalizedPath := filepath.ToSlash(path)
	if strings.Contains(normalizedPath, "/tests/") || strings.HasPrefix(normalizedPath, "tests/") {
		return strings.HasSuffix(base, ".py")
	}

	return false
}

func isCSharpTestFile(path string) bool {
	// Check filename pattern first
	if dotnetast.IsCSharpTestFileName(path) {
		return true
	}

	// Check directory patterns for .NET project conventions
	normalizedPath := filepath.ToSlash(path)
	if strings.Contains(normalizedPath, "/test/") || strings.Contains(normalizedPath, "/tests/") {
		return true
	}
	if strings.Contains(normalizedPath, ".Tests/") || strings.Contains(normalizedPath, ".Test/") {
		return true
	}
	if strings.Contains(normalizedPath, ".Specs/") || strings.Contains(normalizedPath, ".Spec/") {
		return true
	}
	// "Tests/" as project folder pattern (both capitalized and lowercase)
	if strings.HasPrefix(normalizedPath, "Tests/") || strings.Contains(normalizedPath, "/Tests/") ||
		strings.HasPrefix(normalizedPath, "tests/") {
		return true
	}

	return false
}

func isRubyTestFile(path string) bool {
	base := filepath.Base(path)

	// Exclude config/helper files
	if base == "spec_helper.rb" || base == "rails_helper.rb" {
		return false
	}

	// RSpec convention: *_spec.rb
	if strings.HasSuffix(base, "_spec.rb") {
		return true
	}

	// Files in spec/ directory (excluding spec/support/ subdirectory)
	normalizedPath := filepath.ToSlash(path)
	if strings.Contains(normalizedPath, "/spec/") || strings.HasPrefix(normalizedPath, "spec/") {
		// Exclude spec/support/ directory (helpers, not tests)
		if strings.Contains(normalizedPath, "/spec/support/") || strings.HasPrefix(normalizedPath, "spec/support/") {
			return false
		}
		return strings.HasSuffix(base, ".rb")
	}

	return false
}

func isRustTestFile(path string) bool {
	base := filepath.Base(path)

	// Rust test file convention: *_test.rs
	if strings.HasSuffix(base, "_test.rs") {
		return true
	}

	normalizedPath := filepath.ToSlash(path)

	// tests/ directory: all .rs files are candidates (content matcher filters non-tests)
	if strings.Contains(normalizedPath, "/tests/") || strings.HasPrefix(normalizedPath, "tests/") {
		return strings.HasSuffix(base, ".rs")
	}

	// src/ directory: Rust places unit tests inline with #[cfg(test)] modules
	if strings.Contains(normalizedPath, "/src/") || strings.HasPrefix(normalizedPath, "src/") {
		return strings.HasSuffix(base, ".rs")
	}

	return false
}

func isCppTestFile(path string) bool {
	base := filepath.Base(path)
	baseLower := strings.ToLower(base)

	// Strip extension to check name patterns
	ext := filepath.Ext(baseLower)
	name := strings.TrimSuffix(baseLower, ext)

	// Google Test conventions: *_test, *_unittest
	if strings.HasSuffix(name, "_test") || strings.HasSuffix(name, "_unittest") {
		return true
	}

	// *Test pattern (e.g., DatabaseTest.cc) - uppercase T avoids false positives like "contest.cc"
	baseOriginal := filepath.Base(path)
	nameOriginal := strings.TrimSuffix(baseOriginal, filepath.Ext(baseOriginal))
	if strings.HasSuffix(nameOriginal, "Test") && len(nameOriginal) > 4 {
		return true
	}

	normalizedPath := filepath.ToSlash(path)

	// test/ or tests/ directory
	if strings.Contains(normalizedPath, "/test/") || strings.Contains(normalizedPath, "/tests/") {
		return true
	}
	if strings.HasPrefix(normalizedPath, "test/") || strings.HasPrefix(normalizedPath, "tests/") {
		return true
	}

	return false
}

func isPHPTestFile(path string) bool {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, ".php")

	if strings.HasSuffix(name, "Test") || strings.HasSuffix(name, "Tests") {
		return true
	}
	if strings.HasPrefix(name, "Test") {
		return true
	}

	normalizedPath := filepath.ToSlash(path)

	if strings.Contains(normalizedPath, "/test/") || strings.Contains(normalizedPath, "/tests/") {
		return true
	}
	if strings.HasPrefix(normalizedPath, "test/") || strings.HasPrefix(normalizedPath, "tests/") {
		return true
	}

	return false
}

func isSwiftTestFile(path string) bool {
	return swiftast.IsSwiftTestFile(path)
}

func matchesAnyPattern(path, rootPath string, patterns []string) bool {
	relPath, err := filepath.Rel(rootPath, path)
	if err != nil {
		return false
	}
	relPath = filepath.ToSlash(relPath)

	for _, pattern := range patterns {
		matched, err := doublestar.Match(pattern, relPath)
		if err != nil {
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

func Scan(ctx context.Context, src source.Source, opts ...ScanOption) (*ScanResult, error) {
	scanner := NewScanner(opts...)
	return scanner.Scan(ctx, src)
}
