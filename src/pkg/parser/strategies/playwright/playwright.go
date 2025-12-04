package playwright

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/src/pkg/domain"
	"github.com/specvital/core/src/pkg/parser"
	"github.com/specvital/core/src/pkg/parser/strategies"
	"github.com/specvital/core/src/pkg/parser/strategies/shared/jstest"
)

const (
	frameworkName = "playwright"

	// Function names for Playwright test API
	funcTest         = "test"
	funcTestDescribe = "test.describe"

	// Playwright-specific modifier
	modifierFixme = "fixme"
)

// playwrightImportPattern matches import/require statements for '@playwright/test'.
// Limitations:
// - Does not match dynamic imports: import('@playwright/test')
// - Does not match re-exports: export { test } from '@playwright/test'
var playwrightImportPattern = regexp.MustCompile(`(?:import\s+.*\s+from|require\()\s*['"]@playwright/test['"]`)

type Strategy struct{}

func NewStrategy() *Strategy {
	return &Strategy{}
}

func RegisterDefault() {
	strategies.Register(NewStrategy())
}

func (s *Strategy) Name() string {
	return frameworkName
}

func (s *Strategy) Priority() int {
	return strategies.DefaultPriority
}

func (s *Strategy) Languages() []domain.Language {
	return []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript}
}

func (s *Strategy) CanHandle(filename string, content []byte) bool {
	// Playwright E2E tests: only .ts and .js files
	ext := filepath.Ext(filename)
	if ext != ".ts" && ext != ".js" {
		return false
	}

	if !jstest.IsTestFile(filename) {
		return false
	}

	return hasPlaywrightImport(content)
}

func (s *Strategy) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	lang := jstest.DetectLanguage(filename)

	tree, err := parser.ParseWithPool(ctx, lang, source)
	if err != nil {
		return nil, fmt.Errorf("playwright parser: failed to parse %s: %w", filename, err)
	}
	defer tree.Close()
	root := tree.RootNode()

	testFile := &domain.TestFile{
		Path:      filename,
		Language:  lang,
		Framework: frameworkName,
	}

	parseNode(root, source, filename, testFile, nil)

	return testFile, nil
}

// Helper functions (alphabetically ordered)

func hasPlaywrightImport(content []byte) bool {
	return playwrightImportPattern.Match(content)
}

func parseFunctionName(node *sitter.Node, source []byte) (string, domain.TestStatus) {
	switch node.Type() {
	case "identifier":
		name := parser.GetNodeText(node, source)
		if name == funcTest {
			return funcTest, domain.TestStatusPending
		}
		return "", domain.TestStatusPending
	case "member_expression":
		return parseMemberExpression(node, source)
	default:
		return "", domain.TestStatusPending
	}
}

func parseMemberExpression(node *sitter.Node, source []byte) (string, domain.TestStatus) {
	obj := node.ChildByFieldName("object")
	prop := node.ChildByFieldName("property")

	if obj == nil || prop == nil {
		return "", domain.TestStatusPending
	}

	switch obj.Type() {
	case "identifier":
		return parseSimpleMemberExpression(obj, prop, source)
	case "member_expression":
		return parseNestedMemberExpression(obj, prop, source)
	}

	return "", domain.TestStatusPending
}

func parseModifierStatus(modifier string) domain.TestStatus {
	switch modifier {
	case jstest.ModifierSkip:
		return domain.TestStatusSkipped
	case jstest.ModifierOnly:
		return domain.TestStatusOnly
	case modifierFixme:
		return domain.TestStatusFixme
	default:
		return domain.TestStatusPending
	}
}

func parseNestedMemberExpression(obj *sitter.Node, prop *sitter.Node, source []byte) (string, domain.TestStatus) {
	innerObj := obj.ChildByFieldName("object")
	innerProp := obj.ChildByFieldName("property")

	if innerObj == nil || innerProp == nil {
		return "", domain.TestStatusPending
	}

	objName := parser.GetNodeText(innerObj, source)
	if objName != funcTest {
		return "", domain.TestStatusPending
	}

	middleProp := parser.GetNodeText(innerProp, source)
	if middleProp == "describe" {
		outerProp := parser.GetNodeText(prop, source)
		return funcTestDescribe, parseModifierStatus(outerProp)
	}

	return "", domain.TestStatusPending
}

func parseNode(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)

		switch child.Type() {
		case "expression_statement":
			if expr := parser.FindChildByType(child, "call_expression"); expr != nil {
				processCallExpression(expr, source, filename, file, currentSuite)
			}
		default:
			parseNode(child, source, filename, file, currentSuite)
		}
	}
}

func parseSimpleMemberExpression(obj *sitter.Node, prop *sitter.Node, source []byte) (string, domain.TestStatus) {
	objName := parser.GetNodeText(obj, source)
	if objName != funcTest {
		return "", domain.TestStatusPending
	}

	propName := parser.GetNodeText(prop, source)
	switch propName {
	case "describe":
		return funcTestDescribe, domain.TestStatusPending
	case jstest.ModifierSkip:
		return funcTest, domain.TestStatusSkipped
	case jstest.ModifierOnly:
		return funcTest, domain.TestStatusOnly
	case modifierFixme:
		return funcTest, domain.TestStatusFixme
	}

	return "", domain.TestStatusPending
}

func processCallExpression(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return
	}

	args := node.ChildByFieldName("arguments")
	if args == nil {
		return
	}

	funcName, status := parseFunctionName(funcNode, source)
	if funcName == "" {
		return
	}

	switch funcName {
	case funcTestDescribe:
		processSuite(node, args, source, filename, file, currentSuite, status)
	case funcTest:
		processTest(node, args, source, filename, file, currentSuite, status)
	}
}

func processSuite(callNode *sitter.Node, args *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus) {
	name := jstest.ExtractTestName(args, source)
	if name == "" {
		return
	}

	suite := domain.TestSuite{
		Name:     name,
		Status:   status,
		Location: parser.GetLocation(callNode, filename),
	}

	if callback := jstest.FindCallback(args); callback != nil {
		body := callback.ChildByFieldName("body")
		if body != nil {
			parseNode(body, source, filename, file, &suite)
		}
	}

	jstest.AddSuiteToTarget(suite, parentSuite, file)
}

func processTest(callNode *sitter.Node, args *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus) {
	name := jstest.ExtractTestName(args, source)
	if name == "" {
		return
	}

	test := domain.Test{
		Name:     name,
		Status:   status,
		Location: parser.GetLocation(callNode, filename),
	}

	jstest.AddTestToTarget(test, parentSuite, file)
}
