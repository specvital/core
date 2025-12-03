package jstest

import (
	"context"
	"fmt"
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/domain"
	"github.com/specvital/core/parser"
)

// DetectLanguage determines the programming language based on file extension.
func DetectLanguage(filename string) domain.Language {
	ext := filepath.Ext(filename)
	switch ext {
	case ".js", ".jsx":
		return domain.LanguageJavaScript
	default:
		return domain.LanguageTypeScript
	}
}

// AddTestToTarget adds a test to the appropriate parent (suite or file).
func AddTestToTarget(test domain.Test, parentSuite *domain.TestSuite, file *domain.TestFile) {
	if parentSuite != nil {
		parentSuite.Tests = append(parentSuite.Tests, test)
	} else {
		file.Tests = append(file.Tests, test)
	}
}

// AddSuiteToTarget adds a suite to the appropriate parent (suite or file).
func AddSuiteToTarget(suite domain.TestSuite, parentSuite *domain.TestSuite, file *domain.TestFile) {
	if parentSuite != nil {
		parentSuite.Suites = append(parentSuite.Suites, suite)
	} else {
		file.Suites = append(file.Suites, suite)
	}
}

// ResolveEachNames generates test names from a template and test cases.
func ResolveEachNames(template string, testCases []string) []string {
	if len(testCases) == 0 {
		return []string{template + DynamicCasesSuffix}
	}

	names := make([]string, len(testCases))
	for i, testCase := range testCases {
		names[i] = FormatEachName(template, testCase)
	}

	return names
}

// ParseCallbackBody parses the body of a callback function.
func ParseCallbackBody(callback *sitter.Node, source []byte, filename string, file *domain.TestFile, suite *domain.TestSuite) {
	body := callback.ChildByFieldName("body")
	if body != nil {
		ParseNode(body, source, filename, file, suite)
	}
}

// ProcessTest creates a test from a call expression.
func ProcessTest(callNode *sitter.Node, args *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus) {
	name := ExtractTestName(args, source)
	if name == "" {
		return
	}

	test := domain.Test{
		Name:     name,
		Status:   status,
		Location: parser.GetLocation(callNode, filename),
	}

	AddTestToTarget(test, parentSuite, file)
}

// ProcessSuite creates a test suite from a call expression.
// Handles both regular suites with callbacks and pending suites without callbacks.
func ProcessSuite(callNode *sitter.Node, args *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus) {
	name := ExtractTestName(args, source)
	if name == "" {
		return
	}

	suite := domain.TestSuite{
		Name:     name,
		Status:   status,
		Location: parser.GetLocation(callNode, filename),
	}

	if callback := FindCallback(args); callback != nil {
		ParseCallbackBody(callback, source, filename, file, &suite)
	}

	AddSuiteToTarget(suite, parentSuite, file)
}

// ProcessEachTests creates multiple tests from a .each() call.
func ProcessEachTests(callNode *sitter.Node, testCases []string, nameTemplate string, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus) {
	names := ResolveEachNames(nameTemplate, testCases)

	for _, name := range names {
		test := domain.Test{
			Name:     name,
			Status:   status,
			Location: parser.GetLocation(callNode, filename),
		}

		AddTestToTarget(test, parentSuite, file)
	}
}

// ProcessEachSuites creates multiple suites from a describe.each() call.
func ProcessEachSuites(callNode *sitter.Node, testCases []string, nameTemplate string, callback *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus) {
	if callback == nil {
		return
	}

	names := ResolveEachNames(nameTemplate, testCases)

	for _, name := range names {
		suite := domain.TestSuite{
			Name:     name,
			Status:   status,
			Location: parser.GetLocation(callNode, filename),
		}

		ParseCallbackBody(callback, source, filename, file, &suite)
		AddSuiteToTarget(suite, parentSuite, file)
	}
}

// ProcessEachCall handles .each() call patterns for both describe and test/it.
func ProcessEachCall(outerCall, innerCall, outerArgs *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	innerFunc := innerCall.ChildByFieldName("function")
	innerArgs := innerCall.ChildByFieldName("arguments")

	if innerFunc == nil || innerArgs == nil {
		return
	}

	funcName, status := ParseFunctionName(innerFunc, source)
	if funcName == "" {
		return
	}

	testCases := ExtractEachTestCases(innerArgs, source)
	nameTemplate := ExtractTestName(outerArgs, source)
	callback := FindCallback(outerArgs)

	switch funcName {
	case FuncDescribe + "." + ModifierEach:
		ProcessEachSuites(outerCall, testCases, nameTemplate, callback, source, filename, file, currentSuite, status)
	case FuncIt + "." + ModifierEach, FuncTest + "." + ModifierEach:
		ProcessEachTests(outerCall, testCases, nameTemplate, filename, file, currentSuite, status)
	}
}

// ProcessCallExpression processes a call expression node to extract test/suite definitions.
func ProcessCallExpression(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return
	}

	args := node.ChildByFieldName("arguments")
	if args == nil {
		return
	}

	if funcNode.Type() == "call_expression" {
		ProcessEachCall(node, funcNode, args, source, filename, file, currentSuite)
		return
	}

	funcName, status := ParseFunctionName(funcNode, source)
	if funcName == "" {
		return
	}

	switch funcName {
	case FuncDescribe:
		ProcessSuite(node, args, source, filename, file, currentSuite, status)
	case FuncIt, FuncTest:
		ProcessTest(node, args, source, filename, file, currentSuite, status)
	}
}

// ParseNode recursively traverses the AST to find and process test definitions.
func ParseNode(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)

		switch child.Type() {
		case "expression_statement":
			if expr := parser.FindChildByType(child, "call_expression"); expr != nil {
				ProcessCallExpression(expr, source, filename, file, currentSuite)
			}
		default:
			ParseNode(child, source, filename, file, currentSuite)
		}
	}
}

// Parse is the main entry point for parsing JavaScript/TypeScript test files.
func Parse(ctx context.Context, source []byte, filename string, framework string) (*domain.TestFile, error) {
	lang := DetectLanguage(filename)

	p := parser.NewTSParser(lang)

	tree, err := p.Parse(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
	}
	defer tree.Close()
	root := tree.RootNode()

	testFile := &domain.TestFile{
		Path:      filename,
		Language:  lang,
		Framework: framework,
	}

	ParseNode(root, source, filename, testFile, nil)

	return testFile, nil
}
