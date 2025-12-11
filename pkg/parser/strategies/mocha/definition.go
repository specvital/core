package mocha

import (
	"context"
	"regexp"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
	"github.com/specvital/core/pkg/parser/framework/matchers"
	"github.com/specvital/core/pkg/parser/strategies/shared/configutil"
	"github.com/specvital/core/pkg/parser/strategies/shared/jstest"
)

const frameworkName = "mocha"

func init() {
	framework.Register(NewDefinition())
}

func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      frameworkName,
		Languages: []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("mocha", "mocha/"),
			matchers.NewConfigMatcher(
				".mocharc.cjs",
				".mocharc.js",
				".mocharc.json",
				".mocharc.jsonc",
				".mocharc.mjs",
				".mocharc.yaml",
				".mocharc.yml",
				"mocha.opts",
			),
			&MochaContentMatcher{},
		},
		ConfigParser: &MochaConfigParser{},
		Parser:       &MochaParser{},
		Priority:     framework.PriorityGeneric,
	}
}

// MochaContentMatcher detects Mocha-specific API patterns in file content.
type MochaContentMatcher struct{}

var mochaPatterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	{regexp.MustCompile(`\bthis\.timeout\s*\(`), "this.timeout()"},
	{regexp.MustCompile(`\bthis\.slow\s*\(`), "this.slow()"},
	{regexp.MustCompile(`\bthis\.retries\s*\(`), "this.retries()"},
	{regexp.MustCompile(`\bthis\.skip\s*\(\s*\)`), "this.skip()"},
	{regexp.MustCompile(`\bthis\.currentTest\b`), "this.currentTest"},
	{regexp.MustCompile(`\bmocha\.setup\s*\(`), "mocha.setup()"},
	{regexp.MustCompile(`\bmocha\.run\s*\(`), "mocha.run()"},
}

func (m *MochaContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileContent {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		content = []byte(signal.Value)
	}

	for _, p := range mochaPatterns {
		if p.pattern.Match(content) {
			return framework.PartialMatch(40, "Found Mocha-specific pattern: "+p.desc)
		}
	}

	return framework.NoMatch()
}

type MochaConfigParser struct{}

func (p *MochaConfigParser) Parse(ctx context.Context, configPath string, content []byte) (*framework.ConfigScope, error) {
	// Mocha doesn't have a standard root directory config field.
	// Root is always the directory containing the config file.
	scope := framework.NewConfigScope(configPath, "")
	scope.Framework = frameworkName
	scope.GlobalsMode = true
	scope.TestPatterns = parseSpec(content)
	return scope, nil
}

// Config parsing regex patterns.
// Limitation: Escaped quotes, nested structures, and template literals are not fully supported.
// This is acceptable because scope-based detection is an optimization;
// import/content matchers provide baseline detection.
var (
	specSinglePattern = regexp.MustCompile(`["']?spec["']?\s*:\s*['"]([^'"]+)['"]`)
	specArrayPattern  = regexp.MustCompile(`["']?spec["']?\s*:\s*\[([^\]]+)\]`)
)

func parseSpec(content []byte) []string {
	if match := specSinglePattern.FindSubmatch(content); match != nil {
		return []string{string(match[1])}
	}
	if match := specArrayPattern.FindSubmatch(content); match != nil {
		return configutil.ExtractQuotedStrings(match[1])
	}
	return nil
}

type MochaParser struct{}

func (p *MochaParser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	return jstest.Parse(ctx, source, filename, frameworkName)
}
