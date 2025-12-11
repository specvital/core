package cypress

import (
	"context"
	"regexp"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
	"github.com/specvital/core/pkg/parser/framework/matchers"
	"github.com/specvital/core/pkg/parser/strategies/shared/configutil"
	"github.com/specvital/core/pkg/parser/strategies/shared/jstest"
)

const frameworkName = "cypress"

func init() {
	framework.Register(NewDefinition())
}

func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      frameworkName,
		Languages: []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("cypress", "cypress/"),
			matchers.NewConfigMatcher(
				"cypress.config.cjs",
				"cypress.config.js",
				"cypress.config.mjs",
				"cypress.config.mts",
				"cypress.config.ts",
			),
			&CypressFilenameMatcher{},
			&CypressContentMatcher{},
		},
		ConfigParser: &CypressConfigParser{},
		Parser:       &CypressParser{},
		Priority:     framework.PriorityE2E,
	}
}

// CypressFilenameMatcher matches Cypress-specific file patterns (.cy.{js,ts,jsx,tsx}).
type CypressFilenameMatcher struct{}

var cypressFilenamePattern = regexp.MustCompile(`\.cy\.(js|ts|jsx|tsx)$`)

func (m *CypressFilenameMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileName {
		return framework.NoMatch()
	}

	if cypressFilenamePattern.MatchString(signal.Value) {
		return framework.DefiniteMatch("filename: *.cy.{js,ts,jsx,tsx}")
	}

	return framework.NoMatch()
}

// CypressContentMatcher matches Cypress-specific patterns (cy.*, Cypress.*).
type CypressContentMatcher struct{}

var cypressPatterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	{regexp.MustCompile(`\bcy\.visit\s*\(`), "cy.visit()"},
	{regexp.MustCompile(`\bcy\.get\s*\(`), "cy.get()"},
	{regexp.MustCompile(`\bcy\.contains\s*\(`), "cy.contains()"},
	{regexp.MustCompile(`\bcy\.intercept\s*\(`), "cy.intercept()"},
	{regexp.MustCompile(`\bcy\.request\s*\(`), "cy.request()"},
	{regexp.MustCompile(`\bcy\.wait\s*\(`), "cy.wait()"},
	{regexp.MustCompile(`\bcy\.fixture\s*\(`), "cy.fixture()"},
	{regexp.MustCompile(`\bCypress\.Commands\.add\s*\(`), "Cypress.Commands.add()"},
	{regexp.MustCompile(`\bCypress\.env\s*\(`), "Cypress.env()"},
}

func (m *CypressContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileContent {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		content = []byte(signal.Value)
	}

	for _, p := range cypressPatterns {
		if p.pattern.Match(content) {
			return framework.PartialMatch(40, "Found Cypress-specific pattern: "+p.desc)
		}
	}

	return framework.NoMatch()
}

type CypressConfigParser struct{}

func (p *CypressConfigParser) Parse(ctx context.Context, configPath string, content []byte) (*framework.ConfigScope, error) {
	scope := framework.NewConfigScope(configPath, "")
	scope.Framework = frameworkName
	scope.GlobalsMode = true // Cypress injects globals (cy, Cypress) by default

	e2ePatterns := parseSpecPattern(content, "e2e")
	componentPatterns := parseSpecPattern(content, "component")
	scope.TestPatterns = append(e2ePatterns, componentPatterns...)
	scope.ExcludePatterns = parseExcludeSpecPattern(content)

	return scope, nil
}

// Config parsing regex patterns.
// Limitation: Complex expressions, multiline patterns, and template literals are not fully supported.
// This is acceptable because scope-based detection is an optimization;
// filename/content matchers provide baseline detection.
var (
	e2eSpecPatternSingleRegex       = regexp.MustCompile(`(?s)e2e\s*:\s*\{[\s\S]*?specPattern\s*:\s*['"]([^'"]+)['"]`)
	e2eSpecPatternArrayRegex        = regexp.MustCompile(`(?s)e2e\s*:\s*\{[\s\S]*?specPattern\s*:\s*\[([^\]]+)\]`)
	componentSpecPatternSingleRegex = regexp.MustCompile(`(?s)component\s*:\s*\{[\s\S]*?specPattern\s*:\s*['"]([^'"]+)['"]`)
	componentSpecPatternArrayRegex  = regexp.MustCompile(`(?s)component\s*:\s*\{[\s\S]*?specPattern\s*:\s*\[([^\]]+)\]`)
	excludeSpecPatternSingleRegex   = regexp.MustCompile(`excludeSpecPattern\s*:\s*['"]([^'"]+)['"]`)
	excludeSpecPatternArrayRegex    = regexp.MustCompile(`excludeSpecPattern\s*:\s*\[([^\]]+)\]`)
)

func parsePattern(content []byte, singleRegex, arrayRegex *regexp.Regexp) []string {
	if match := singleRegex.FindSubmatch(content); match != nil {
		return []string{string(match[1])}
	}
	if match := arrayRegex.FindSubmatch(content); match != nil {
		return configutil.ExtractQuotedStrings(match[1])
	}
	return nil
}

func parseSpecPattern(content []byte, section string) []string {
	switch section {
	case "e2e":
		return parsePattern(content, e2eSpecPatternSingleRegex, e2eSpecPatternArrayRegex)
	case "component":
		return parsePattern(content, componentSpecPatternSingleRegex, componentSpecPatternArrayRegex)
	default:
		return nil
	}
}

func parseExcludeSpecPattern(content []byte) []string {
	return parsePattern(content, excludeSpecPatternSingleRegex, excludeSpecPatternArrayRegex)
}

type CypressParser struct{}

func (p *CypressParser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	return jstest.Parse(ctx, source, filename, frameworkName)
}
