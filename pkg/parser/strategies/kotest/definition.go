// Package kotest implements Kotest test framework support for Kotlin test files.
package kotest

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
	"github.com/specvital/core/pkg/parser/strategies/shared/kotlinast"
)

const frameworkName = "kotest"

// Detection confidence scores (aligned with 4-stage detection system).
const (
	confidenceFileName = 20 // Filename pattern match
	confidenceContent  = 40 // Content pattern match
)

func init() {
	framework.Register(NewDefinition())
}

func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      frameworkName,
		Languages: []domain.Language{domain.LanguageKotlin},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher(
				"io.kotest.core",
				"io.kotest.core.spec",
				"io.kotest.core.spec.style",
			),
			&KotestFileMatcher{},
			&KotestContentMatcher{},
		},
		ConfigParser: nil,
		Parser:       &KotestParser{},
		Priority:     framework.PriorityGeneric,
	}
}

// KotestFileMatcher matches *Test.kt, *Spec.kt files.
type KotestFileMatcher struct{}

func (m *KotestFileMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileName {
		return framework.NoMatch()
	}

	if kotlinast.IsKotlinTestFile(signal.Value) {
		return framework.PartialMatch(confidenceFileName, "Kotest file naming convention")
	}

	return framework.NoMatch()
}

// KotestContentMatcher matches Kotest-specific patterns.
type KotestContentMatcher struct{}

var kotestPatterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	{regexp.MustCompile(`:\s*FunSpec\s*\(`), "extends FunSpec"},
	{regexp.MustCompile(`:\s*StringSpec\s*\(`), "extends StringSpec"},
	{regexp.MustCompile(`:\s*BehaviorSpec\s*\(`), "extends BehaviorSpec"},
	{regexp.MustCompile(`:\s*DescribeSpec\s*\(`), "extends DescribeSpec"},
	{regexp.MustCompile(`:\s*WordSpec\s*\(`), "extends WordSpec"},
	{regexp.MustCompile(`:\s*FreeSpec\s*\(`), "extends FreeSpec"},
	{regexp.MustCompile(`:\s*FeatureSpec\s*\(`), "extends FeatureSpec"},
	{regexp.MustCompile(`:\s*ExpectSpec\s*\(`), "extends ExpectSpec"},
	{regexp.MustCompile(`:\s*ShouldSpec\s*\(`), "extends ShouldSpec"},
	{regexp.MustCompile(`:\s*AnnotationSpec\s*\(`), "extends AnnotationSpec"},
	{regexp.MustCompile(`\bshouldBe\s+`), "shouldBe matcher"},
	{regexp.MustCompile(`\bshouldNotBe\s+`), "shouldNotBe matcher"},
	{regexp.MustCompile(`\bshouldThrow<`), "shouldThrow assertion"},
	{regexp.MustCompile(`import\s+io\.kotest\.`), "Kotest import"},
}

func (m *KotestContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileContent {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		content = []byte(signal.Value)
	}

	for _, p := range kotestPatterns {
		if p.pattern.Match(content) {
			return framework.PartialMatch(confidenceContent, "Found Kotest pattern: "+p.desc)
		}
	}

	return framework.NoMatch()
}

// KotestParser extracts test definitions from Kotlin Kotest files.
type KotestParser struct{}

