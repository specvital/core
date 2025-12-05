// Package matchers provides framework-specific detection rules.
package matchers

import (
	"context"

	"github.com/specvital/core/pkg/domain"
)

type Matcher interface {
	Name() string
	Languages() []domain.Language
	MatchImport(importPath string) bool
	ConfigPatterns() []string
	ExtractImports(ctx context.Context, content []byte) []string
}
