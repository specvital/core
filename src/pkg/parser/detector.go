package parser

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

const (
	// DefaultMaxFileSize is the default maximum file size for detection (10MB).
	DefaultMaxFileSize = 10 * 1024 * 1024 // 10MB

	jsTestInfix      = ".test."
	jsSpecInfix      = ".spec."
	jsTestsDir       = "/__tests__/"
	jsTestsDirPrefix = "__tests__/"
)

// DefaultSkipPatterns contains directory names that are skipped by default during detection.
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

// ErrInvalidRootPath is returned when the root path does not exist or is not a directory.
var ErrInvalidRootPath = errors.New("detector: root path does not exist or is not accessible")

// DetectionResult contains detected test files and any errors encountered during traversal.
type DetectionResult struct {
	// Errors contains non-fatal errors encountered during directory traversal.
	Errors []error
	// Files contains paths to detected test files.
	Files []string
}

// DetectorOptions configures the behavior of [DetectTestFiles].
type DetectorOptions struct {
	// SkipPatterns specifies directory names to skip during detection.
	SkipPatterns []string
	// Patterns specifies glob patterns to filter test files.
	Patterns []string
	// MaxFileSize is the maximum file size in bytes to include.
	MaxFileSize int64
}

// DetectorOption is a functional option for configuring [DetectTestFiles].
type DetectorOption func(*DetectorOptions)

// WithSkipPatterns returns a [DetectorOption] that replaces the skip patterns.
func WithSkipPatterns(patterns []string) DetectorOption {
	return func(o *DetectorOptions) {
		o.SkipPatterns = patterns
	}
}

// WithPatterns returns a [DetectorOption] that filters files by glob patterns.
func WithPatterns(patterns []string) DetectorOption {
	return func(o *DetectorOptions) {
		o.Patterns = patterns
	}
}

// WithMaxFileSize returns a [DetectorOption] that sets the maximum file size.
func WithMaxFileSize(size int64) DetectorOption {
	return func(o *DetectorOptions) {
		o.MaxFileSize = size
	}
}

// DetectTestFiles walks the directory tree and detects test files.
// It identifies test files by their naming conventions:
//   - JavaScript/TypeScript: *.test.ts, *.spec.ts, __tests__/*.ts
//   - Go: *_test.go
//
// The function continues on errors, collecting them in [DetectionResult.Errors].
func DetectTestFiles(ctx context.Context, rootPath string, opts ...DetectorOption) (*DetectionResult, error) {
	options := &DetectorOptions{
		SkipPatterns: DefaultSkipPatterns,
		Patterns:     nil,
		MaxFileSize:  DefaultMaxFileSize,
	}

	for _, opt := range opts {
		opt(options)
	}

	rootInfo, err := os.Stat(rootPath)
	if err != nil {
		return nil, ErrInvalidRootPath
	}
	if !rootInfo.IsDir() {
		return nil, ErrInvalidRootPath
	}

	skipSet := buildSkipSet(options.SkipPatterns)

	result := &DetectionResult{
		Files:  []string{},
		Errors: []error{},
	}

	err = filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}

		if walkErr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("access error at %s: %w", path, walkErr))
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

		if len(options.Patterns) > 0 {
			if !matchesAnyPattern(path, rootPath, options.Patterns) {
				return nil
			}
		}

		if options.MaxFileSize > 0 {
			info, err := d.Info()
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to get file info for %s: %w", path, err))
				return nil
			}
			if info.Size() > options.MaxFileSize {
				return nil
			}
		}

		result.Files = append(result.Files, path)
		return nil
	})

	if err != nil {
		return result, err
	}

	return result, nil
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
	case ".ts", ".tsx", ".js", ".jsx":
		return isJSTestFile(path)
	case ".go":
		return isGoTestFile(path)
	default:
		return false
	}
}

func isGoTestFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, "_test.go")
}

func isJSTestFile(path string) bool {
	base := filepath.Base(path)
	lowerBase := strings.ToLower(base)

	// *.test.*, *.spec.*
	if strings.Contains(lowerBase, jsTestInfix) || strings.Contains(lowerBase, jsSpecInfix) {
		return true
	}

	// __tests__ directory
	normalizedPath := filepath.ToSlash(path)
	if strings.Contains(normalizedPath, jsTestsDir) || strings.HasPrefix(normalizedPath, jsTestsDirPrefix) {
		return true
	}

	return false
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
			// Invalid pattern syntax - skip this pattern
			continue
		}
		if matched {
			return true
		}
	}
	return false
}