func (p *KotestParser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	cleanSource := kotlinast.SanitizeSource(source)

	tree, err := parser.ParseWithPool(ctx, domain.LanguageKotlin, cleanSource)
	if err != nil {
		return nil, fmt.Errorf("kotest parser: failed to parse %s: %w", filename, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	suites := parseTestClasses(root, cleanSource, filename)

	return &domain.TestFile{
		Path:      filename,
		Language:  domain.LanguageKotlin,
		Framework: frameworkName,
		Suites:    suites,
	}, nil
}

func parseTestClasses(root *sitter.Node, source []byte, filename string) []domain.TestSuite {
	var suites []domain.TestSuite

	parser.WalkTree(root, func(node *sitter.Node) bool {
		if node.Type() == kotlinast.NodeClassDeclaration {
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
	className := kotlinast.GetClassName(node, source)
	if className == "" {
		return nil
	}

	isSpec, specStyle := kotlinast.IsKotestSpec(node, source)
	if !isSpec {
		return nil
	}

	status := domain.TestStatusActive
	modifier := ""

	modifiers := kotlinast.GetModifiers(node)
	if kotlinast.HasAnnotation(modifiers, source, "Disabled") ||
		kotlinast.HasAnnotation(modifiers, source, "Ignore") {
		status = domain.TestStatusSkipped
		modifier = "@Disabled"
	}

	suite := &domain.TestSuite{
		Name:     className,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(node, filename),
	}

	// Kotest specs use constructor lambda: class MyTest : FunSpec({ ... })
	// Try constructor lambda first (most common pattern)
	lambda := kotlinast.GetConstructorLambda(node)
	if lambda != nil {
		switch specStyle {
		case kotlinast.SpecStringSpec:
			parseStringSpecTests(lambda, source, filename, suite)
		case kotlinast.SpecFunSpec:
			parseFunSpecTests(lambda, source, filename, suite)
		case kotlinast.SpecBehaviorSpec:
			parseBehaviorSpecTests(lambda, source, filename, suite)
		case kotlinast.SpecDescribeSpec:
			parseDescribeSpecTests(lambda, source, filename, suite)
		case kotlinast.SpecWordSpec:
			parseWordSpecTests(lambda, source, filename, suite)
		case kotlinast.SpecFreeSpec:
			parseFreeSpecTests(lambda, source, filename, suite)
		case kotlinast.SpecShouldSpec:
			parseShouldSpecTests(lambda, source, filename, suite)
		default:
			parseGenericSpecTests(lambda, source, filename, suite, specStyle)
		}
	}

	// Also check class body for AnnotationSpec style (uses methods with @Test)
	body := kotlinast.GetClassBody(node)
	if body != nil && specStyle == kotlinast.SpecAnnotationSpec {
		parseAnnotationSpecTests(body, source, filename, suite)
	}

	if len(suite.Tests) == 0 && len(suite.Suites) == 0 {
		return nil
	}

	return suite
}

// StringSpec: "test name" { ... }
// In StringSpec, the test name is a string literal followed by a lambda block.
func parseStringSpecTests(body *sitter.Node, source []byte, filename string, suite *domain.TestSuite) {
	parser.WalkTree(body, func(n *sitter.Node) bool {
		if n.Type() == kotlinast.NodeCallExpression {
			testName := ""
			hasLambda := false

			for i := 0; i < int(n.ChildCount()); i++ {
				child := n.Child(i)
				if child.Type() == kotlinast.NodeStringLiteral {
					testName = kotlinast.ExtractStringContent(child, source)
				}
				if child.Type() == kotlinast.NodeCallSuffix {
					hasLambda = true
				}
			}

			if testName != "" && hasLambda {
				status := domain.TestStatusActive
				modifier := ""
				if strings.HasPrefix(testName, "!") {
					status = domain.TestStatusSkipped
					modifier = "!"
					testName = strings.TrimPrefix(testName, "!")
				}

				test := domain.Test{
					Name:     testName,
					Status:   status,
					Modifier: modifier,
					Location: parser.GetLocation(n, filename),
				}
				suite.Tests = append(suite.Tests, test)
			}
			return false
		}
		return true
	})
}

// FunSpec: test("name") { ... }, context("name") { ... }
func parseFunSpecTests(body *sitter.Node, source []byte, filename string, suite *domain.TestSuite) {
	parseCallExpressions(body, source, filename, suite, func(name string, node *sitter.Node, source []byte) bool {
		switch name {
		case "test", "xtest", "context", "xcontext":
			return true
		}
		return false
	})
}

// BehaviorSpec: Given/When/Then pattern
func parseBehaviorSpecTests(body *sitter.Node, source []byte, filename string, suite *domain.TestSuite) {
	parseCallExpressions(body, source, filename, suite, func(name string, node *sitter.Node, source []byte) bool {
		switch strings.ToLower(name) {
		case "given", "xgiven", "when", "xwhen", "then", "xthen", "and", "xand":
			return true
		}
		// Also handle backtick-escaped versions
		if name == "`given`" || name == "`when`" || name == "`then`" {
			return true
		}
		return false
	})
}

// DescribeSpec: describe/it pattern
func parseDescribeSpecTests(body *sitter.Node, source []byte, filename string, suite *domain.TestSuite) {
	parseCallExpressions(body, source, filename, suite, func(name string, node *sitter.Node, source []byte) bool {
		switch name {
		case "describe", "xdescribe", "context", "xcontext", "it", "xit":
			return true
		}
		return false
	})
}

// AnnotationSpec: @Test annotated methods
func parseAnnotationSpecTests(body *sitter.Node, source []byte, filename string, suite *domain.TestSuite) {
	parser.WalkTree(body, func(node *sitter.Node) bool {
		if node.Type() == kotlinast.NodeFunctionDeclaration {
			modifiers := kotlinast.GetModifiers(node)
			if kotlinast.HasAnnotation(modifiers, source, "Test") {
				test := parseAnnotatedTestMethod(node, source, filename, modifiers)
				if test != nil {
					suite.Tests = append(suite.Tests, *test)
				}
			}
			return false
		}
		return true
	})
}

func parseAnnotatedTestMethod(node *sitter.Node, source []byte, filename string, modifiers *sitter.Node) *domain.Test {
	name := getFunctionName(node, source)
	if name == "" {
		return nil
	}

	status := domain.TestStatusActive
	modifier := ""

	if kotlinast.HasAnnotation(modifiers, source, "Disabled") ||
		kotlinast.HasAnnotation(modifiers, source, "Ignore") {
		status = domain.TestStatusSkipped
		modifier = "@Disabled"
	}

	return &domain.Test{
		Name:     name,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(node, filename),
	}
}

func getFunctionName(node *sitter.Node, source []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == kotlinast.NodeIdentifier {
			return child.Content(source)
		}
	}
	return ""
}

// Generic spec style parsing (fallback for unhandled spec styles)
func parseGenericSpecTests(body *sitter.Node, source []byte, filename string, suite *domain.TestSuite, _ string) {
	parseCallExpressions(body, source, filename, suite, func(name string, node *sitter.Node, source []byte) bool {
		switch name {
		case "test", "xtest", "it", "xit", "should", "xshould",
			"describe", "xdescribe", "context", "xcontext",
			"feature", "xfeature", "scenario", "xscenario",
			"expect", "xexpect":
			return true
		}
		return false
	})
}

type testMatcher func(name string, node *sitter.Node, source []byte) bool

func parseCallExpressions(node *sitter.Node, source []byte, filename string, suite *domain.TestSuite, matcher testMatcher) {
	parser.WalkTree(node, func(n *sitter.Node) bool {
		if n.Type() == kotlinast.NodeCallExpression {
			processCallExpression(n, source, filename, suite, matcher)
			return false
		}
		// For StringSpec, string literals in init block are tests
		if n.Type() == kotlinast.NodeStringLiteral || n.Type() == kotlinast.NodeLineStringLiteral {
			// Check if this string has a lambda (making it a test)
			parent := n.Parent()
			if parent != nil && (parent.Type() == kotlinast.NodeInfixExpression || parent.Type() == kotlinast.NodePostfixExpression) {
				testName := kotlinast.ExtractStringContent(n, source)
				if testName != "" {
					test := domain.Test{
						Name:     testName,
						Status:   domain.TestStatusActive,
						Location: parser.GetLocation(n, filename),
					}
					suite.Tests = append(suite.Tests, test)
				}
			}
		}
		return true
	})
}

func processCallExpression(node *sitter.Node, source []byte, filename string, suite *domain.TestSuite, matcher testMatcher) {
	funcName, innerCall := getInnermostCallName(node, source)
	if funcName == "" {
		return
	}

	if !matcher(funcName, node, source) {
		lambda := kotlinast.GetLambdaFromCall(node)
		if lambda != nil {
			parseCallExpressions(lambda, source, filename, suite, matcher)
		}
		return
	}

	status, modifier := getStatusAndModifier(funcName)

	testName := ""
	if innerCall != nil {
		testName = kotlinast.GetFirstStringArgument(innerCall, source)
	}
	if testName == "" {
		testName = kotlinast.GetFirstStringArgument(node, source)
	}

	// Handle suite-like constructs (describe, context, given, etc.)
	if isSuiteFunction(funcName) {
		// Skip suites without names - they're likely parsing errors
		if testName == "" {
			lambda := kotlinast.GetLambdaFromCall(node)
			if lambda != nil {
				parseCallExpressions(lambda, source, filename, suite, matcher)
			}
			return
		}

		nestedSuite := domain.TestSuite{
			Name:     testName,
			Status:   status,
			Modifier: modifier,
			Location: parser.GetLocation(node, filename),
		}

		lambda := kotlinast.GetLambdaFromCall(node)
		if lambda != nil {
			parseCallExpressions(lambda, source, filename, &nestedSuite, matcher)
		}

		if len(nestedSuite.Tests) > 0 || len(nestedSuite.Suites) > 0 {
			suite.Suites = append(suite.Suites, nestedSuite)
		}
		return
	}

	if testName == "" {
		testName = funcName
	}

	test := domain.Test{
		Name:     testName,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(node, filename),
	}
	suite.Tests = append(suite.Tests, test)
}

// getInnermostCallName extracts the function name from potentially nested call expressions.
// In Kotlin, test("name") { } is parsed as: call_expression(call_expression("test"), call_suffix({ }))
// Returns the function name and the inner call node (for argument extraction).
func getInnermostCallName(node *sitter.Node, source []byte) (string, *sitter.Node) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == kotlinast.NodeCallExpression {
			innerName := kotlinast.GetCallExpressionName(child, source)
			if innerName != "" {
				return innerName, child
			}
		}
	}
	return kotlinast.GetCallExpressionName(node, source), node
}

func isSuiteFunction(name string) bool {
	switch strings.ToLower(strings.TrimPrefix(name, "x")) {
	case "describe", "context", "given", "when", "feature":
		return true
	}
	return false
}

func getStatusAndModifier(funcName string) (domain.TestStatus, string) {
	// Handle x-prefixed (xtest, xdescribe, etc.) - skipped
	if strings.HasPrefix(funcName, "x") || strings.HasPrefix(funcName, "X") {
		return domain.TestStatusSkipped, funcName
	}
	// Handle ! prefix (BDD negation in Kotest) - skipped
	if strings.HasPrefix(funcName, "!") {
		return domain.TestStatusSkipped, "!" + funcName
	}
	return domain.TestStatusActive, ""
}

// WordSpec: "context" should { "test" { } } or "context" When { "test" { } }
// Uses infix_expression for should/When containers, call_expression for leaf tests
func parseWordSpecTests(body *sitter.Node, source []byte, filename string, suite *domain.TestSuite) {
	parser.WalkTree(body, func(n *sitter.Node) bool {
		if n.Type() == kotlinast.NodeInfixExpression {
			if processed := processWordSpecInfix(n, source, filename, suite); processed {
				return false
			}
		}
		return true
	})
}

func processWordSpecInfix(node *sitter.Node, source []byte, filename string, suite *domain.TestSuite) bool {
	var contextName string
	var operator string
	var lambda *sitter.Node

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case kotlinast.NodeStringLiteral, kotlinast.NodeLineStringLiteral:
			contextName = kotlinast.ExtractStringContent(child, source)
		case kotlinast.NodeIdentifier:
			operator = child.Content(source)
		case kotlinast.NodeCallSuffix:
			lambda = kotlinast.GetLambdaFromCallSuffix(child)
		case kotlinast.NodeLambdaLiteral:
			lambda = child
		case kotlinast.NodeAnnotatedLambda:
			lambda = kotlinast.GetLambdaFromAnnotatedLambda(child)
		}
	}

	if !isWordSpecOperator(operator) {
		return false
	}

	if contextName == "" || lambda == nil {
		return false
	}

	status, modifier := getStatusAndModifier(operator)

	nestedSuite := domain.TestSuite{
		Name:     contextName,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(node, filename),
	}

	parseWordSpecNestedContent(lambda, source, filename, &nestedSuite)

	if len(nestedSuite.Tests) > 0 || len(nestedSuite.Suites) > 0 {
		suite.Suites = append(suite.Suites, nestedSuite)
	}

	return true
}

