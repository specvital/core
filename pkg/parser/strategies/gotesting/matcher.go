package gotesting

import (
	"context"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/detection/extraction"
	"github.com/specvital/core/pkg/parser/detection/matchers"
)

func init() {
	matchers.Register(&Matcher{})
}

type Matcher struct{}

func (m *Matcher) Name() string                       { return frameworkName }
func (m *Matcher) Languages() []domain.Language       { return []domain.Language{domain.LanguageGo} }
func (m *Matcher) MatchImport(importPath string) bool { return importPath == "testing" }
func (m *Matcher) ConfigPatterns() []string           { return nil }

func (m *Matcher) ExtractImports(ctx context.Context, content []byte) []string {
	return extraction.ExtractGoImports(ctx, content)
}
