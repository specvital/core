// Package junit5 implements JUnit 5 (Jupiter) test framework support for Java test files.
package junit5

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
	"github.com/specvital/core/pkg/parser/strategies/shared/javaast"
)

func init() {
	framework.Register(NewDefinition())
}

func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      framework.FrameworkJUnit5,
		Languages: []domain.Language{domain.LanguageJava, domain.LanguageKotlin},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher(
				"org.junit.jupiter.api.Test",
				"org.junit.jupiter.api.",
				"org.junit.jupiter.params.",
			),
			&javaast.JavaTestFileMatcher{},
			&KotlinTestFileMatcher{},
			&JUnit5ContentMatcher{},
		},
		ConfigParser: nil,
		Parser:       &JUnit5Parser{},
		Priority:     framework.PriorityGeneric,
	}
}

// JUnit5ContentMatcher matches JUnit 5 specific patterns.
type JUnit5ContentMatcher struct{}

var junit5Patterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	{regexp.MustCompile(`@Test\s*(?:\(|$|\n)`), "@Test annotation"},
	{regexp.MustCompile(`@ParameterizedTest`), "@ParameterizedTest annotation"},
	{regexp.MustCompile(`@RepeatedTest`), "@RepeatedTest annotation"},
	{regexp.MustCompile(`@TestFactory`), "@TestFactory annotation"},
	{regexp.MustCompile(`@TestTemplate`), "@TestTemplate annotation"},
	{regexp.MustCompile(`@Nested`), "@Nested annotation"},
	{regexp.MustCompile(`@DisplayName`), "@DisplayName annotation"},
	{regexp.MustCompile(`import\s+org\.junit\.jupiter`), "JUnit Jupiter import"},
}

func (m *JUnit5ContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileContent {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		content = []byte(signal.Value)
	}

	// Exclude JUnit 4 files: has org.junit.* import but no org.junit.jupiter import
	if javaast.JUnit4ImportPattern.Match(content) && !javaast.JUnit5ImportPattern.Match(content) {
		return framework.NoMatch()
	}

	for _, p := range junit5Patterns {
		if p.pattern.Match(content) {
			return framework.PartialMatch(40, "Found JUnit 5 pattern: "+p.desc)
		}
	}

	return framework.NoMatch()
}

// JUnit5Parser extracts test definitions from Java JUnit 5 files.
type JUnit5Parser struct{}

func (p *JUnit5Parser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	// Route to Kotlin parser for .kt files
	if strings.HasSuffix(filename, ".kt") {
		return parseKotlinTestFile(ctx, source, filename)
	}

	tree, err := parser.ParseWithPool(ctx, domain.LanguageJava, source)
	if err != nil {
		return nil, fmt.Errorf("junit5 parser: failed to parse %s: %w", filename, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	suites := parseTestClasses(root, source, filename)

	return &domain.TestFile{
		Path:      filename,
		Language:  domain.LanguageJava,
		Framework: framework.FrameworkJUnit5,
		Suites:    suites,
	}, nil
}

// maxNestedDepth limits recursion depth for @Nested class parsing.
// JUnit 5 allows arbitrary nesting, but we limit to prevent stack overflow.
const maxNestedDepth = 20

func parseTestClasses(root *sitter.Node, source []byte, filename string) []domain.TestSuite {
	var suites []domain.TestSuite

	parser.WalkTree(root, func(node *sitter.Node) bool {
		if node.Type() == javaast.NodeClassDeclaration {
			if suite := parseTestClassWithDepth(node, source, filename, 0); suite != nil {
				suites = append(suites, *suite)
			}
			return false // Don't recurse into nested classes here
		}
		return true
	})

	return suites
}

func parseTestClassWithDepth(node *sitter.Node, source []byte, filename string, depth int) *domain.TestSuite {
	if depth > maxNestedDepth {
		return nil
	}
	className := javaast.GetClassName(node, source)
	if className == "" {
		return nil
	}

	modifiers := javaast.GetModifiers(node)
	classStatus, classModifier := getClassStatusAndModifier(modifiers, source)

	body := javaast.GetClassBody(node)
	if body == nil {
		return nil
	}

	var tests []domain.Test
	var nestedSuites []domain.TestSuite

	for i := 0; i < int(body.ChildCount()); i++ {
		child := body.Child(i)

		switch child.Type() {
		case javaast.NodeMethodDeclaration:
			if test := parseTestMethod(child, source, filename, classStatus, classModifier); test != nil {
				tests = append(tests, *test)
			}

		case javaast.NodeClassDeclaration:
			// Handle @Nested classes
			nestedModifiers := javaast.GetModifiers(child)
			if javaast.HasAnnotation(nestedModifiers, source, "Nested") {
				if nested := parseTestClassWithDepth(child, source, filename, depth+1); nested != nil {
					nestedSuites = append(nestedSuites, *nested)
				}
			}
		}
	}

	if len(tests) == 0 && len(nestedSuites) == 0 {
		return nil
	}

	return &domain.TestSuite{
		Name:     className,
		Status:   classStatus,
		Modifier: classModifier,
		Location: parser.GetLocation(node, filename),
		Tests:    tests,
		Suites:   nestedSuites,
	}
}

func parseTestMethod(node *sitter.Node, source []byte, filename string, classStatus domain.TestStatus, classModifier string) *domain.Test {
	modifiers := javaast.GetModifiers(node)
	if modifiers == nil {
		return nil
	}

	annotations := javaast.GetAnnotations(modifiers)
	isTest := false
	var displayName string
	status := classStatus
	modifier := classModifier

	for _, ann := range annotations {
		name := javaast.GetAnnotationName(ann, source)

		switch name {
		case "Test", "ParameterizedTest", "RepeatedTest", "TestFactory", "TestTemplate":
			isTest = true
		case "Disabled":
			status = domain.TestStatusSkipped
			modifier = "@Disabled"
		case "DisplayName":
			displayName = javaast.GetAnnotationArgument(ann, source)
		default:
			// Detect custom @TestTemplate-based annotations (e.g., @CartesianProductTest)
			// JUnit convention: custom test annotations typically end with "Test"
			if strings.HasSuffix(name, "Test") {
				isTest = true
			}
		}
	}

	if !isTest {
		return nil
	}

	methodName := javaast.GetMethodName(node, source)
	if methodName == "" {
		return nil
	}

	testName := methodName
	if displayName != "" {
		testName = displayName
	}

	return &domain.Test{
		Name:     testName,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(node, filename),
	}
}

func getClassStatusAndModifier(modifiers *sitter.Node, source []byte) (domain.TestStatus, string) {
	if modifiers == nil {
		return domain.TestStatusActive, ""
	}

	if javaast.HasAnnotation(modifiers, source, "Disabled") {
		return domain.TestStatusSkipped, "@Disabled"
	}

	return domain.TestStatusActive, ""
}
