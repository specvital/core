// Package rspec implements RSpec test framework support for Ruby test files.
package rspec

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

const frameworkName = "rspec"

func init() {
	framework.Register(NewDefinition())
}

func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      frameworkName,
		Languages: []domain.Language{domain.LanguageRuby},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher(
				"require 'rspec'",
				"require \"rspec\"",
				"require 'rspec/core'",
				"require \"rspec/core\"",
			),
			matchers.NewConfigMatcher(
				".rspec",
				"spec_helper.rb",
				"rails_helper.rb",
			),
			&RSpecFileMatcher{},
			&RSpecContentMatcher{},
		},
		ConfigParser: nil,
		Parser:       &RSpecParser{},
		Priority:     framework.PriorityGeneric,
	}
}

// RSpecFileMatcher matches *_spec.rb and spec/**/*.rb files.
type RSpecFileMatcher struct{}

func (m *RSpecFileMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileName {
		return framework.NoMatch()
	}

	filename := signal.Value
	base := filename
	if idx := strings.LastIndex(filename, "/"); idx >= 0 {
		base = filename[idx+1:]
	}

	if strings.HasSuffix(base, "_spec.rb") {
		return framework.PartialMatch(20, "RSpec file naming: *_spec.rb")
	}

	return framework.NoMatch()
}

// RSpecContentMatcher matches RSpec-specific patterns.
type RSpecContentMatcher struct{}

var rspecPatterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	// ReDoS-safe patterns: use \s+ instead of .* to avoid catastrophic backtracking
	{regexp.MustCompile(`RSpec\.describe\s*[\(\s]`), "RSpec.describe block"},
	{regexp.MustCompile(`\bdescribe\s+(?:'[^']*'|"[^"]*")\s+do\b`), "describe block"},
	{regexp.MustCompile(`\bcontext\s+(?:'[^']*'|"[^"]*")\s+do\b`), "context block"},
	{regexp.MustCompile(`\bit\s+(?:'[^']*'|"[^"]*")\s+do\b`), "it block"},
	{regexp.MustCompile(`\bspecify\s+(?:'[^']*'|"[^"]*")\s+do\b`), "specify block"},
	{regexp.MustCompile(`\bsubject\s*[\(\{]`), "subject definition"},
	{regexp.MustCompile(`\blet\s*[\(\:]`), "let definition"},
	{regexp.MustCompile(`\bbefore\s*[\(\{:]`), "before hook"},
	{regexp.MustCompile(`\bexpect\s*\(`), "expect assertion"},
}

func (m *RSpecContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileContent {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		content = []byte(signal.Value)
	}

	for _, p := range rspecPatterns {
		if p.pattern.Match(content) {
			return framework.PartialMatch(40, "Found RSpec pattern: "+p.desc)
		}
	}

	return framework.NoMatch()
}

// RSpecParser extracts test definitions from Ruby RSpec files.
type RSpecParser struct{}

