package jstest

import (
	"context"
	"fmt"
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser"
)

// DetectLanguage determines the programming language based on file extension.
func DetectLanguage(filename string) domain.Language {
	ext := filepath.Ext(filename)
	switch ext {
	case ".js", ".jsx":
		return domain.LanguageJavaScript
	case ".tsx":
		return domain.LanguageTSX
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

// ParseCallbackBody parses the body of a callback function.
func ParseCallbackBody(callback *sitter.Node, source []byte, filename string, file *domain.TestFile, suite *domain.TestSuite) {
	body := callback.ChildByFieldName("body")
	if body != nil {
		ParseNode(body, source, filename, file, suite)
	}
}

// ProcessTest creates a test from a call expression.
func ProcessTest(callNode *sitter.Node, args *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus, modifier string) {
	name := ExtractTestName(args, source)
	if name == "" {
		return
	}

	test := domain.Test{
		Name:     name,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(callNode, filename),
	}

	AddTestToTarget(test, parentSuite, file)
}

// ProcessSuite creates a test suite from a call expression.
// Handles both regular suites with callbacks and pending suites without callbacks.
func ProcessSuite(callNode *sitter.Node, args *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus, modifier string) {
	name := ExtractTestName(args, source)
	if name == "" {
		return
	}

	suite := domain.TestSuite{
		Name:     name,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(callNode, filename),
	}

	if callback := FindCallback(args); callback != nil {
		ParseCallbackBody(callback, source, filename, file, &suite)
	}

	AddSuiteToTarget(suite, parentSuite, file)
}

// ProcessEachTests creates a single test from a .each() call.
// Per ADR-02, dynamic test patterns are counted as 1 test regardless of runtime count.
func ProcessEachTests(callNode *sitter.Node, _ []string, nameTemplate string, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus, modifier string) {
	if nameTemplate == "" {
		return
	}

	test := domain.Test{
		Name:     nameTemplate + DynamicCasesSuffix,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(callNode, filename),
	}

	AddTestToTarget(test, parentSuite, file)
}

// ProcessEachSuites creates a single suite from a describe.each() call.
// Per ADR-02, dynamic test patterns are counted as 1 test regardless of runtime count.
func ProcessEachSuites(callNode *sitter.Node, _ []string, nameTemplate string, callback *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus, modifier string) {
	if callback == nil {
		return
	}

	if nameTemplate == "" {
		return
	}

	suite := domain.TestSuite{
		Name:     nameTemplate + DynamicCasesSuffix,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(callNode, filename),
	}

	ParseCallbackBody(callback, source, filename, file, &suite)
	AddSuiteToTarget(suite, parentSuite, file)
}

// ProcessEachCall handles .each() call patterns for both describe and test/it.
func ProcessEachCall(outerCall, innerCall, outerArgs *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	innerFunc := innerCall.ChildByFieldName("function")
	innerArgs := innerCall.ChildByFieldName("arguments")

	if innerFunc == nil || innerArgs == nil {
		return
	}

	funcName, status, modifier := ParseFunctionName(innerFunc, source)
	if funcName == "" {
		return
	}

	testCases := ExtractEachTestCases(innerArgs, source)
	nameTemplate := ExtractTestName(outerArgs, source)
	callback := FindCallback(outerArgs)

	switch funcName {
	case FuncDescribe + "." + ModifierEach, FuncContext + "." + ModifierEach, FuncSuite + "." + ModifierEach:
		ProcessEachSuites(outerCall, testCases, nameTemplate, callback, source, filename, file, currentSuite, status, modifier)
	case FuncIt + "." + ModifierEach, FuncTest + "." + ModifierEach, FuncSpecify + "." + ModifierEach:
		ProcessEachTests(outerCall, testCases, nameTemplate, filename, file, currentSuite, status, modifier)
	}
}

// ProcessCallExpression processes a call expression node to extract test/suite definitions.
func ProcessCallExpression(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	processCallExpressionWithMode(node, source, filename, file, currentSuite, false)
}

func processCallExpressionWithMode(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite, isDynamic bool) {
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return
	}

	args := node.ChildByFieldName("arguments")
	if args == nil {
		return
	}

	if !isDynamic && funcNode.Type() == "call_expression" {
		ProcessEachCall(node, funcNode, args, source, filename, file, currentSuite)
		return
	}

	if callback := findArrayIteratorCallback(funcNode, args, source); callback != nil {
		parseDynamicCallback(callback, source, filename, file, currentSuite)
		return
	}

	funcName, status, modifier := ParseFunctionName(funcNode, source)
	if funcName == "" {
		return
	}

	switch funcName {
	case FuncBench:
		if !isDynamic {
			ProcessTest(node, args, source, filename, file, currentSuite, status, modifier)
		}
	case FuncDescribe, FuncContext, FuncSuite:
		processTestSuite(node, args, source, filename, file, currentSuite, status, modifier, isDynamic)
	case FuncIt, FuncTest, FuncSpecify:
		processTestCase(node, args, source, filename, file, currentSuite, status, modifier, isDynamic)
	default:
		// Traverse callbacks of unrecognized functions (e.g., describeMatrix, describeIf).
		// This allows detecting tests inside custom wrapper functions.
		if callback := FindLastCallback(args); callback != nil {
			ParseCallbackBody(callback, source, filename, file, currentSuite)
		}
	}
}

func processTestCase(callNode *sitter.Node, args *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus, modifier string, isDynamic bool) {
	name := ExtractTestName(args, source)
	if name == "" {
		return
	}

	if isDynamic {
		name += DynamicCasesSuffix
	}

	test := domain.Test{
		Name:     name,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(callNode, filename),
	}

	AddTestToTarget(test, parentSuite, file)
}

func processTestSuite(callNode *sitter.Node, args *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus, modifier string, isDynamic bool) {
	name := ExtractTestName(args, source)
	if name == "" {
		return
	}

	if isDynamic {
		name += DynamicCasesSuffix
	}

	suite := domain.TestSuite{
		Name:     name,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(callNode, filename),
	}

	if callback := FindCallback(args); callback != nil {
		ParseCallbackBody(callback, source, filename, file, &suite)
	}

	AddSuiteToTarget(suite, parentSuite, file)
}

// ParseNode recursively traverses the AST to find and process test definitions.
func ParseNode(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	parseNodeWithMode(node, source, filename, file, currentSuite, false)
}

func parseNodeWithMode(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite, isDynamic bool) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)

		switch child.Type() {
		case "expression_statement":
			if expr := parser.FindChildByType(child, "call_expression"); expr != nil {
				processCallExpressionWithMode(expr, source, filename, file, currentSuite, isDynamic)
			}
		case "variable_declaration", "lexical_declaration":
			processVariableDeclaration(child, source, filename, file, currentSuite, isDynamic)
		case "for_statement", "for_in_statement", "while_statement", "do_statement":
			parseLoopBody(child, source, filename, file, currentSuite)
		default:
			parseNodeWithMode(child, source, filename, file, currentSuite, isDynamic)
		}
	}
}

