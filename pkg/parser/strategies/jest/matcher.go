package jest

import (
	"context"
	"regexp"
	"strings"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/detection/extraction"
	"github.com/specvital/core/pkg/parser/detection/matchers"
)

const matcherPriority = matchers.PriorityGeneric

var injectGlobalsFalsePattern = regexp.MustCompile(`injectGlobals\s*:\s*false`)

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

func (m *Matcher) ParseConfig(ctx context.Context, content []byte) *matchers.ConfigInfo {
	globalsDisabled := extraction.MatchPatternExcludingComments(ctx, content, injectGlobalsFalsePattern)
	return &matchers.ConfigInfo{
		Framework:   frameworkName,
		GlobalsMode: !globalsDisabled,
	}
}

func (m *Matcher) Priority() int {
	return matcherPriority
}