func (p *RSpecParser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	tree, err := parser.ParseWithPool(ctx, domain.LanguageRuby, source)
	if err != nil {
		return nil, fmt.Errorf("rspec parser: failed to parse %s: %w", filename, err)
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

// RSpec keywords for test definitions.
const (
	funcDescribe = "describe"
	funcContext  = "context"
	funcIt       = "it"
	funcSpecify  = "specify"
	funcExample  = "example"

	// Skip modifiers
	modifierX       = "x"
	modifierSkip    = "skip"
	modifierPending = "pending"
)

func parseNode(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		processNode(child, source, filename, file, currentSuite)
	}
}

func processNode(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	switch node.Type() {
	case rubyast.NodeCall, rubyast.NodeMethodCall:
		processCallExpression(node, source, filename, file, currentSuite)
	default:
		parseNode(node, source, filename, file, currentSuite)
	}
}

func processCallExpression(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	funcName, status := parseFunctionName(node, source)
	if funcName == "" {
		return
	}

	switch funcName {
	case funcDescribe, funcContext:
		processSuite(node, source, filename, file, currentSuite, status)
	case funcIt, funcSpecify, funcExample:
		processTest(node, source, filename, file, currentSuite, status)
	case modifierSkip, modifierPending:
		processPendingBlock(node, source, filename, file, currentSuite)
	}
}

func parseFunctionName(node *sitter.Node, source []byte) (string, domain.TestStatus) {
	// Handle method call: receiver.method
	methodNode := node.ChildByFieldName("method")
	if methodNode != nil {
		methodName := parser.GetNodeText(methodNode, source)

		// Check for RSpec.describe pattern
		receiver := node.ChildByFieldName("receiver")
		if receiver != nil {
			receiverName := parser.GetNodeText(receiver, source)
			if receiverName == "RSpec" && methodName == funcDescribe {
				return funcDescribe, ""
			}
		}

		// Check for skip/pending modifiers
		status := getStatusFromMethod(methodName)
		baseName := getBaseMethod(methodName)
		if baseName != "" {
			return baseName, status
		}

		return methodName, status
	}

	// Handle simple call: method_name
	nameNode := parser.FindChildByType(node, rubyast.NodeIdentifier)
	if nameNode != nil {
		name := parser.GetNodeText(nameNode, source)
		status := getStatusFromMethod(name)
		baseName := getBaseMethod(name)
		if baseName != "" {
			return baseName, status
		}
		return name, status
	}

	return "", ""
}

func getStatusFromMethod(name string) domain.TestStatus {
	// Handle x-prefixed (xdescribe, xit, etc.)
	if strings.HasPrefix(name, modifierX) {
		return domain.TestStatusSkipped
	}
	// skip is a status indicator
	if name == modifierSkip {
		return domain.TestStatusSkipped
	}
	// RSpec pending runs test but expects failure (xfail semantics)
	if name == modifierPending {
		return domain.TestStatusXfail
	}
	return domain.TestStatusActive
}

func getBaseMethod(name string) string {
	// Map prefixed methods to base methods
	switch name {
	case "xdescribe", "fdescribe":
		return funcDescribe
	case "xcontext", "fcontext":
		return funcContext
	case "xit", "fit":
		return funcIt
	case "xspecify", "fspecify":
		return funcSpecify
	case "xexample", "fexample":
		return funcExample
	}
	return ""
}

func processSuite(node *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus) {
	name := extractName(node, source)
	if name == "" {
		return
	}

	suite := domain.TestSuite{
		Name:     name,
		Status:   status,
		Location: parser.GetLocation(node, filename),
	}

	// Parse the block content
	block := findBlock(node)
	if block != nil {
		parseNode(block, source, filename, file, &suite)
	}

	addSuiteToTarget(suite, parentSuite, file)
}

func processTest(node *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus) {
	name := extractName(node, source)
	if name == "" {
		// Handle pending tests without description: it { ... } or specify { ... }
		name = "(anonymous)"
	}

	test := domain.Test{
		Name:     name,
		Status:   status,
		Location: parser.GetLocation(node, filename),
	}

	addTestToTarget(test, parentSuite, file)
}

func processPendingBlock(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	// Handle skip/pending with string: skip "reason" or pending "reason"
	name := extractName(node, source)
	if name == "" {
		return
	}

	// Check if there's a block attached
	block := findBlock(node)
	if block != nil {
		// Create a skipped suite with the content
		suite := domain.TestSuite{
			Name:     name,
			Status:   domain.TestStatusSkipped,
			Location: parser.GetLocation(node, filename),
		}
		parseNode(block, source, filename, file, &suite)
		addSuiteToTarget(suite, currentSuite, file)
	} else {
		// Just a pending marker, create a skipped test
		test := domain.Test{
			Name:     name,
			Status:   domain.TestStatusSkipped,
			Location: parser.GetLocation(node, filename),
		}
		addTestToTarget(test, currentSuite, file)
	}
}

func extractName(node *sitter.Node, source []byte) string {
	// Try argument_list first
	args := node.ChildByFieldName("arguments")
	if args != nil {
		return extractNameFromArgs(args, source)
	}

	// Look for direct string or symbol child after method name
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case rubyast.NodeString:
			return extractStringContent(child, source)
		case rubyast.NodeSymbol, rubyast.NodeSimpleSymbol:
			return extractSymbolContent(child, source)
		case rubyast.NodeIdentifier:
			// Class/module name for describe
			text := parser.GetNodeText(child, source)
			if text != funcDescribe && text != funcContext && text != funcIt && text != funcSpecify && text != funcExample {
				return text
			}
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
			// Class/module name reference
			return parser.GetNodeText(child, source)
		case "scope_resolution", "constant":
			// Handle MyClass::MyModule pattern
			return parser.GetNodeText(child, source)
		}
	}
	return ""
}

func extractStringContent(node *sitter.Node, source []byte) string {
	text := parser.GetNodeText(node, source)
	// Remove surrounding quotes
	if len(text) >= 2 {
		if (text[0] == '"' && text[len(text)-1] == '"') ||
			(text[0] == '\'' && text[len(text)-1] == '\'') {
			return text[1 : len(text)-1]
		}
	}
	return text
}

func extractSymbolContent(node *sitter.Node, source []byte) string {
	text := parser.GetNodeText(node, source)
	// Remove leading colon from symbol
	if len(text) > 0 && text[0] == ':' {
		return text[1:]
	}
	return text
}

func findBlock(node *sitter.Node) *sitter.Node {
	// Look for block or do_block child
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == rubyast.NodeBlock || child.Type() == rubyast.NodeDoBlock {
			return child
		}
	}
	return nil
}

func addTestToTarget(test domain.Test, parentSuite *domain.TestSuite, file *domain.TestFile) {
	if parentSuite != nil {
		parentSuite.Tests = append(parentSuite.Tests, test)
	} else {
		file.Tests = append(file.Tests, test)
	}
}

func addSuiteToTarget(suite domain.TestSuite, parentSuite *domain.TestSuite, file *domain.TestFile) {
	if parentSuite != nil {
		parentSuite.Suites = append(parentSuite.Suites, suite)
	} else {
		file.Suites = append(file.Suites, suite)
	}
}
