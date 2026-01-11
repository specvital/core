package parser

import (
	"time"

	"github.com/specvital/core/pkg/parser/framework"
)

// ScanOptions configures scanner behavior.
type ScanOptions struct {
	// ExcludePatterns specifies directory names to skip during file discovery.
	// These are combined with DefaultSkipPatterns.
	ExcludePatterns []string

	// ExtractDomainHints enables extraction of domain classification metadata.
	// When true, imports, function calls, and variable names are extracted.
	// Default: true (opt-out via WithDomainHints(false)).
	ExtractDomainHints bool

	// MaxFileSize is the maximum file size in bytes to process.
	// Files larger than this are skipped.
	MaxFileSize int64

	// Patterns specifies glob patterns to filter test files.
	// Empty means all test file candidates are processed.
	Patterns []string

	// Registry is the framework registry to use for detection.
	// If nil, uses framework.DefaultRegistry().
	Registry *framework.Registry

	// Timeout is the maximum duration for the entire scan operation.
	// Zero or negative values use DefaultTimeout.
	Timeout time.Duration

	// Workers specifies the number of concurrent file parsers.
	// Zero or negative values use runtime.GOMAXPROCS(0).
	Workers int
}

// ScanOption is a functional option for configuring Scanner.
type ScanOption func(*ScanOptions)

// WithWorkers sets the number of concurrent file parsers.
// Negative values are ignored.
func WithWorkers(n int) ScanOption {
	return func(o *ScanOptions) {
		if n >= 0 {
			o.Workers = n
		}
	}
}

// WithTimeout sets the scan timeout duration.
// Negative values are ignored.
func WithTimeout(d time.Duration) ScanOption {
	return func(o *ScanOptions) {
		if d >= 0 {
			o.Timeout = d
		}
	}
}

// WithDomainHints enables or disables domain hints extraction.
// Domain hints include imports, function calls, and variable names
// useful for AI-based domain classification.
// Default: true (enabled).
func WithDomainHints(enabled bool) ScanOption {
	return func(o *ScanOptions) {
		o.ExtractDomainHints = enabled
	}
}

// WithExcludePatterns adds directory patterns to skip during file discovery.
func WithExcludePatterns(patterns []string) ScanOption {
	return func(o *ScanOptions) {
		o.ExcludePatterns = patterns
	}
}

// WithMaxFileSize sets the maximum file size to process.
func WithMaxFileSize(size int64) ScanOption {
	return func(o *ScanOptions) {
		o.MaxFileSize = size
	}
}

// WithPatterns sets glob patterns to filter test files.
func WithPatterns(patterns []string) ScanOption {
	return func(o *ScanOptions) {
		o.Patterns = patterns
	}
}

// WithRegistry sets the framework registry to use.
func WithRegistry(registry *framework.Registry) ScanOption {
	return func(o *ScanOptions) {
		o.Registry = registry
	}
}

func applyDefaults(opts *ScanOptions) {
	// ExtractDomainHints defaults to true when opts is zero-initialized.
	// Since bool zero value is false, we need special handling.
	// Users must explicitly call WithDomainHints(false) to disable.
	// This is handled by initializing to true in newDefaultOptions().
	if opts.MaxFileSize <= 0 {
		opts.MaxFileSize = DefaultMaxFileSize
	}
	if opts.Registry == nil {
		opts.Registry = framework.DefaultRegistry()
	}
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultTimeout
	}
}

// newDefaultOptions returns ScanOptions with default values.
func newDefaultOptions() ScanOptions {
	return ScanOptions{
		ExtractDomainHints: true,
	}
}

// Backward compatibility aliases

// WithExclude is an alias for WithExcludePatterns.
func WithExclude(patterns []string) ScanOption {
	return WithExcludePatterns(patterns)
}

// WithScanPatterns is an alias for WithPatterns.
func WithScanPatterns(patterns []string) ScanOption {
	return WithPatterns(patterns)
}

// WithScanMaxFileSize is an alias for WithMaxFileSize.
func WithScanMaxFileSize(size int64) ScanOption {
	return func(o *ScanOptions) {
		if size >= 0 {
			o.MaxFileSize = size
		}
	}
}
