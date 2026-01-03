// Package testng implements TestNG test framework support for Java test files.
package testng

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

const frameworkName = framework.FrameworkTestNG

// maxNestedDepth limits recursion depth for nested class parsing.
// TestNG allows arbitrary nesting, but we limit to prevent stack overflow.
const maxNestedDepth = 20

func init() {
	framework.Register(NewDefinition())
}

// NewDefinition creates a TestNG framework definition.
func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      frameworkName,
		Languages: []domain.Language{domain.LanguageJava},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher(
				"org.testng.annotations.Test",
				"org.testng.annotations.",
				"org.testng.",
			),
			&TestNGFileMatcher{},
			&TestNGContentMatcher{},
		},
		ConfigParser: nil,
		Parser:       &TestNGParser{},
		// PrioritySpecialized (200) > JUnit5's PriorityGeneric (100) to resolve @Test annotation collision.
		// Both frameworks use @Test but with different packages (org.testng vs org.junit.jupiter).
		// Import matching provides strong disambiguation (60 pts), but priority ensures correct
		// framework selection when imports are ambiguous (e.g., wildcard imports).
		Priority: framework.PrioritySpecialized,
	}
}

// TestNGFileMatcher matches *Test.java and Test*.java files.
type TestNGFileMatcher struct{}

func (m *TestNGFileMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileName {
		return framework.NoMatch()
	}

	filename := signal.Value
	base := filename
	if idx := strings.LastIndex(filename, "/"); idx >= 0 {
		base = filename[idx+1:]
	}

	if !strings.HasSuffix(base, ".java") {
		return framework.NoMatch()
	}

	name := strings.TrimSuffix(base, ".java")

	if strings.HasSuffix(name, "Test") || strings.HasSuffix(name, "Tests") {
		return framework.PartialMatch(20, "TestNG file naming: *Test.java")
	}

	if strings.HasPrefix(name, "Test") {
		return framework.PartialMatch(20, "TestNG file naming: Test*.java")
	}

	return framework.NoMatch()
}

// TestNGContentMatcher matches TestNG specific patterns.
type TestNGContentMatcher struct{}

// Pre-compiled regex patterns for annotation attribute extraction.
// Note: These patterns handle simple string literals. Complex expressions
// (escaped quotes, constant references) are not supported.
var (
	enabledFalsePattern = regexp.MustCompile(`enabled\s*=\s*false`)
	descriptionPattern  = regexp.MustCompile(`description\s*=\s*"([^"]*)"`)
)

var testngPatterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	{regexp.MustCompile(`import\s+org\.testng`), "TestNG import"},
	{regexp.MustCompile(`@Test\s*\([^)]*enabled\s*=`), "@Test(enabled=) annotation"},
	{regexp.MustCompile(`@DataProvider`), "@DataProvider annotation"},
	{regexp.MustCompile(`@BeforeClass`), "@BeforeClass annotation"},
	{regexp.MustCompile(`@AfterClass`), "@AfterClass annotation"},
	{regexp.MustCompile(`@BeforeMethod`), "@BeforeMethod annotation"},
	{regexp.MustCompile(`@AfterMethod`), "@AfterMethod annotation"},
}

// configAnnotations defines TestNG configuration annotations that exclude a method from being a test.
// When a class has @Test annotation, all public methods become tests EXCEPT those with config annotations.
var configAnnotations = map[string]bool{
	"BeforeMethod": true,
	"AfterMethod":  true,
	"BeforeClass":  true,
	"AfterClass":   true,
	"BeforeSuite":  true,
	"AfterSuite":   true,
	"BeforeTest":   true,
	"AfterTest":    true,
	"BeforeGroups": true,
	"AfterGroups":  true,
	"DataProvider": true,
	"Factory":      true,
}

func (m *TestNGContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileContent {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		content = []byte(signal.Value)
	}

	for _, p := range testngPatterns {
		if p.pattern.Match(content) {
			return framework.PartialMatch(40, "Found TestNG pattern: "+p.desc)
		}
	}

	return framework.NoMatch()
}

// TestNGParser extracts test definitions from Java TestNG files.
type TestNGParser struct{}

