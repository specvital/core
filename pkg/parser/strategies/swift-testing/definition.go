// Package swifttesting implements Swift Testing framework support for Swift test files.
// Swift Testing is Apple's modern testing framework introduced with Xcode 16 (2024),
// using @Test and @Suite attributes instead of XCTest's naming conventions.
package swifttesting

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
	"github.com/specvital/core/pkg/parser/strategies/shared/swiftast"
)

const frameworkName = framework.FrameworkSwiftTesting

const (
	confidenceFileName = 20
	confidenceContent  = 40
)

func init() {
	framework.Register(NewDefinition())
}

func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      frameworkName,
		Languages: []domain.Language{domain.LanguageSwift},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("import Testing"),
			&SwiftTestingFileMatcher{},
			&SwiftTestingContentMatcher{},
		},
		ConfigParser: nil,
		Parser:       &SwiftTestingParser{},
		Priority:     framework.PrioritySpecialized,
	}
}

// SwiftTestingFileMatcher matches Swift test files.
type SwiftTestingFileMatcher struct{}

func (m *SwiftTestingFileMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileName {
		return framework.NoMatch()
	}

	if swiftast.IsSwiftTestFile(signal.Value) {
		return framework.PartialMatch(confidenceFileName, "Swift Testing file naming convention")
	}

	return framework.NoMatch()
}

// SwiftTestingContentMatcher matches Swift Testing-specific patterns.
type SwiftTestingContentMatcher struct{}

var swiftTestingPatterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	{regexp.MustCompile(`@Test\b`), "@Test attribute"},
	{regexp.MustCompile(`@Suite\b`), "@Suite attribute"},
	{regexp.MustCompile(`#expect\(`), "#expect macro"},
	{regexp.MustCompile(`#require\(`), "#require macro"},
	{regexp.MustCompile(`import\s+Testing`), "Testing import"},
}

func (m *SwiftTestingContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileContent {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		content = []byte(signal.Value)
	}

	for _, p := range swiftTestingPatterns {
		if p.pattern.Match(content) {
			return framework.PartialMatch(confidenceContent, "Found Swift Testing pattern: "+p.desc)
		}
	}

	return framework.NoMatch()
}

// SwiftTestingParser extracts test definitions from Swift Testing files.
type SwiftTestingParser struct{}

func (p *SwiftTestingParser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	tree, err := parser.ParseWithPool(ctx, domain.LanguageSwift, source)
	if err != nil {
		return nil, fmt.Errorf("swift-testing parser: failed to parse %s: %w", filename, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	suites := parseSuites(root, source, filename)

	return &domain.TestFile{
		Path:      filename,
		Language:  domain.LanguageSwift,
		Framework: frameworkName,
		Suites:    suites,
	}, nil
}

func parseSuites(root *sitter.Node, source []byte, filename string) []domain.TestSuite {
	var suites []domain.TestSuite

	parser.WalkTree(root, func(node *sitter.Node) bool {
		if node.Type() == swiftast.NodeClassDeclaration {
			if suite := parseSuiteDeclaration(node, source, filename); suite != nil {
				suites = append(suites, *suite)
			}
			return false
		}
		return true
	})

	return suites
}

func parseSuiteDeclaration(node *sitter.Node, source []byte, filename string) *domain.TestSuite {
	name := swiftast.GetClassName(node, source)
	if name == "" {
		return nil
	}

	if !hasAttribute(node, source, "@Suite") && !hasTestFunctionsInside(node, source) {
		return nil
	}

	body := swiftast.GetClassBody(node)
	if body == nil {
		return nil
	}

	suite := &domain.TestSuite{
		Name:     name,
		Status:   domain.TestStatusActive,
		Location: parser.GetLocation(node, filename),
	}

	parseTestFunctions(body, source, filename, suite)

	if len(suite.Tests) == 0 {
		return nil
	}

	return suite
}

func hasTestFunctionsInside(node *sitter.Node, source []byte) bool {
	body := swiftast.GetClassBody(node)
	if body == nil {
		return false
	}

	found := false
	parser.WalkTree(body, func(n *sitter.Node) bool {
		if n.Type() == swiftast.NodeFunctionDeclaration {
			if hasAttribute(n, source, "@Test") {
				found = true
				return false
			}
		}
		return true
	})

	return found
}

func parseTestFunctions(body *sitter.Node, source []byte, filename string, suite *domain.TestSuite) {
	parser.WalkTree(body, func(node *sitter.Node) bool {
		if node.Type() == swiftast.NodeFunctionDeclaration {
			if test := parseTestFunction(node, source, filename); test != nil {
				suite.Tests = append(suite.Tests, *test)
			}
			return false
		}
		return true
	})
}

func parseTestFunction(node *sitter.Node, source []byte, filename string) *domain.Test {
	if !hasAttribute(node, source, "@Test") {
		return nil
	}

	funcName := swiftast.GetFunctionName(node, source)
	if funcName == "" {
		return nil
	}

	status := domain.TestStatusActive
	modifier := ""

	if hasAttributeContaining(node, source, ".disabled") {
		status = domain.TestStatusSkipped
		modifier = "@Test(.disabled)"
	}

	if isAsyncFunction(node, source) {
		modifier = appendModifier(modifier, "async")
	}

	return &domain.Test{
		Name:     funcName,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(node, filename),
	}
}

// hasAttribute checks if a node has an attribute with the given prefix.
func hasAttribute(node *sitter.Node, source []byte, prefix string) bool {
	return findAttribute(node, source, func(content string) bool {
		return strings.HasPrefix(content, prefix)
	})
}

// hasAttributeContaining checks if a node has an attribute containing the given substring.
func hasAttributeContaining(node *sitter.Node, source []byte, substr string) bool {
	return findAttribute(node, source, func(content string) bool {
		return strings.Contains(content, substr)
	})
}

// findAttribute traverses modifiers to find an attribute matching the predicate.
func findAttribute(node *sitter.Node, source []byte, predicate func(string) bool) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == swiftast.NodeModifiers {
			for j := 0; j < int(child.ChildCount()); j++ {
				attr := child.Child(j)
				if attr.Type() == swiftast.NodeAttribute && predicate(attr.Content(source)) {
					return true
				}
			}
		}
	}
	return false
}

func isAsyncFunction(node *sitter.Node, source []byte) bool {
	content := node.Content(source)
	return strings.Contains(content, "async ")
}

func appendModifier(existing, new string) string {
	if existing == "" {
		return new
	}
	return existing + ", " + new
}