func isWordSpecOperator(op string) bool {
	switch op {
	case "should", "xshould", "Should", "xShould",
		"When", "xWhen", "`when`", "x`when`":
		return true
	}
	return false
}

func parseWordSpecNestedContent(body *sitter.Node, source []byte, filename string, suite *domain.TestSuite) {
	parser.WalkTree(body, func(n *sitter.Node) bool {
		if n.Type() == kotlinast.NodeInfixExpression {
			if processed := processWordSpecInfix(n, source, filename, suite); processed {
				return false
			}
		}
		if n.Type() == kotlinast.NodeCallExpression {
			if processed := processStringCallTest(n, source, filename, suite); processed {
				return false
			}
		}

		return true
	})
}

func processStringCallTest(node *sitter.Node, source []byte, filename string, suite *domain.TestSuite) bool {
	var testName string
	var hasLambda bool

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case kotlinast.NodeStringLiteral, kotlinast.NodeLineStringLiteral:
			testName = kotlinast.ExtractStringContent(child, source)
		case kotlinast.NodeCallSuffix:
			if kotlinast.GetLambdaFromCallSuffix(child) != nil {
				hasLambda = true
			}
		}
	}

	if testName == "" || !hasLambda {
		return false
	}

	status := domain.TestStatusActive
	modifier := ""

	test := domain.Test{
		Name:     testName,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(node, filename),
	}
	suite.Tests = append(suite.Tests, test)

	return true
}