// processVariableDeclaration extracts and processes call expressions from variable declarations.
func processVariableDeclaration(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite, isDynamic bool) {
	for i := 0; i < int(node.ChildCount()); i++ {
		declarator := node.Child(i)
		if declarator == nil || declarator.Type() != "variable_declarator" {
			continue
		}

		valueNode := declarator.ChildByFieldName("value")
		if valueNode == nil {
			continue
		}

		callExpr := findInnerCallExpression(valueNode)
		if callExpr != nil {
			processCallExpressionWithMode(callExpr, source, filename, file, currentSuite, isDynamic)
		} else {
			parseNodeWithMode(valueNode, source, filename, file, currentSuite, isDynamic)
		}
	}
}

// findInnerCallExpression finds the innermost test call_expression in a node.
func findInnerCallExpression(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}

	if node.Type() != "call_expression" {
		return nil
	}

	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return node
	}

	if funcNode.Type() == "member_expression" {
		objectNode := funcNode.ChildByFieldName("object")
		if inner := findInnerCallExpression(objectNode); inner != nil {
			return inner
		}
	}

	return node
}

// parseLoopBody parses test definitions inside loops (for, while, do-while) as dynamic tests.
func parseLoopBody(loopNode *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	body := loopNode.ChildByFieldName("body")
	if body == nil {
		return
	}
	parseNodeWithMode(body, source, filename, file, currentSuite, true)
}

// findArrayIteratorCallback extracts callback from array iterator methods (forEach, map).
func findArrayIteratorCallback(funcNode, args *sitter.Node, source []byte) *sitter.Node {
	if funcNode.Type() != "member_expression" {
		return nil
	}

	prop := funcNode.ChildByFieldName("property")
	if prop == nil {
		return nil
	}

	propName := parser.GetNodeText(prop, source)

	switch propName {
	case "forEach", "map":
		return FindCallback(args)
	}

	return nil
}

// parseDynamicCallback parses test definitions inside dynamic contexts (forEach, map).
func parseDynamicCallback(callback *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	body := callback.ChildByFieldName("body")
	if body == nil {
		return
	}
	parseNodeWithMode(body, source, filename, file, currentSuite, true)
}

// Parse is the main entry point for parsing JavaScript/TypeScript test files.
func Parse(ctx context.Context, source []byte, filename string, framework string) (*domain.TestFile, error) {
	lang := DetectLanguage(filename)

	tree, err := parser.ParseWithPool(ctx, lang, source)
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
