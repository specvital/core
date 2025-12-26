// Package junit4 implements JUnit 4 test framework support for Java test files.
package junit4

import (
	"context"
	"fmt"
	"regexp"

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
		Name:      framework.FrameworkJUnit4,
		Languages: []domain.Language{domain.LanguageJava},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher(
				"org.junit.Test",
				"org.junit.Before",
				"org.junit.After",
				"org.junit.BeforeClass",
				"org.junit.AfterClass",
				"org.junit.Ignore",
				"org.junit.runner.",
			),
			&javaast.JavaTestFileMatcher{},
			&JUnit4ContentMatcher{},
		},
		ConfigParser: nil,
		Parser:       &JUnit4Parser{},
		Priority:     framework.PriorityGeneric,
	}
}

// JUnit4ContentMatcher matches JUnit 4 specific patterns.
type JUnit4ContentMatcher struct{}

var junit4Patterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	{regexp.MustCompile(`@Test\s*(?:\(|$|\n)`), "@Test annotation"},
	{regexp.MustCompile(`@Before\s*(?:\(|$|\n)`), "@Before annotation"},
	{regexp.MustCompile(`@After\s*(?:\(|$|\n)`), "@After annotation"},
	{regexp.MustCompile(`@BeforeClass`), "@BeforeClass annotation"},
	{regexp.MustCompile(`@AfterClass`), "@AfterClass annotation"},
	{regexp.MustCompile(`@Ignore`), "@Ignore annotation"},
	{regexp.MustCompile(`@RunWith`), "@RunWith annotation"},
}

func (m *JUnit4ContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileContent {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		content = []byte(signal.Value)
	}

	// Exclude JUnit 5 files: has org.junit.jupiter import
	if javaast.JUnit5ImportPattern.Match(content) {
		return framework.NoMatch()
	}

	// Must have JUnit 4 import to be considered JUnit 4
	if !javaast.JUnit4ImportPattern.Match(content) {
		return framework.NoMatch()
	}

	for _, p := range junit4Patterns {
		if p.pattern.Match(content) {
			return framework.PartialMatch(40, "Found JUnit 4 pattern: "+p.desc)
		}
	}

	return framework.NoMatch()
}

// JUnit4Parser extracts test definitions from Java JUnit 4 files.
type JUnit4Parser struct{}

func (p *JUnit4Parser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	tree, err := parser.ParseWithPool(ctx, domain.LanguageJava, source)
	if err != nil {
		return nil, fmt.Errorf("junit4 parser: failed to parse %s: %w", filename, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	suites := parseTestClasses(root, source, filename)

	return &domain.TestFile{
		Path:      filename,
		Language:  domain.LanguageJava,
		Framework: framework.FrameworkJUnit4,
		Suites:    suites,
	}, nil
}

func parseTestClasses(root *sitter.Node, source []byte, filename string) []domain.TestSuite {
	var suites []domain.TestSuite

	parser.WalkTree(root, func(node *sitter.Node) bool {
		if node.Type() == javaast.NodeClassDeclaration {
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
	className := javaast.GetClassName(node, source)
	if className == "" {
		return nil
	}

	modifiers := javaast.GetModifiers(node)
	classStatus, classModifier := getClassStatusAndModifier(modifiers, source)
	runWith := getRunWithValue(modifiers, source)

	body := javaast.GetClassBody(node)
	if body == nil {
		return nil
	}

	var tests []domain.Test

	for i := 0; i < int(body.ChildCount()); i++ {
		child := body.Child(i)

		if child.Type() == javaast.NodeMethodDeclaration {
			if test := parseTestMethod(child, source, filename, classStatus, classModifier); test != nil {
				tests = append(tests, *test)
			}
		}
	}

	if len(tests) == 0 {
		return nil
	}

	suite := &domain.TestSuite{
		Name:     className,
		Status:   classStatus,
		Modifier: classModifier,
		Location: parser.GetLocation(node, filename),
		Tests:    tests,
	}

	if runWith != "" {
		suite.Modifier = "@RunWith(" + runWith + ")"
	}

	return suite
}

func parseTestMethod(node *sitter.Node, source []byte, filename string, classStatus domain.TestStatus, classModifier string) *domain.Test {
	modifiers := javaast.GetModifiers(node)
	if modifiers == nil {
		return nil
	}

	annotations := javaast.GetAnnotations(modifiers)
	isTest := false
	status := classStatus
	modifier := classModifier

	for _, ann := range annotations {
		name := javaast.GetAnnotationName(ann, source)

		switch name {
		case "Test":
			isTest = true
		case "Ignore":
			status = domain.TestStatusSkipped
			modifier = "@Ignore"
		}
	}

	if !isTest {
		return nil
	}

	methodName := javaast.GetMethodName(node, source)
	if methodName == "" {
		return nil
	}

	return &domain.Test{
		Name:     methodName,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(node, filename),
	}
}

func getClassStatusAndModifier(modifiers *sitter.Node, source []byte) (domain.TestStatus, string) {
	if modifiers == nil {
		return domain.TestStatusActive, ""
	}

	if javaast.HasAnnotation(modifiers, source, "Ignore") {
		return domain.TestStatusSkipped, "@Ignore"
	}

	return domain.TestStatusActive, ""
}

func getRunWithValue(modifiers *sitter.Node, source []byte) string {
	if modifiers == nil {
		return ""
	}

	annotations := javaast.GetAnnotations(modifiers)
	for _, ann := range annotations {
		name := javaast.GetAnnotationName(ann, source)
		if name == "RunWith" {
			return javaast.GetAnnotationArgument(ann, source)
		}
	}

	return ""
}
