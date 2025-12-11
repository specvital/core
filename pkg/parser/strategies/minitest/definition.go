// Package minitest implements Minitest test framework support for Ruby test files.
package minitest

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
	"github.com/specvital/core/pkg/parser/strategies/shared/rubyast"
)

// Constants for Minitest framework parsing.
const (
	frameworkName = "minitest"
	funcDescribe  = "describe"
	funcIt        = "it"

	// maxSkipSearchDepth limits recursion depth for skip detection.
	// Unlike RSpec's static skip modifiers (xdescribe, xit), Minitest uses runtime `skip` calls
	// within test bodies. This requires recursive AST traversal with depth limiting.
	// Value of 20 is chosen based on analysis of real-world Minitest code:
	// - Typical test method depth: 3-5 levels (def -> blocks -> conditionals)
	// - Complex nested structures: rarely exceed 10 levels
	// - Safety margin: 2x typical maximum to handle edge cases
	maxSkipSearchDepth = 20
)

func init() {
	framework.Register(NewDefinition())
}

func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      frameworkName,
		Languages: []domain.Language{domain.LanguageRuby},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher(
				"require 'minitest'",
				"require \"minitest\"",
				"require 'minitest/autorun'",
				"require \"minitest/autorun\"",
				"require 'minitest/spec'",
				"require \"minitest/spec\"",
			),
			&MinitestFileMatcher{},
			&MinitestContentMatcher{},
		},
		ConfigParser: nil,
		Parser:       &MinitestParser{},
		Priority:     framework.PriorityGeneric,
	}
}

// MinitestFileMatcher matches *_test.rb and test/**/*.rb files.
type MinitestFileMatcher struct{}

func (m *MinitestFileMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileName {
		return framework.NoMatch()
	}

	filename := signal.Value
	base := filename
	if idx := strings.LastIndex(filename, "/"); idx >= 0 {
		base = filename[idx+1:]
	}

	if strings.HasSuffix(base, "_test.rb") {
		return framework.PartialMatch(20, "Minitest file naming: *_test.rb")
	}

	if strings.HasPrefix(filename, "test/") && strings.HasSuffix(base, ".rb") {
		return framework.PartialMatch(15, "Minitest file in test/ directory")
	}

	return framework.NoMatch()
}

// MinitestContentMatcher matches Minitest-specific patterns.
type MinitestContentMatcher struct{}

var minitestPatterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	{regexp.MustCompile(`class\s+\w+\s*<\s*Minitest::Test`), "Minitest::Test class"},
	{regexp.MustCompile(`class\s+\w+\s*<\s*Minitest::Spec`), "Minitest::Spec class"},
	{regexp.MustCompile(`\bdef\s+test_\w+`), "def test_* method"},
	{regexp.MustCompile(`\bdescribe\s+(?:'[^']*'|"[^"]*"|[\w:]+)\s+do\b`), "describe block"},
	{regexp.MustCompile(`\bit\s+(?:'[^']*'|"[^"]*")\s+do\b`), "it block"},
	{regexp.MustCompile(`\bassert\s*[\(\s]`), "assert assertion"},
	{regexp.MustCompile(`\bassert_equal\s*[\(\s]`), "assert_equal assertion"},
	{regexp.MustCompile(`\brefute\s*[\(\s]`), "refute assertion"},
	{regexp.MustCompile(`\bmust_equal\s*[\(\s]`), "must_equal expectation"},
	{regexp.MustCompile(`\bwont_be\s*[\(\s]`), "wont_be expectation"},
}

func (m *MinitestContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileContent {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		content = []byte(signal.Value)
	}

	for _, p := range minitestPatterns {
		if p.pattern.Match(content) {
			return framework.PartialMatch(40, "Found Minitest pattern: "+p.desc)
		}
	}

	return framework.NoMatch()
}

// MinitestParser extracts test definitions from Ruby Minitest files.
type MinitestParser struct{}

func (p *MinitestParser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	tree, err := parser.ParseWithPool(ctx, domain.LanguageRuby, source)
	if err != nil {
		return nil, fmt.Errorf("minitest parser: failed to parse %s: %w", filename, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	file := &domain.TestFile{
		Path:      filename,
		Language:  domain.LanguageRuby,
		Framework: frameworkName,
	}

	parseNode(root, source, filename, file, nil)
	return file, nil
}

func parseNode(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		processNode(child, source, filename, file, currentSuite)
	}
}

func processNode(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	switch node.Type() {
	case "class":
		processClass(node, source, filename, file, currentSuite)
	case "method":
		processMethod(node, source, filename, file, currentSuite)
	case rubyast.NodeCall, rubyast.NodeMethodCall:
		processCallExpression(node, source, filename, file, currentSuite)
	default:
		parseNode(node, source, filename, file, currentSuite)
	}
}

func processClass(node *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite) {
	superclass := node.ChildByFieldName("superclass")
	if superclass == nil {
		parseNode(node, source, filename, file, parentSuite)
		return
	}

	superclassName := parser.GetNodeText(superclass, source)
	if !strings.Contains(superclassName, "Minitest::Test") &&
		!strings.Contains(superclassName, "Minitest::Spec") &&
		!strings.HasSuffix(superclassName, "Test") {
		parseNode(node, source, filename, file, parentSuite)
		return
	}

	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		parseNode(node, source, filename, file, parentSuite)
		return
	}

	className := parser.GetNodeText(nameNode, source)
	suite := domain.TestSuite{
		Name:     className,
		Status:   domain.TestStatusActive,
		Location: parser.GetLocation(node, filename),
	}

	body := node.ChildByFieldName("body")
	if body != nil {
		parseNode(body, source, filename, file, &suite)
	}

	addSuiteToTarget(suite, parentSuite, file)
}