// FreeSpec: "context" - { "test" { } }
// Uses additive_expression (minus operator) for containers, call_expression for leaf tests
func parseFreeSpecTests(body *sitter.Node, source []byte, filename string, suite *domain.TestSuite) {
	parser.WalkTree(body, func(n *sitter.Node) bool {
		if n.Type() == kotlinast.NodeAdditiveExpression {
			if processed := processFreeSpecContainer(n, source, filename, suite); processed {
				return false
			}
		}
		if n.Type() == kotlinast.NodeInfixExpression {
			if processed := processFreeSpecInfix(n, source, filename, suite); processed {
				return false
			}
		}
		if n.Type() == kotlinast.NodeCallExpression {
			if processed := processStringCallTest(n, source, filename, suite); processed {
				return false
			}
		}

		return true
	})
}

func processFreeSpecContainer(node *sitter.Node, source []byte, filename string, suite *domain.TestSuite) bool {
	var contextName string
	var lambda *sitter.Node
	var hasMinus bool

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case kotlinast.NodeStringLiteral, kotlinast.NodeLineStringLiteral:
			contextName = kotlinast.ExtractStringContent(child, source)
		case kotlinast.NodeLambdaLiteral:
			lambda = child
		case kotlinast.NodeAnnotatedLambda:
			lambda = kotlinast.GetLambdaFromAnnotatedLambda(child)
		case kotlinast.NodeCallSuffix:
			lambda = kotlinast.GetLambdaFromCallSuffix(child)
		}
		if child.Content(source) == "-" {
			hasMinus = true
		}
	}

	if !hasMinus || contextName == "" || lambda == nil {
		return false
	}

	nestedSuite := domain.TestSuite{
		Name:     contextName,
		Status:   domain.TestStatusActive,
		Location: parser.GetLocation(node, filename),
	}
	parseFreeSpecTests(lambda, source, filename, &nestedSuite)

	if len(nestedSuite.Tests) > 0 || len(nestedSuite.Suites) > 0 {
		suite.Suites = append(suite.Suites, nestedSuite)
	}

	return true
}

