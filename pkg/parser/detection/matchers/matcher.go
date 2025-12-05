// Package matchers provides framework-specific detection rules.
package matchers

import (
	"context"

	"github.com/specvital/core/pkg/domain"
)

// Priority determines detection order when multiple frameworks could match.
// Higher values = checked first. Use increments of 50 for future insertions.
const (
	PriorityGeneric     = 100 // jest, go-testing
	PriorityE2E         = 150 // playwright
	PrioritySpecialized = 200 // vitest (has specific globals mode)
)

type ConfigInfo struct {
	Framework   string
	GlobalsMode bool
}

// Matcher defines the interface for framework-specific detection.
type Matcher interface {
	Name() string
	Languages() []domain.Language
	MatchImport(importPath string) bool
	ConfigPatterns() []string
	ExtractImports(ctx context.Context, content []byte) []string
	ParseConfig(ctx context.Context, content []byte) *ConfigInfo
	Priority() int
}
