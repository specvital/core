package vitest

import (
	"context"
	"regexp"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/detection/extraction"
	"github.com/specvital/core/pkg/parser/framework"
	"github.com/specvital/core/pkg/parser/framework/matchers"
	"github.com/specvital/core/pkg/parser/strategies/shared/jstest"
)

const frameworkName = "vitest"

func init() {
	framework.Register(NewDefinition())
}

func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      frameworkName,
		Languages: []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("vitest", "vitest/"),
			matchers.NewConfigMatcher(
				"vitest.config.js",
				"vitest.config.ts",
				"vitest.config.mjs",
				"vitest.config.mts",
			),
			&VitestContentMatcher{},
		},
		ConfigParser: &VitestConfigParser{},
		Parser:       &VitestParser{},
		Priority:     framework.PrioritySpecialized,
	}
}

type VitestConfigParser struct{}

func (p *VitestConfigParser) Parse(ctx context.Context, configPath string, content []byte) (*framework.ConfigScope, error) {
	root := parseRoot(content)
	scope := framework.NewConfigScope(configPath, root)
	scope.Framework = frameworkName
	scope.GlobalsMode = parseGlobals(ctx, content)
	scope.Include = parseInclude(content)
	scope.Exclude = parseExclude(content)
	return scope, nil
}

type VitestParser struct{}

func (p *VitestParser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	return jstest.Parse(ctx, source, filename, frameworkName)
}

var (
	configRootPattern    = regexp.MustCompile(`root\s*:\s*['"]([^'"]+)['"]`)
	configGlobalsPattern = regexp.MustCompile(`globals\s*:\s*true`)
	// Pattern to remove coverage block (which has its own include/exclude)
	configCoveragePattern = regexp.MustCompile(`(?s)coverage\s*:\s*\{[^}]*(?:\{[^}]*\}[^}]*)*\}`)
	configIncludePattern  = regexp.MustCompile(`(?:^|[,\s])include\s*:\s*\[([^\]]+)\]`)
	configExcludePattern  = regexp.MustCompile(`(?:^|[,\s])exclude\s*:\s*\[([^\]]+)\]`)
	configItemPattern     = regexp.MustCompile(`['"]([^'"]+)['"]`)
)

func parseRoot(content []byte) string {
	if match := configRootPattern.FindSubmatch(content); match != nil {
		return string(match[1])
	}
	return ""
}

func parseGlobals(ctx context.Context, content []byte) bool {
	return extraction.MatchPatternExcludingComments(ctx, content, configGlobalsPattern)
}

func parseInclude(content []byte) []string {
	// Remove coverage block first to avoid matching coverage.include
	cleaned := configCoveragePattern.ReplaceAll(content, []byte{})
	match := configIncludePattern.FindSubmatch(cleaned)
	if match == nil {
		return nil
	}
	return extractPatterns(match[1])
}

func parseExclude(content []byte) []string {
	// Remove coverage block first to avoid matching coverage.exclude
	cleaned := configCoveragePattern.ReplaceAll(content, []byte{})
	match := configExcludePattern.FindSubmatch(cleaned)
	if match == nil {
		return nil
	}
	return extractPatterns(match[1])
}

func extractPatterns(content []byte) []string {
	items := configItemPattern.FindAllSubmatch(content, -1)
	if len(items) == 0 {
		return nil
	}
	var patterns []string
	for _, item := range items {
		patterns = append(patterns, string(item[1]))
	}
	return patterns
}

// VitestContentMatcher matches vitest-specific patterns (vi.fn, vi.mock, etc.).
type VitestContentMatcher struct{}

var vitestPatterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	{regexp.MustCompile(`\bbench\s*\(`), "bench()"},
	{regexp.MustCompile(`\bbench\.skip\s*\(`), "bench.skip()"},
	{regexp.MustCompile(`\bbench\.only\s*\(`), "bench.only()"},
	{regexp.MustCompile(`\bvi\.fn\s*\(`), "vi.fn()"},
	{regexp.MustCompile(`\bvi\.mock\s*\(`), "vi.mock()"},
	{regexp.MustCompile(`\bvi\.spyOn\s*\(`), "vi.spyOn()"},
	{regexp.MustCompile(`\bvi\.useFakeTimers\s*\(`), "vi.useFakeTimers()"},
	{regexp.MustCompile(`\bvi\.clearAllMocks\s*\(`), "vi.clearAllMocks()"},
	{regexp.MustCompile(`\bvi\.resetAllMocks\s*\(`), "vi.resetAllMocks()"},
	{regexp.MustCompile(`\bvi\.restoreAllMocks\s*\(`), "vi.restoreAllMocks()"},
	{regexp.MustCompile(`\bvi\.stubGlobal\s*\(`), "vi.stubGlobal()"},
	{regexp.MustCompile(`\bvi\.stubEnv\s*\(`), "vi.stubEnv()"},
}

func (m *VitestContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileContent {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		content = []byte(signal.Value)
	}

	for _, p := range vitestPatterns {
		if p.pattern.Match(content) {
			return framework.PartialMatch(40, "Found Vitest-specific pattern: "+p.desc)
		}
	}

	return framework.NoMatch()
}
