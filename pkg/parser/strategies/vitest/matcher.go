package vitest

import (
	"context"
	"regexp"
	"strings"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/detection/extraction"
	"github.com/specvital/core/pkg/parser/detection/matchers"
)

const matcherPriority = matchers.PrioritySpecialized

var globalsPattern = regexp.MustCompile(`globals\s*:\s*true`)

func init() {
	matchers.Register(&Matcher{})
}

type Matcher struct{}

func (m *Matcher) Name() string { return frameworkName }

func (m *Matcher) Languages() []domain.Language {
	return []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript}
}

func (m *Matcher) MatchImport(importPath string) bool {
	return importPath == "vitest" || strings.HasPrefix(importPath, "vitest/")
}

func (m *Matcher) ConfigPatterns() []string {
	return []string{"vitest.config.js", "vitest.config.ts", "vitest.config.mjs", "vitest.config.mts"}
}

func (m *Matcher) ExtractImports(ctx context.Context, content []byte) []string {
	return extraction.ExtractJSImports(ctx, content)
}

func (m *Matcher) ParseConfig(ctx context.Context, content []byte) *matchers.ConfigInfo {
	return &matchers.ConfigInfo{
		Framework:   frameworkName,
		GlobalsMode: extraction.MatchPatternExcludingComments(ctx, content, globalsPattern),
	}
}

func (m *Matcher) Priority() int {
	return matcherPriority
}