func (p *TestNGParser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	cleanSource := javaast.SanitizeSource(source)

	tree, err := parser.ParseWithPool(ctx, domain.LanguageJava, cleanSource)
	if err != nil {
		return nil, fmt.Errorf("testng parser: failed to parse %s: %w", filename, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	suites := parseTestClasses(root, cleanSource, filename)

	return &domain.TestFile{
		Path:      filename,
		Language:  domain.LanguageJava,
		Framework: frameworkName,
		Suites:    suites,
	}, nil
}

func parseTestClasses(root *sitter.Node, source []byte, filename string) []domain.TestSuite {
	var suites []domain.TestSuite

	parser.WalkTree(root, func(node *sitter.Node) bool {
		if node.Type() == javaast.NodeClassDeclaration {
			if suite := parseTestClassWithDepth(node, source, filename, 0); suite != nil {
				suites = append(suites, *suite)
			}
			return false // Don't recurse into nested classes here; handled by parseTestClassWithDepth
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
	hasClassLevelTest := javaast.HasAnnotation(modifiers, source, "Test")

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
			if test := parseTestMethod(child, source, filename, classStatus, classModifier, hasClassLevelTest); test != nil {
				tests = append(tests, *test)
			}

		case javaast.NodeClassDeclaration:
			// Handle nested classes (TestNG doesn't require @Nested annotation unlike JUnit5)
			if nested := parseTestClassWithDepth(child, source, filename, depth+1); nested != nil {
				nestedSuites = append(nestedSuites, *nested)
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

func parseTestMethod(node *sitter.Node, source []byte, filename string, classStatus domain.TestStatus, classModifier string, hasClassLevelTest bool) *domain.Test {
	modifiers := javaast.GetModifiers(node)

	annotations := javaast.GetAnnotations(modifiers)
	hasMethodTest := false
	var description string
	status := classStatus
	modifier := classModifier

	for _, ann := range annotations {
		name := javaast.GetAnnotationName(ann, source)

		if configAnnotations[name] {
			return nil
		}

		if name == "Test" {
			hasMethodTest = true
			if hasEnabledFalse(ann, source) {
				status = domain.TestStatusSkipped
				modifier = "@Test(enabled=false)"
			}
			description = getTestDescription(ann, source)
		}
	}

	isTest := hasMethodTest
	if !isTest && hasClassLevelTest {
		isTest = isPublicMethod(modifiers)
	}

	if !isTest {
		return nil
	}

	methodName := javaast.GetMethodName(node, source)
	if methodName == "" {
		return nil
	}

	testName := methodName
	if description != "" {
		testName = description
	}

	return &domain.Test{
		Name:     testName,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(node, filename),
	}
}

// isPublicMethod checks if a method has the public modifier.
func isPublicMethod(modifiers *sitter.Node) bool {
	if modifiers == nil {
		return false
	}

	for i := 0; i < int(modifiers.ChildCount()); i++ {
		child := modifiers.Child(i)
		if child.Type() == "public" {
			return true
		}
	}
	return false
}

// hasEnabledFalse checks if the @Test annotation has enabled=false.
func hasEnabledFalse(annotation *sitter.Node, source []byte) bool {
	for i := 0; i < int(annotation.ChildCount()); i++ {
		child := annotation.Child(i)
		if child.Type() == javaast.NodeAnnotationArgumentList {
			content := child.Content(source)
			return enabledFalsePattern.MatchString(content)
		}
	}
	return false
}

// getTestDescription extracts the description attribute from @Test annotation.
func getTestDescription(annotation *sitter.Node, source []byte) string {
	for i := 0; i < int(annotation.ChildCount()); i++ {
		child := annotation.Child(i)
		if child.Type() == javaast.NodeAnnotationArgumentList {
			content := child.Content(source)
			matches := descriptionPattern.FindStringSubmatch(content)
			if len(matches) > 1 {
				return matches[1]
			}
		}
	}
	return ""
}

func getClassStatusAndModifier(modifiers *sitter.Node, source []byte) (domain.TestStatus, string) {
	if modifiers == nil {
		return domain.TestStatusActive, ""
	}

	annotations := javaast.GetAnnotations(modifiers)
	for _, ann := range annotations {
		name := javaast.GetAnnotationName(ann, source)
		if name == "Test" && hasEnabledFalse(ann, source) {
			return domain.TestStatusSkipped, "@Test(enabled=false)"
		}
	}

	return domain.TestStatusActive, ""
}