func processMethod(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	methodName := parser.GetNodeText(nameNode, source)
	if !strings.HasPrefix(methodName, "test_") {
		return
	}

	status := domain.TestStatusActive
	body := node.ChildByFieldName("body")
	if body != nil && hasSkipCall(body, source) {
		status = domain.TestStatusSkipped
	}

	test := domain.Test{
		Name:     methodName,
		Status:   status,
		Location: parser.GetLocation(node, filename),
	}

	addTestToTarget(test, currentSuite, file)
}

func hasSkipCall(node *sitter.Node, source []byte) bool {
	return hasSkipCallWithDepth(node, source, 0)
}

func hasSkipCallWithDepth(node *sitter.Node, source []byte, depth int) bool {
	if depth > maxSkipSearchDepth {
		return false
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == rubyast.NodeCall || child.Type() == rubyast.NodeMethodCall {
			methodNode := child.ChildByFieldName("method")
			if methodNode != nil && parser.GetNodeText(methodNode, source) == "skip" {
				return true
			}
			nameNode := parser.FindChildByType(child, rubyast.NodeIdentifier)
			if nameNode != nil && parser.GetNodeText(nameNode, source) == "skip" {
				return true
			}
		} else if child.Type() == rubyast.NodeIdentifier {
			if parser.GetNodeText(child, source) == "skip" {
				return true
			}
		}
		if hasSkipCallWithDepth(child, source, depth+1) {
			return true
		}
	}
	return false
}

func processCallExpression(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	funcName := parseFunctionName(node, source)
	if funcName == "" {
		return
	}

	switch funcName {
	case funcDescribe:
		processSuite(node, source, filename, file, currentSuite)
	case funcIt:
		processTest(node, source, filename, file, currentSuite)
	}
}

func parseFunctionName(node *sitter.Node, source []byte) string {
	// Handle method call: receiver.method
	methodNode := node.ChildByFieldName("method")
	if methodNode != nil {
		return parser.GetNodeText(methodNode, source)
	}

	// Handle simple call: method_name
	nameNode := parser.FindChildByType(node, rubyast.NodeIdentifier)
	if nameNode != nil {
		return parser.GetNodeText(nameNode, source)
	}

	return ""
}

func processSuite(node *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite) {
	name := extractName(node, source)
	if name == "" {
		return
	}

	suite := domain.TestSuite{
		Name:     name,
		Status:   domain.TestStatusActive,
		Location: parser.GetLocation(node, filename),
	}

	// Parse the block content
	block := findBlock(node)
	if block != nil {
		parseNode(block, source, filename, file, &suite)
	}

	addSuiteToTarget(suite, parentSuite, file)
}

func processTest(node *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite) {
	name := extractName(node, source)
	if name == "" {
		name = "(anonymous)"
	}

	// Check for skip in block
	status := domain.TestStatusActive
	block := findBlock(node)
	if block != nil && hasSkipCall(block, source) {
		status = domain.TestStatusSkipped
	}

	test := domain.Test{
		Name:     name,
		Status:   status,
		Location: parser.GetLocation(node, filename),
	}

	addTestToTarget(test, parentSuite, file)
}

func extractName(node *sitter.Node, source []byte) string {
	// Try argument_list first
	args := node.ChildByFieldName("arguments")
	if args != nil {
		return extractNameFromArgs(args, source)
	}

	// Look for direct string, symbol or identifier child
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case rubyast.NodeString:
			return extractStringContent(child, source)
		case rubyast.NodeSymbol, rubyast.NodeSimpleSymbol:
			return extractSymbolContent(child, source)
		case rubyast.NodeIdentifier:
			text := parser.GetNodeText(child, source)
			if text != funcDescribe && text != funcIt {
				return text
			}
		case "scope_resolution", "constant":
			return parser.GetNodeText(child, source)
		}
	}

	return ""
}

func extractNameFromArgs(args *sitter.Node, source []byte) string {
	for i := 0; i < int(args.ChildCount()); i++ {
		child := args.Child(i)
		switch child.Type() {
		case rubyast.NodeString:
			return extractStringContent(child, source)
		case rubyast.NodeSymbol, rubyast.NodeSimpleSymbol:
			return extractSymbolContent(child, source)
		case rubyast.NodeIdentifier:
			return parser.GetNodeText(child, source)
		case "scope_resolution", "constant":
			return parser.GetNodeText(child, source)
		}
	}
	return ""
}

func extractStringContent(node *sitter.Node, source []byte) string {
	return rubyast.ExtractStringContent(node, source)
}

func extractSymbolContent(node *sitter.Node, source []byte) string {
	return rubyast.ExtractSymbolContent(node, source)
}

func findBlock(node *sitter.Node) *sitter.Node {
	return rubyast.FindBlock(node)
}

func addTestToTarget(test domain.Test, parentSuite *domain.TestSuite, file *domain.TestFile) {
	rubyast.AddTestToTarget(test, parentSuite, file)
}

func addSuiteToTarget(suite domain.TestSuite, parentSuite *domain.TestSuite, file *domain.TestFile) {
	rubyast.AddSuiteToTarget(suite, parentSuite, file)
}
