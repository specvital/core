// Package gtest implements Google Test framework support for C++ test files.
package gtest

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser"
	"github.com/specvital/core/pkg/parser/framework"
	"github.com/specvital/core/pkg/parser/framework/matchers"
)

const frameworkName = "gtest"

// Tree-sitter node types for C++
const (
	nodeFunctionDefinition = "function_definition"
	nodeFunctionDeclarator = "function_declarator"
	nodeIdentifier         = "identifier"
	nodeParameterList      = "parameter_list"
	nodeParameterDecl      = "parameter_declaration"
	nodeTypeIdentifier     = "type_identifier"
)

// Google Test macros
var gtestMacros = map[string]bool{
	"TEST":         true,
	"TEST_F":       true,
	"TEST_P":       true,
	"TYPED_TEST":   true,
	"TYPED_TEST_P": true,
}

func init() {
	framework.Register(NewDefinition())
}

func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      frameworkName,
		Languages: []domain.Language{domain.LanguageCpp},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher(
				"gtest/gtest.h",
				"#include <gtest/gtest.h>",
			),
			&GTestFileMatcher{},
			&GTestContentMatcher{},
		},
		ConfigParser: nil,
		Parser:       &GTestParser{},
		Priority:     framework.PriorityGeneric,
	}
}

// GTestFileMatcher matches C++ test file naming conventions used by Google Test.
// It recognizes patterns like *_test.cc, *_unittest.cpp, *Test.cc, and files in test directories.
type GTestFileMatcher struct{}

func (m *GTestFileMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileName {
		return framework.NoMatch()
	}

	filename := signal.Value

	// Common gtest file patterns
	if strings.HasSuffix(filename, "_test.cc") ||
		strings.HasSuffix(filename, "_test.cpp") ||
		strings.HasSuffix(filename, "_test.cxx") ||
		strings.HasSuffix(filename, "_unittest.cc") ||
		strings.HasSuffix(filename, "_unittest.cpp") ||
		strings.HasSuffix(filename, "_unittest.cxx") ||
		strings.HasSuffix(filename, "Test.cpp") ||
		strings.HasSuffix(filename, "Test.cc") ||
		strings.HasSuffix(filename, "Test.cxx") {
		return framework.PartialMatch(20, "Google Test file naming convention")
	}

	// Test directory patterns
	if strings.HasSuffix(filename, ".cc") ||
		strings.HasSuffix(filename, ".cpp") ||
		strings.HasSuffix(filename, ".cxx") {
		if strings.Contains(filename, "/test/") || strings.Contains(filename, "/tests/") {
			return framework.PartialMatch(15, "C++ file in test directory")
		}
	}

	return framework.NoMatch()
}

// GTestContentMatcher matches Google Test-specific patterns in file content.
// It detects gtest includes, TEST/TEST_F/TEST_P macros, and ::testing::Test base classes.
type GTestContentMatcher struct{}

var gtestPatterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	{regexp.MustCompile(`#include\s*<gtest/gtest\.h>`), "#include <gtest/gtest.h>"},
	{regexp.MustCompile(`#include\s*"gtest/gtest\.h"`), "#include \"gtest/gtest.h\""},
	{regexp.MustCompile(`\bTEST\s*\(`), "TEST() macro"},
	{regexp.MustCompile(`\bTEST_F\s*\(`), "TEST_F() macro"},
	{regexp.MustCompile(`\bTEST_P\s*\(`), "TEST_P() macro"},
	{regexp.MustCompile(`\bTYPED_TEST\s*\(`), "TYPED_TEST() macro"},
	{regexp.MustCompile(`\bTYPED_TEST_P\s*\(`), "TYPED_TEST_P() macro"},
	{regexp.MustCompile(`\bINSTANTIATE_TEST_SUITE_P\s*\(`), "INSTANTIATE_TEST_SUITE_P() macro"},
	{regexp.MustCompile(`::testing::Test`), "::testing::Test base class"},
}

func (m *GTestContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileContent {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		content = []byte(signal.Value)
	}

	for _, p := range gtestPatterns {
		if p.pattern.Match(content) {
			return framework.PartialMatch(40, "Found Google Test pattern: "+p.desc)
		}
	}

	return framework.NoMatch()
}

// GTestParser extracts test definitions from C++ Google Test files.
type GTestParser struct{}

