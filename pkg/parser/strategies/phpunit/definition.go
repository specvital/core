// Package phpunit implements PHPUnit test framework support for PHP test files.
package phpunit

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser"
	"github.com/specvital/core/pkg/parser/framework"
	"github.com/specvital/core/pkg/parser/framework/matchers"
	"github.com/specvital/core/pkg/parser/strategies/shared/phpast"
)

const frameworkName = "phpunit"

func init() {
	framework.Register(NewDefinition())
}

func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      frameworkName,
		Languages: []domain.Language{domain.LanguagePHP},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher(
				"PHPUnit\\Framework\\TestCase",
				"use PHPUnit\\",
			),
			matchers.NewConfigMatcher(
				"phpunit.xml",
				"phpunit.xml.dist",
				"phpunit.dist.xml",
			),
			&PHPUnitFileMatcher{},
			&PHPUnitContentMatcher{},
		},
		ConfigParser: nil,
		Parser:       &PHPUnitParser{},
		Priority:     framework.PriorityGeneric,
	}
}

// PHPUnitFileMatcher matches *Test.php, *Tests.php, Test*.php files.
type PHPUnitFileMatcher struct{}

func (m *PHPUnitFileMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileName {
		return framework.NoMatch()
	}

	if phpast.IsPHPTestFileName(signal.Value) {
		return framework.PartialMatch(20, "PHPUnit file naming convention")
	}

	return framework.NoMatch()
}

// PHPUnitContentMatcher matches PHPUnit specific patterns.
type PHPUnitContentMatcher struct{}

var phpunitPatterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	{regexp.MustCompile(`extends\s+TestCase`), "extends TestCase"},
	{regexp.MustCompile(`use\s+PHPUnit\\Framework\\TestCase`), "use PHPUnit\\Framework\\TestCase"},
	{regexp.MustCompile(`@test\b`), "@test annotation"},
	{regexp.MustCompile(`@dataProvider\s+\w+`), "@dataProvider annotation"},
	{regexp.MustCompile(`#\[Test\]`), "#[Test] attribute"},
	{regexp.MustCompile(`#\[DataProvider\(`), "#[DataProvider] attribute"},
	{regexp.MustCompile(`\$this->assert\w+\(`), "PHPUnit assertion"},
	{regexp.MustCompile(`\$this->expect\w+\(`), "PHPUnit expectation"},
}

func (m *PHPUnitContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileContent {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		content = []byte(signal.Value)
	}

	for _, p := range phpunitPatterns {
		if p.pattern.Match(content) {
			return framework.PartialMatch(40, "Found PHPUnit pattern: "+p.desc)
		}
	}

	return framework.NoMatch()
}

// PHPUnitParser extracts test definitions from PHP PHPUnit files.
type PHPUnitParser struct{}

func (p *PHPUnitParser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	tree, err := parser.ParseWithPool(ctx, domain.LanguagePHP, source)
	if err != nil {
		return nil, fmt.Errorf("phpunit parser: failed to parse %s: %w", filename, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	suites := parseTestClasses(root, source, filename)

	return &domain.TestFile{
		Path:      filename,
		Language:  domain.LanguagePHP,
		Framework: frameworkName,
		Suites:    suites,
	}, nil
}

func parseTestClasses(root *sitter.Node, source []byte, filename string) []domain.TestSuite {
	var suites []domain.TestSuite

	parser.WalkTree(root, func(node *sitter.Node) bool {
		if node.Type() == phpast.NodeClassDeclaration {
			if suite := parseTestClass(node, source, filename); suite != nil {
				suites = append(suites, *suite)
			}
			return false
		}
		return true
	})

	return suites
}

func parseTestClass(node *sitter.Node, source []byte, filename string) *domain.TestSuite {
	className := phpast.GetClassName(node, source)
	if className == "" {
		return nil
	}

	if !phpast.ExtendsTestCase(node, source) {
		return nil
	}

	body := phpast.GetDeclarationList(node)
	if body == nil {
		return nil
	}

	var tests []domain.Test
	var prevComment *sitter.Node

	for i := 0; i < int(body.ChildCount()); i++ {
		child := body.Child(i)

		switch child.Type() {
		case phpast.NodeComment:
			prevComment = child

		case phpast.NodeMethodDeclaration:
			if test := parseTestMethod(child, source, filename, prevComment); test != nil {
				tests = append(tests, *test)
			}
			prevComment = nil

		default:
			prevComment = nil
		}
	}

	if len(tests) == 0 {
		return nil
	}

	return &domain.TestSuite{
		Name:     className,
		Status:   domain.TestStatusActive,
		Location: parser.GetLocation(node, filename),
		Tests:    tests,
	}
}

func parseTestMethod(node *sitter.Node, source []byte, filename string, prevComment *sitter.Node) *domain.Test {
	methodName := phpast.GetMethodName(node, source)
	if methodName == "" {
		return nil
	}

	attrs := phpast.GetAttributes(node)
	hasTestAttr := phpast.HasTestAttribute(attrs, source)

	hasTestAnnotation := false
	if prevComment != nil {
		commentText := prevComment.Content(source)
		hasTestAnnotation = phpast.HasTestAnnotation(commentText)
	}

	hasTestPrefix := strings.HasPrefix(methodName, "test")

	if !hasTestAttr && !hasTestAnnotation && !hasTestPrefix {
		return nil
	}

	status := domain.TestStatusActive
	modifier := ""
	if skipped, skipMod := phpast.HasSkipAttribute(attrs, source); skipped {
		status = domain.TestStatusSkipped
		modifier = skipMod
	}

	return &domain.Test{
		Name:     methodName,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(node, filename),
	}
}
