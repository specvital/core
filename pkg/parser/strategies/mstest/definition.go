// Package mstest implements MSTest framework support for C# test files.
package mstest

import (
	"context"
	"fmt"
	"regexp"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser"
	"github.com/specvital/core/pkg/parser/framework"
	"github.com/specvital/core/pkg/parser/framework/matchers"
	"github.com/specvital/core/pkg/parser/strategies/shared/dotnetast"
)

const frameworkName = "mstest"

func init() {
	framework.Register(NewDefinition())
}

func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      frameworkName,
		Languages: []domain.Language{domain.LanguageCSharp},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher(
				"Microsoft.VisualStudio.TestTools.UnitTesting",
				"using Microsoft.VisualStudio.TestTools.UnitTesting",
			),
			&MSTestFileMatcher{},
			&MSTestContentMatcher{},
		},
		ConfigParser: nil,
		Parser:       &MSTestParser{},
		Priority:     framework.PriorityGeneric,
	}
}

// MSTestFileMatcher matches *Test.cs, *Tests.cs, Test*.cs, *Spec.cs, *Specs.cs files.
type MSTestFileMatcher struct{}

func (m *MSTestFileMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileName {
		return framework.NoMatch()
	}

	if dotnetast.IsCSharpTestFileName(signal.Value) {
		return framework.PartialMatch(20, "MSTest file naming convention")
	}

	return framework.NoMatch()
}

// MSTestContentMatcher matches MSTest specific patterns.
type MSTestContentMatcher struct{}

var mstestPatterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	{regexp.MustCompile(`\[TestMethod\]`), "[TestMethod] attribute"},
	{regexp.MustCompile(`\[TestClass\]`), "[TestClass] attribute"},
	{regexp.MustCompile(`\[DataRow\(`), "[DataRow] attribute"},
	{regexp.MustCompile(`\[DataTestMethod\]`), "[DataTestMethod] attribute"},
	{regexp.MustCompile(`\[DynamicData\(`), "[DynamicData] attribute"},
	{regexp.MustCompile(`using\s+Microsoft\.VisualStudio\.TestTools\.UnitTesting\s*;`), "using Microsoft.VisualStudio.TestTools.UnitTesting"},
	{regexp.MustCompile(`\[Ignore\]`), "[Ignore] attribute"},
}

func (m *MSTestContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileContent {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		content = []byte(signal.Value)
	}

	for _, p := range mstestPatterns {
		if p.pattern.Match(content) {
			return framework.PartialMatch(40, "Found MSTest pattern: "+p.desc)
		}
	}

	return framework.NoMatch()
}

// MSTestParser extracts test definitions from C# MSTest files.
type MSTestParser struct{}

func (p *MSTestParser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	tree, err := parser.ParseWithPool(ctx, domain.LanguageCSharp, source)
	if err != nil {
		return nil, fmt.Errorf("mstest parser: failed to parse %s: %w", filename, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	suites := parseTestClasses(root, source, filename)

	return &domain.TestFile{
		Path:      filename,
		Language:  domain.LanguageCSharp,
		Framework: frameworkName,
		Suites:    suites,
	}, nil
}

// maxNestedDepth limits recursion depth for nested class parsing.
const maxNestedDepth = 20

func getClassStatusAndModifier(attrLists []*sitter.Node, source []byte) (domain.TestStatus, string) {
	for _, attr := range dotnetast.GetAttributes(attrLists) {
		name := dotnetast.GetAttributeName(attr, source)
		if name == "Ignore" || name == "IgnoreAttribute" {
			return domain.TestStatusSkipped, "[Ignore]"
		}
	}
	return domain.TestStatusActive, ""
}

// getDisplayNameFromAttribute extracts DisplayName from [TestMethod(DisplayName = "...")] or similar.
func getDisplayNameFromAttribute(attr *sitter.Node, source []byte) string {
	argList := dotnetast.FindAttributeArgumentList(attr)
	if argList == nil {
		return ""
	}

	for i := 0; i < int(argList.ChildCount()); i++ {
		arg := argList.Child(i)
		if arg.Type() == dotnetast.NodeAttributeArgument {
			name, value := dotnetast.ParseAssignmentExpression(arg, source)
			if name == "DisplayName" {
				return value
			}
		}
	}
	return ""
}

// isIgnoredWithModifier checks if [Ignore] or [IgnoreAttribute] is applied.
func isIgnoredWithModifier(attrLists []*sitter.Node, source []byte) (bool, string) {
	for _, attr := range dotnetast.GetAttributes(attrLists) {
		name := dotnetast.GetAttributeName(attr, source)
		if name == "Ignore" || name == "IgnoreAttribute" {
			return true, "[Ignore]"
		}
	}
	return false, ""
}

func parseTestClasses(root *sitter.Node, source []byte, filename string) []domain.TestSuite {
	var suites []domain.TestSuite

	parser.WalkTree(root, func(node *sitter.Node) bool {
		if node.Type() == dotnetast.NodeClassDeclaration {
			if suite := parseTestClassWithDepth(node, source, filename, 0); suite != nil {
				suites = append(suites, *suite)
			}
			return false
		}
		return true
	})

	return suites
}

func parseTestClassWithDepth(node *sitter.Node, source []byte, filename string, depth int) *domain.TestSuite {
	if depth > maxNestedDepth {
		return nil
	}

	className := dotnetast.GetClassName(node, source)
	if className == "" {
		return nil
	}

	attrLists := dotnetast.GetAttributeLists(node)
	classStatus, classModifier := getClassStatusAndModifier(attrLists, source)

	body := dotnetast.GetDeclarationList(node)
	if body == nil {
		return nil
	}

	var tests []domain.Test
	var nestedSuites []domain.TestSuite

	for i := 0; i < int(body.ChildCount()); i++ {
		child := body.Child(i)

		switch child.Type() {
		case dotnetast.NodeMethodDeclaration:
			if test := parseTestMethod(child, source, filename, classStatus, classModifier); test != nil {
				tests = append(tests, *test)
			}

		case dotnetast.NodeClassDeclaration:
			// MSTest discovers nested classes containing test methods, regardless of [TestClass] attribute.
			// The [TestClass] attribute is optional but commonly used for clarity.
			// Nested classes inherit parent's [Ignore] status through classStatus propagation.
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

func parseTestMethod(node *sitter.Node, source []byte, filename string, classStatus domain.TestStatus, classModifier string) *domain.Test {
	attrLists := dotnetast.GetAttributeLists(node)
	if len(attrLists) == 0 {
		return nil
	}

	attributes := dotnetast.GetAttributes(attrLists)
	isTest := false
	status := classStatus
	modifier := classModifier
	var displayName string

	for _, attr := range attributes {
		name := dotnetast.GetAttributeName(attr, source)

		switch name {
		case "TestMethod", "TestMethodAttribute":
			isTest = true
			displayName = getDisplayNameFromAttribute(attr, source)
		case "DataTestMethod", "DataTestMethodAttribute":
			isTest = true
			displayName = getDisplayNameFromAttribute(attr, source)
		}
	}

	if !isTest {
		return nil
	}

	if ignored, mod := isIgnoredWithModifier(attrLists, source); ignored {
		status = domain.TestStatusSkipped
		modifier = mod
	}

	methodName := dotnetast.GetMethodName(node, source)
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
