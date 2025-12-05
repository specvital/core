package jest

import (
	"context"
	"strings"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/detection/extraction"
	"github.com/specvital/core/pkg/parser/detection/matchers"
)

func init() {
	matchers.Register(&Matcher{})
}

type Matcher struct{}

func (m *Matcher) Name() string { return frameworkName }
func (m *Matcher) Languages() []domain.Language {
	return []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript}
}

func (m *Matcher) MatchImport(importPath string) bool {
	return importPath == "@jest/globals" || importPath == "jest" || strings.HasPrefix(importPath, "@jest/")
}

func (m *Matcher) ConfigPatterns() []string {
	return []string{"jest.config.js", "jest.config.ts", "jest.config.mjs", "jest.config.cjs", "jest.config.json"}
}

func (m *Matcher) ExtractImports(ctx context.Context, content []byte) []string {
	return extraction.ExtractJSImports(ctx, content)
}