func processFreeSpecInfix(node *sitter.Node, source []byte, filename string, suite *domain.TestSuite) bool {
	var contextName string
	var operator string
	var lambda *sitter.Node

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case kotlinast.NodeStringLiteral, kotlinast.NodeLineStringLiteral:
			contextName = kotlinast.ExtractStringContent(child, source)
		case kotlinast.NodeIdentifier:
			operator = child.Content(source)
		case kotlinast.NodeCallSuffix:
			lambda = kotlinast.GetLambdaFromCallSuffix(child)
		case kotlinast.NodeLambdaLiteral:
			lambda = child
		case kotlinast.NodeAnnotatedLambda:
			lambda = kotlinast.GetLambdaFromAnnotatedLambda(child)
		}
	}

	// FreeSpec uses "minus" as the operator name for "-"
	if operator != "minus" {
		return false
	}

	if contextName == "" || lambda == nil {
		return false
	}

	nestedSuite := domain.TestSuite{
		Name:     contextName,
		Status:   domain.TestStatusActive,
		Location: parser.GetLocation(node, filename),
	}

	parseFreeSpecTests(lambda, source, filename, &nestedSuite)

	if len(nestedSuite.Tests) > 0 || len(nestedSuite.Suites) > 0 {
		suite.Suites = append(suite.Suites, nestedSuite)
	}

	return true
}

// ShouldSpec: should("test") { } and context("context") { }
func parseShouldSpecTests(body *sitter.Node, source []byte, filename string, suite *domain.TestSuite) {
	parseCallExpressions(body, source, filename, suite, func(name string, node *sitter.Node, source []byte) bool {
		switch name {
		case "should", "xshould", "context", "xcontext":
			return true
		}
		return false
	})
}