func (p *GTestParser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	tree, err := parser.ParseWithPool(ctx, domain.LanguageCpp, source)
	if err != nil {
		return nil, fmt.Errorf("gtest parser: failed to parse %s: %w", filename, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	file := &domain.TestFile{
		Path:      filename,
		Language:  domain.LanguageCpp,
		Framework: frameworkName,
	}

	// Group tests by suite name
	suiteMap := make(map[string]*domain.TestSuite)

	parser.WalkTree(root, func(node *sitter.Node) bool {
		if node.Type() != nodeFunctionDefinition {
			return true
		}

		test, suiteName := parseGTestMacro(node, source, filename)
		if test == nil {
			return true // Continue traversing to find tests
		}

		// Get or create suite
		suite, exists := suiteMap[suiteName]
		if !exists {
			suiteStatus, suiteModifier := getSuiteStatus(suiteName)
			suite = &domain.TestSuite{
				Location: parser.GetLocation(node, filename),
				Modifier: suiteModifier,
				Name:     suiteName,
				Status:   suiteStatus,
			}
			suiteMap[suiteName] = suite
		}

		suite.Tests = append(suite.Tests, *test)
		return false
	})

	// Collect suites in deterministic order (sorted by name)
	suiteNames := make([]string, 0, len(suiteMap))
	for name := range suiteMap {
		suiteNames = append(suiteNames, name)
	}
	sort.Strings(suiteNames)
	for _, name := range suiteNames {
		file.Suites = append(file.Suites, *suiteMap[name])
	}

	return file, nil
}

// parseGTestMacro parses TEST(), TEST_F(), TEST_P(), TYPED_TEST(), or TYPED_TEST_P() macro.
// Returns test and suite name, or nil if not a gtest macro.
func parseGTestMacro(node *sitter.Node, source []byte, filename string) (*domain.Test, string) {
	declarator := parser.FindChildByType(node, nodeFunctionDeclarator)
	if declarator == nil {
		return nil, ""
	}

	// Get macro name (TEST, TEST_F, TEST_P)
	macroIdent := parser.FindChildByType(declarator, nodeIdentifier)
	if macroIdent == nil {
		return nil, ""
	}
	macroName := parser.GetNodeText(macroIdent, source)
	if !gtestMacros[macroName] {
		return nil, ""
	}

	// Get parameter list (SuiteName, TestName)
	paramList := parser.FindChildByType(declarator, nodeParameterList)
	if paramList == nil {
		return nil, ""
	}

	params := extractParameters(paramList, source)
	if len(params) < 2 {
		return nil, ""
	}

	suiteName := params[0]
	testName := params[1]

	// Determine test status based on DISABLED_ prefix
	status, modifier := getTestStatus(suiteName, testName)

	return &domain.Test{
		Location: parser.GetLocation(node, filename),
		Modifier: modifier,
		Name:     testName,
		Status:   status,
	}, suiteName
}

// extractParameters extracts parameter names from parameter_list.
func extractParameters(paramList *sitter.Node, source []byte) []string {
	var params []string
	for i := 0; i < int(paramList.ChildCount()); i++ {
		child := paramList.Child(i)
		if child.Type() == nodeParameterDecl {
			// Parameter declaration contains type_identifier as the "type"
			typeIdent := parser.FindChildByType(child, nodeTypeIdentifier)
			if typeIdent != nil {
				params = append(params, parser.GetNodeText(typeIdent, source))
			}
		}
	}
	return params
}

const disabledPrefix = "DISABLED_"

// checkDisabledStatus returns skipped status if name has DISABLED_ prefix.
func checkDisabledStatus(name string) (domain.TestStatus, string) {
	if strings.HasPrefix(name, disabledPrefix) {
		return domain.TestStatusSkipped, disabledPrefix
	}
	return domain.TestStatusActive, ""
}

// getTestStatus determines test status based on DISABLED_ prefix in test or suite name.
func getTestStatus(suiteName, testName string) (domain.TestStatus, string) {
	// Test-level DISABLED_ takes precedence
	if status, modifier := checkDisabledStatus(testName); status == domain.TestStatusSkipped {
		return status, modifier
	}
	// Suite-level DISABLED_ applies to all tests
	return checkDisabledStatus(suiteName)
}

// getSuiteStatus determines suite status based on DISABLED_ prefix.
func getSuiteStatus(suiteName string) (domain.TestStatus, string) {
	return checkDisabledStatus(suiteName)
}
