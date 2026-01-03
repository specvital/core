package detection

import "fmt"

// DetectionSource indicates how the framework was detected.
type DetectionSource string

const (
	// SourceImport indicates detection via explicit import statement (highest confidence).
	SourceImport DetectionSource = "import"

	// SourceStrongFilename indicates detection via strong filename pattern (e.g., *.cy.ts).
	// This takes precedence over config scope because explicit filename patterns
	// represent clear developer intent.
	SourceStrongFilename DetectionSource = "strong-filename"

	// SourceConfigScope indicates detection via config file scope.
	SourceConfigScope DetectionSource = "config-scope"

	// SourceContentPattern indicates detection via content pattern matching.
	SourceContentPattern DetectionSource = "content-pattern"

	// SourceUnknown indicates no framework was detected.
	SourceUnknown DetectionSource = "unknown"
)

// Result represents the outcome of framework detection for a test file.
type Result struct {
	// Framework is the detected framework name (e.g., "jest", "vitest").
	// Empty string if no framework detected.
	Framework string

	// Source indicates how the framework was detected.
	Source DetectionSource

	// Scope is the config scope that applies to this file (if scope-based detection succeeded).
	// May be nil if no config scope applies.
	Scope interface{} // framework.ConfigScope, but avoid import cycle
}

// IsDetected returns true if a framework was detected.
func (r Result) IsDetected() bool {
	return r.Framework != "" && r.Source != SourceUnknown
}

func (r Result) String() string {
	if r.Framework == "" {
		return "no framework detected"
	}
	return fmt.Sprintf("%s (source: %s)", r.Framework, r.Source)
}

// Unknown returns a Result indicating no framework was detected.
func Unknown() Result {
	return Result{
		Framework: "",
		Source:    SourceUnknown,
	}
}

// Confirmed returns a Result indicating a framework was definitively detected.
func Confirmed(framework string, source DetectionSource) Result {
	return Result{
		Framework: framework,
		Source:    source,
	}
}

// ConfirmedWithScope returns a Result with config scope information.
func ConfirmedWithScope(framework string, scope interface{}) Result {
	return Result{
		Framework: framework,
		Source:    SourceConfigScope,
		Scope:     scope,
	}
}
