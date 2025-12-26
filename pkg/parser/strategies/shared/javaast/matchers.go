package javaast

import (
	"context"
	"regexp"
	"strings"

	"github.com/specvital/core/pkg/parser/framework"
)

// Import patterns for JUnit version detection.
var (
	// JUnit4ImportPattern detects JUnit 4 imports (org.junit.Test, org.junit.Assert, org.junit.*, etc.)
	// Matches: org.junit.Test, org.junit.Assert, org.junit.* (wildcard)
	// Does NOT match: org.junit.jupiter (handled by JUnit5ImportPattern)
	JUnit4ImportPattern = regexp.MustCompile(`import\s+(?:static\s+)?org\.junit\.(?:\*|[A-Z])`)

	// JUnit5ImportPattern detects JUnit 5 (Jupiter) imports
	JUnit5ImportPattern = regexp.MustCompile(`import\s+(?:static\s+)?org\.junit\.jupiter`)
)

// JavaTestFileMatcher matches common Java test file naming patterns.
// Supports: *Test.java, *Tests.java, Test*.java
// Excludes: src/main/ (production code in Maven/Gradle structure)
type JavaTestFileMatcher struct{}

func (m *JavaTestFileMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileName {
		return framework.NoMatch()
	}

	filename := signal.Value

	// Exclude src/main/ (production code in Maven/Gradle structure)
	if strings.Contains(filename, "/src/main/") || strings.HasPrefix(filename, "src/main/") {
		return framework.NoMatch()
	}

	base := filename
	if idx := strings.LastIndex(filename, "/"); idx >= 0 {
		base = filename[idx+1:]
	}

	if !strings.HasSuffix(base, ".java") {
		return framework.NoMatch()
	}

	name := strings.TrimSuffix(base, ".java")

	if strings.HasSuffix(name, "Test") || strings.HasSuffix(name, "Tests") {
		return framework.PartialMatch(20, "JUnit file naming: *Test.java")
	}

	if strings.HasPrefix(name, "Test") {
		return framework.PartialMatch(20, "JUnit file naming: Test*.java")
	}

	return framework.NoMatch()
}
