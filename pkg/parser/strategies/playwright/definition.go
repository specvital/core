package playwright

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser"
	"github.com/specvital/core/pkg/parser/framework"
	"github.com/specvital/core/pkg/parser/framework/matchers"
	"github.com/specvital/core/pkg/parser/strategies/shared/jstest"
)

const (
	frameworkName    = "playwright"
	funcTest         = "test"
	funcTestDescribe = "test.describe"
	modifierFixme    = "fixme"
)

func init() {
	framework.Register(NewDefinition())
}

func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      frameworkName,
		Languages: []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("@playwright/test", "@playwright/test/"),
			matchers.NewConfigMatcher(
				"playwright.config.js",
				"playwright.config.ts",
				"playwright.config.mjs",
				"playwright.config.mts",
			),
		},
		ConfigParser: &PlaywrightConfigParser{},
		Parser:       &PlaywrightParser{},
		Priority:     framework.PriorityE2E,
	}
}

type PlaywrightConfigParser struct{}

func (p *PlaywrightConfigParser) Parse(ctx context.Context, configPath string, content []byte) (*framework.ConfigScope, error) {
	scope := framework.NewConfigScope(configPath, "")
	scope.Framework = frameworkName
	scope.GlobalsMode = false // Playwright always requires explicit imports
	return scope, nil
}

type PlaywrightParser struct{}

func (p *PlaywrightParser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
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

func parseFunctionName(node *sitter.Node, source []byte) (string, domain.TestStatus) {
	switch node.Type() {
	case "identifier":
		name := parser.GetNodeText(node, source)
		if name == funcTest {
			return funcTest, domain.TestStatusActive
		}
		return "", domain.TestStatusActive
	case "member_expression":
		return parseMemberExpression(node, source)
	default:
		return "", domain.TestStatusActive
	}
}

func parseMemberExpression(node *sitter.Node, source []byte) (string, domain.TestStatus) {
	obj := node.ChildByFieldName("object")
	prop := node.ChildByFieldName("property")

	if obj == nil || prop == nil {
		return "", domain.TestStatusActive
	}

	switch obj.Type() {
	case "identifier":
		return parseSimpleMemberExpression(obj, prop, source)
	case "member_expression":
		return parseNestedMemberExpression(obj, prop, source)
	}

	return "", domain.TestStatusActive
}

func parseModifierStatus(modifier string) domain.TestStatus {
	switch modifier {
	case jstest.ModifierSkip, modifierFixme:
		return domain.TestStatusSkipped
	case jstest.ModifierOnly:
		return domain.TestStatusFocused
	default:
		return domain.TestStatusActive
	}
}

func parseNestedMemberExpression(obj *sitter.Node, prop *sitter.Node, source []byte) (string, domain.TestStatus) {
	innerObj := obj.ChildByFieldName("object")
	innerProp := obj.ChildByFieldName("property")

	if innerObj == nil || innerProp == nil {
		return "", domain.TestStatusActive
	}

	objName := parser.GetNodeText(innerObj, source)
	if objName != funcTest {
		return "", domain.TestStatusActive
	}

	middleProp := parser.GetNodeText(innerProp, source)
	if middleProp == "describe" {
		outerProp := parser.GetNodeText(prop, source)
		return funcTestDescribe, parseModifierStatus(outerProp)
	}

	return "", domain.TestStatusActive
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
		return "", domain.TestStatusActive
	}

	propName := parser.GetNodeText(prop, source)
	switch propName {
	case "describe":
		return funcTestDescribe, domain.TestStatusActive
	case jstest.ModifierSkip, modifierFixme:
		return funcTest, domain.TestStatusSkipped
	case jstest.ModifierOnly:
		return funcTest, domain.TestStatusFocused
	}

	return "", domain.TestStatusActive
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
