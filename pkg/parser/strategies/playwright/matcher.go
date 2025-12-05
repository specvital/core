package playwright

import (
	"context"
	"strings"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/detection/extraction"
	"github.com/specvital/core/pkg/parser/detection/matchers"
)

const matcherPriority = matchers.PriorityE2E

func init() {
	matchers.Register(&Matcher{})
}

type Matcher struct{}

func (m *Matcher) Name() string { return frameworkName }

func (m *Matcher) Languages() []domain.Language {
	return []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript}
}

func (m *Matcher) MatchImport(importPath string) bool {
	return importPath == "@playwright/test" || strings.HasPrefix(importPath, "@playwright/test/")
}

func (m *Matcher) ConfigPatterns() []string {
	return []string{"playwright.config.js", "playwright.config.ts", "playwright.config.mjs", "playwright.config.mts"}
}

func (m *Matcher) ExtractImports(ctx context.Context, content []byte) []string {
	return extraction.ExtractJSImports(ctx, content)
}

func (m *Matcher) ParseConfig(_ context.Context, _ []byte) *matchers.ConfigInfo {
	// Playwright always requires explicit imports, no globals mode
	return &matchers.ConfigInfo{
		Framework:   frameworkName,
		GlobalsMode: false,
	}
}

func (m *Matcher) Priority() int {
	return matcherPriority
}
