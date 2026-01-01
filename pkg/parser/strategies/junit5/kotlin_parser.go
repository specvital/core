package junit5

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser"
	"github.com/specvital/core/pkg/parser/framework"
	"github.com/specvital/core/pkg/parser/strategies/shared/kotlinast"
)

// KotlinTestFileMatcher matches Kotlin JUnit5 test file naming patterns.
type KotlinTestFileMatcher struct{}

func (m *KotlinTestFileMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileName {
		return framework.NoMatch()
	}

	filename := signal.Value

	// Exclude src/main/ (production code in Maven/Gradle structure)
	if strings.Contains(filename, "/src/main/") || strings.HasPrefix(filename, "src/main/") {
		return framework.NoMatch()
	}

	base := filename
	if idx := strings.LastIndex(filename, "/"); idx >= 0 {
		base = filename[idx+1:]
	}

	if !strings.HasSuffix(base, ".kt") {
		return framework.NoMatch()
	}

	name := strings.TrimSuffix(base, ".kt")

	if strings.HasSuffix(name, "Test") || strings.HasSuffix(name, "Tests") {
		return framework.PartialMatch(20, "JUnit5 Kotlin file naming: *Test.kt")
	}

	if strings.HasPrefix(name, "Test") {
		return framework.PartialMatch(20, "JUnit5 Kotlin file naming: Test*.kt")
	}

	return framework.NoMatch()
}

// parseKotlinTestFile parses a Kotlin JUnit5 test file.
func parseKotlinTestFile(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	// Sanitize source to handle NULL bytes that would cause tree-sitter parsing failures
	cleanSource := kotlinast.SanitizeSource(source)

	tree, err := parser.ParseWithPool(ctx, domain.LanguageKotlin, cleanSource)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	root := tree.RootNode()
	suites := parseKotlinTestClasses(root, cleanSource, filename)

	return &domain.TestFile{
		Path:      filename,
		Language:  domain.LanguageKotlin,
		Framework: framework.FrameworkJUnit5,
		Suites:    suites,
	}, nil
}

func parseKotlinTestClasses(root *sitter.Node, source []byte, filename string) []domain.TestSuite {
	var suites []domain.TestSuite

	parser.WalkTree(root, func(node *sitter.Node) bool {
		if node.Type() == kotlinast.NodeClassDeclaration {
			if suite := parseKotlinTestClass(node, source, filename); suite != nil {
				suites = append(suites, *suite)
			}
			return false
		}
		return true
	})

	return suites
}

func parseKotlinTestClass(node *sitter.Node, source []byte, filename string) *domain.TestSuite {
	className := kotlinast.GetClassName(node, source)
	if className == "" {
		return nil
	}

	// Skip if this is a Kotest spec (handled by kotest parser)
	isSpec, _ := kotlinast.IsKotestSpec(node, source)
	if isSpec {
		return nil
	}

	modifiers := kotlinast.GetModifiers(node)
	classStatus, classModifier := getKotlinClassStatus(modifiers, source)

	body := kotlinast.GetClassBody(node)
	if body == nil {
		return nil
	}

	var tests []domain.Test

	for i := 0; i < int(body.ChildCount()); i++ {
		child := body.Child(i)
		if child.Type() == kotlinast.NodeFunctionDeclaration {
			if test := parseKotlinTestMethod(child, source, filename, classStatus, classModifier); test != nil {
				tests = append(tests, *test)
			}
		}
	}

	if len(tests) == 0 {
		return nil
	}

	return &domain.TestSuite{
		Name:     className,
		Status:   classStatus,
		Modifier: classModifier,
		Location: parser.GetLocation(node, filename),
		Tests:    tests,
	}
}

func parseKotlinTestMethod(node *sitter.Node, source []byte, filename string, classStatus domain.TestStatus, classModifier string) *domain.Test {
	modifiers := kotlinast.GetModifiers(node)

	// Check for test annotations
	isTest := kotlinast.HasAnnotation(modifiers, source, "Test") ||
		kotlinast.HasAnnotation(modifiers, source, "ParameterizedTest") ||
		kotlinast.HasAnnotation(modifiers, source, "RepeatedTest") ||
		kotlinast.HasAnnotation(modifiers, source, "TestFactory") ||
		kotlinast.HasAnnotation(modifiers, source, "TestTemplate") ||
		hasCustomTestAnnotation(modifiers, source)

	if !isTest {
		return nil
	}

	status := classStatus
	modifier := classModifier

	// Check for Disabled annotation
	if kotlinast.HasAnnotation(modifiers, source, "Disabled") ||
		kotlinast.HasAnnotation(modifiers, source, "Ignore") {
		status = domain.TestStatusSkipped
		modifier = "@Disabled"
	}

	methodName := getKotlinFunctionName(node, source)
	if methodName == "" {
		return nil
	}

	// Check for DisplayName annotation
	displayName := getDisplayNameFromAnnotations(modifiers, source)
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

func hasCustomTestAnnotation(modifiers *sitter.Node, source []byte) bool {
	if modifiers == nil {
		return false
	}
	for i := 0; i < int(modifiers.ChildCount()); i++ {
		child := modifiers.Child(i)
		if child.Type() == kotlinast.NodeAnnotation {
			name := kotlinast.GetAnnotationName(child, source)
			// Custom test annotations typically end with "Test"
			if strings.HasSuffix(name, "Test") &&
				name != "Test" &&
				name != "ParameterizedTest" &&
				name != "RepeatedTest" &&
				name != "TestFactory" &&
				name != "TestTemplate" {
				return true
			}
		}
	}
	return false
}

func getDisplayNameFromAnnotations(modifiers *sitter.Node, source []byte) string {
	if modifiers == nil {
		return ""
	}
	for i := 0; i < int(modifiers.ChildCount()); i++ {
		child := modifiers.Child(i)
		if child.Type() == kotlinast.NodeAnnotation {
			name := kotlinast.GetAnnotationName(child, source)
			if name == "DisplayName" {
				return getKotlinAnnotationArgument(child, source)
			}
		}
	}
	return ""
}

func getKotlinClassStatus(modifiers *sitter.Node, source []byte) (domain.TestStatus, string) {
	if modifiers == nil {
		return domain.TestStatusActive, ""
	}

	if kotlinast.HasAnnotation(modifiers, source, "Disabled") ||
		kotlinast.HasAnnotation(modifiers, source, "Ignore") {
		return domain.TestStatusSkipped, "@Disabled"
	}

	return domain.TestStatusActive, ""
}

func getKotlinFunctionName(node *sitter.Node, source []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == kotlinast.NodeIdentifier {
			name := child.Content(source)
			// Handle backtick-escaped names like `1 + 1 = 2`
			if strings.HasPrefix(name, "`") && strings.HasSuffix(name, "`") {
				return name[1 : len(name)-1]
			}
			return name
		}
	}
	return ""
}

func getKotlinAnnotationArgument(ann *sitter.Node, source []byte) string {
	for i := 0; i < int(ann.ChildCount()); i++ {
		child := ann.Child(i)
		if child.Type() == kotlinast.NodeValueArguments {
			for j := 0; j < int(child.ChildCount()); j++ {
				arg := child.Child(j)
				if arg.Type() == kotlinast.NodeValueArgument {
					return extractKotlinStringValue(arg, source)
				}
			}
		}
	}
	return ""
}

func extractKotlinStringValue(node *sitter.Node, source []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == kotlinast.NodeStringLiteral || child.Type() == kotlinast.NodeLineStringLiteral {
			return kotlinast.ExtractStringContent(child, source)
		}
	}
	return ""
}
