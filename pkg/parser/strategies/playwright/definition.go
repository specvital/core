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

	playwrightImportPath = "@playwright/test"
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

	testAliases := extractTestAliases(root, source)
	parseNode(root, source, filename, testFile, nil, testAliases)

	return testFile, nil
}

func extractTestAliases(root *sitter.Node, source []byte) map[string]bool {
	aliases := map[string]bool{funcTest: true}

	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		if child == nil || child.Type() != "import_statement" {
			continue
		}

		if !isPlaywrightImport(child, source) {
			continue
		}

		extractAliasesFromImport(child, source, aliases)
	}

	return aliases
}

func isPlaywrightImport(node *sitter.Node, source []byte) bool {
	sourceNode := node.ChildByFieldName("source")
	if sourceNode == nil {
		return false
	}

	importPath := jstest.UnquoteString(parser.GetNodeText(sourceNode, source))
	return importPath == playwrightImportPath
}

func extractAliasesFromImport(importNode *sitter.Node, source []byte, aliases map[string]bool) {
	for i := 0; i < int(importNode.ChildCount()); i++ {
		child := importNode.Child(i)
		if child == nil || child.Type() != "import_clause" {
			continue
		}

		processImportClause(child, source, aliases)
	}
}

func processImportClause(clause *sitter.Node, source []byte, aliases map[string]bool) {
	for i := 0; i < int(clause.ChildCount()); i++ {
		child := clause.Child(i)
		if child == nil || child.Type() != "named_imports" {
			continue
		}

		processNamedImports(child, source, aliases)
	}
}

func processNamedImports(namedImports *sitter.Node, source []byte, aliases map[string]bool) {
	for i := 0; i < int(namedImports.ChildCount()); i++ {
		specifier := namedImports.Child(i)
		if specifier == nil || specifier.Type() != "import_specifier" {
			continue
		}

		processImportSpecifier(specifier, source, aliases)
	}
}

func processImportSpecifier(specifier *sitter.Node, source []byte, aliases map[string]bool) {
	nameNode := specifier.ChildByFieldName("name")
	aliasNode := specifier.ChildByFieldName("alias")

	if nameNode == nil {
		return
	}

	originalName := parser.GetNodeText(nameNode, source)
	if originalName != funcTest {
		return
	}

	if aliasNode != nil {
		aliasName := parser.GetNodeText(aliasNode, source)
		aliases[aliasName] = true
	}
}

func parseFunctionName(node *sitter.Node, source []byte, testAliases map[string]bool) (string, domain.TestStatus, string) {
	switch node.Type() {
	case "identifier":
		name := parser.GetNodeText(node, source)
		if testAliases[name] {
			return funcTest, domain.TestStatusActive, ""
		}
		return "", domain.TestStatusActive, ""
	case "member_expression":
		return parseMemberExpression(node, source, testAliases)
	default:
		return "", domain.TestStatusActive, ""
	}
}

func parseMemberExpression(node *sitter.Node, source []byte, testAliases map[string]bool) (string, domain.TestStatus, string) {
	obj := node.ChildByFieldName("object")
	prop := node.ChildByFieldName("property")

	if obj == nil || prop == nil {
		return "", domain.TestStatusActive, ""
	}

	switch obj.Type() {
	case "identifier":
		return parseSimpleMemberExpression(obj, prop, source, testAliases)
	case "member_expression":
		return parseNestedMemberExpression(obj, prop, source, testAliases)
	}

	return "", domain.TestStatusActive, ""
}

func parseModifierStatusAndModifier(modifier string) (domain.TestStatus, string) {
	switch modifier {
	case jstest.ModifierSkip:
		return domain.TestStatusSkipped, jstest.ModifierSkip
	case modifierFixme:
		return domain.TestStatusSkipped, modifierFixme
	case jstest.ModifierOnly:
		return domain.TestStatusFocused, jstest.ModifierOnly
	default:
		return domain.TestStatusActive, ""
	}
}

func parseNestedMemberExpression(obj *sitter.Node, prop *sitter.Node, source []byte, testAliases map[string]bool) (string, domain.TestStatus, string) {
	innerObj := obj.ChildByFieldName("object")
	innerProp := obj.ChildByFieldName("property")

	if innerObj == nil || innerProp == nil {
		return "", domain.TestStatusActive, ""
	}

	objName := parser.GetNodeText(innerObj, source)
	if !testAliases[objName] {
		return "", domain.TestStatusActive, ""
	}

	middleProp := parser.GetNodeText(innerProp, source)
	if middleProp == "describe" {
		outerProp := parser.GetNodeText(prop, source)
		status, modifier := parseModifierStatusAndModifier(outerProp)
		return funcTestDescribe, status, modifier
	}

	return "", domain.TestStatusActive, ""
}

func parseNode(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite, testAliases map[string]bool) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)

		switch child.Type() {
		case "expression_statement":
			if expr := parser.FindChildByType(child, "call_expression"); expr != nil {
				processCallExpression(expr, source, filename, file, currentSuite, testAliases)
			}
		default:
			parseNode(child, source, filename, file, currentSuite, testAliases)
		}
	}
}

func parseSimpleMemberExpression(obj *sitter.Node, prop *sitter.Node, source []byte, testAliases map[string]bool) (string, domain.TestStatus, string) {
	objName := parser.GetNodeText(obj, source)
	if !testAliases[objName] {
		return "", domain.TestStatusActive, ""
	}

	propName := parser.GetNodeText(prop, source)
	switch propName {
	case "describe":
		return funcTestDescribe, domain.TestStatusActive, ""
	case jstest.ModifierSkip:
		return funcTest, domain.TestStatusSkipped, jstest.ModifierSkip
	case modifierFixme:
		return funcTest, domain.TestStatusSkipped, modifierFixme
	case jstest.ModifierOnly:
		return funcTest, domain.TestStatusFocused, jstest.ModifierOnly
	}

	return "", domain.TestStatusActive, ""
}

func processCallExpression(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite, testAliases map[string]bool) {
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return
	}

	args := node.ChildByFieldName("arguments")
	if args == nil {
		return
	}

	funcName, status, modifier := parseFunctionName(funcNode, source, testAliases)
	if funcName == "" {
		return
	}

	switch funcName {
	case funcTestDescribe:
		processSuite(node, args, source, filename, file, currentSuite, status, modifier, testAliases)
	case funcTest:
		processTest(node, args, source, filename, file, currentSuite, status, modifier)
	}
}

func processSuite(callNode *sitter.Node, args *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus, modifier string, testAliases map[string]bool) {
	name := jstest.ExtractTestName(args, source)
	if name == "" {
		return
	}

	suite := domain.TestSuite{
		Name:     name,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(callNode, filename),
	}

	if callback := jstest.FindCallback(args); callback != nil {
		body := callback.ChildByFieldName("body")
		if body != nil {
			parseNode(body, source, filename, file, &suite, testAliases)
		}
	}

	jstest.AddSuiteToTarget(suite, parentSuite, file)
}

func processTest(callNode *sitter.Node, args *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus, modifier string) {
	name := jstest.ExtractTestName(args, source)
	if name == "" {
		return
	}

	test := domain.Test{
		Name:     name,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(callNode, filename),
	}

	jstest.AddTestToTarget(test, parentSuite, file)
}
