// Package xunit implements xUnit test framework support for C# test files.
package xunit

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

const frameworkName = "xunit"

func init() {
	framework.Register(NewDefinition())
}

func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      frameworkName,
		Languages: []domain.Language{domain.LanguageCSharp},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher(
				"Xunit",
				"using Xunit",
			),
			&XUnitFileMatcher{},
			&XUnitContentMatcher{},
		},
		ConfigParser: nil,
		Parser:       &XUnitParser{},
		Priority:     framework.PriorityGeneric,
	}
}

// XUnitFileMatcher matches *Test.cs, *Tests.cs, Test*.cs, *Spec.cs, *Specs.cs files.
type XUnitFileMatcher struct{}

func (m *XUnitFileMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileName {
		return framework.NoMatch()
	}

	if dotnetast.IsCSharpTestFileName(signal.Value) {
		return framework.PartialMatch(20, "xUnit file naming convention")
	}

	return framework.NoMatch()
}

// XUnitContentMatcher matches xUnit specific patterns.
type XUnitContentMatcher struct{}

var xunitPatterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	{regexp.MustCompile(`\[Fact\]`), "[Fact] attribute"},
	{regexp.MustCompile(`\[Theory\]`), "[Theory] attribute"},
	{regexp.MustCompile(`\[InlineData\(`), "[InlineData] attribute"},
	{regexp.MustCompile(`\[MemberData\(`), "[MemberData] attribute"},
	{regexp.MustCompile(`\[ClassData\(`), "[ClassData] attribute"},
	{regexp.MustCompile(`using\s+Xunit\s*;`), "using Xunit"},
	{regexp.MustCompile(`\[Skip\(`), "[Skip] attribute"},
}

func (m *XUnitContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileContent {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		content = []byte(signal.Value)
	}

	for _, p := range xunitPatterns {
		if p.pattern.Match(content) {
			return framework.PartialMatch(40, "Found xUnit pattern: "+p.desc)
		}
	}

	return framework.NoMatch()
}

// XUnitParser extracts test definitions from C# xUnit files.
type XUnitParser struct{}

func (p *XUnitParser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	tree, err := parser.ParseWithPool(ctx, domain.LanguageCSharp, source)
	if err != nil {
		return nil, fmt.Errorf("xunit parser: failed to parse %s: %w", filename, err)
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
// C# allows unlimited class nesting, but 20 levels provides a safe buffer
// (real-world maximum observed: 3 in FluentAssertions test suite).
const maxNestedDepth = 20

func getClassStatus(attrLists []*sitter.Node, source []byte) domain.TestStatus {
	for _, attr := range dotnetast.GetAttributes(attrLists) {
		name := dotnetast.GetAttributeName(attr, source)
		if name == "Skip" || name == "SkipAttribute" {
			return domain.TestStatusSkipped
		}
	}
	return domain.TestStatusActive
}

// getDisplayNameFromAttribute extracts DisplayName from [Fact(DisplayName = "...")] or [Theory(DisplayName = "...")].
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

// isSkipped checks if the attribute has Skip parameter set.
func isSkipped(attr *sitter.Node, source []byte) bool {
	argList := dotnetast.FindAttributeArgumentList(attr)
	if argList == nil {
		return false
	}

	for i := 0; i < int(argList.ChildCount()); i++ {
		arg := argList.Child(i)
		if arg.Type() == dotnetast.NodeAttributeArgument {
			name, _ := dotnetast.ParseAssignmentExpression(arg, source)
			if name == "Skip" {
				return true
			}
		}
	}
	return false
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
	classStatus := getClassStatus(attrLists, source)

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
			if test := parseTestMethod(child, source, filename, classStatus); test != nil {
				tests = append(tests, *test)
			}

		case dotnetast.NodeClassDeclaration:
			// xUnit automatically discovers nested classes without annotation.
			// Unlike JUnit5's @Nested requirement, any nested class can contain test methods.
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
		Location: parser.GetLocation(node, filename),
		Tests:    tests,
		Suites:   nestedSuites,
	}
}

func parseTestMethod(node *sitter.Node, source []byte, filename string, classStatus domain.TestStatus) *domain.Test {
	attrLists := dotnetast.GetAttributeLists(node)
	if len(attrLists) == 0 {
		return nil
	}

	attributes := dotnetast.GetAttributes(attrLists)
	isTest := false
	status := classStatus
	var displayName string

	for _, attr := range attributes {
		name := dotnetast.GetAttributeName(attr, source)

		switch name {
		case "Fact", "FactAttribute":
			isTest = true
			displayName = getDisplayNameFromAttribute(attr, source)
			if isSkipped(attr, source) {
				status = domain.TestStatusSkipped
			}
		case "Theory", "TheoryAttribute":
			isTest = true
			displayName = getDisplayNameFromAttribute(attr, source)
			if isSkipped(attr, source) {
				status = domain.TestStatusSkipped
			}
		}
	}

	if !isTest {
		return nil
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
		Location: parser.GetLocation(node, filename),
	}
}
