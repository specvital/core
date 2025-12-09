package unittest

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
	"github.com/specvital/core/pkg/parser/strategies/shared/pyast"
)

const frameworkName = "unittest"

func init() {
	framework.Register(NewDefinition())
}

func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      frameworkName,
		Languages: []domain.Language{domain.LanguagePython},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("unittest"),
			&UnittestFileMatcher{},
			&UnittestContentMatcher{},
		},
		ConfigParser: nil,
		Parser:       &UnittestParser{},
		Priority:     framework.PriorityGeneric,
	}
}

// UnittestFileMatcher matches test_*.py and *_test.py files.
type UnittestFileMatcher struct{}

func (m *UnittestFileMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileName {
		return framework.NoMatch()
	}

	filename := signal.Value
	base := filename
	if idx := strings.LastIndex(filename, "/"); idx >= 0 {
		base = filename[idx+1:]
	}

	if strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py") {
		return framework.PartialMatch(20, "unittest file naming: test_*.py")
	}

	if strings.HasSuffix(base, "_test.py") {
		return framework.PartialMatch(20, "unittest file naming: *_test.py")
	}

	return framework.NoMatch()
}

// UnittestContentMatcher matches unittest-specific patterns.
type UnittestContentMatcher struct{}

var unittestPatterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	{regexp.MustCompile(`unittest\.TestCase`), "unittest.TestCase inheritance"},
	{regexp.MustCompile(`@unittest\.(skip|skipIf|skipUnless|expectedFailure)`), "unittest skip decorator"},
	{regexp.MustCompile(`self\.assert\w+\s*\(`), "self.assert* assertion"},
	{regexp.MustCompile(`unittest\.main\s*\(`), "unittest.main()"},
}

func (m *UnittestContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileContent {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		content = []byte(signal.Value)
	}

	for _, p := range unittestPatterns {
		if p.pattern.Match(content) {
			return framework.PartialMatch(40, "Found unittest pattern: "+p.desc)
		}
	}

	return framework.NoMatch()
}

// UnittestParser extracts test definitions from Python unittest files.
type UnittestParser struct{}

func (p *UnittestParser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	tree, err := parser.ParseWithPool(ctx, domain.LanguagePython, source)
	if err != nil {
		return nil, fmt.Errorf("unittest parser: failed to parse %s: %w", filename, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	suites := parseTestClasses(root, source, filename)

	return &domain.TestFile{
		Path:      filename,
		Language:  domain.LanguagePython,
		Framework: frameworkName,
		Suites:    suites,
	}, nil
}

func parseTestClasses(root *sitter.Node, source []byte, filename string) []domain.TestSuite {
	var suites []domain.TestSuite

	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)

		switch child.Type() {
		case pyast.NodeClassDefinition:
			if suite := parseTestCaseClass(child, source, filename); suite != nil {
				suites = append(suites, *suite)
			}

		case pyast.NodeDecoratedDefinition:
			definition := pyast.GetDecoratedDefinition(child)
			if definition == nil || definition.Type() != pyast.NodeClassDefinition {
				continue
			}

			decorators := pyast.GetDecorators(child)
			status := getStatusFromDecorators(decorators, source)

			if suite := parseTestCaseClassWithStatus(definition, source, filename, status); suite != nil {
				suites = append(suites, *suite)
			}
		}
	}

	return suites
}

func parseTestCaseClass(node *sitter.Node, source []byte, filename string) *domain.TestSuite {
	return parseTestCaseClassWithStatus(node, source, filename, domain.TestStatusActive)
}

func parseTestCaseClassWithStatus(node *sitter.Node, source []byte, filename string, classStatus domain.TestStatus) *domain.TestSuite {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := parser.GetNodeText(nameNode, source)

	// Check if class inherits from TestCase or has Test prefix/suffix
	if !isTestCaseClass(node, source) {
		if !strings.HasPrefix(name, "Test") && !strings.HasSuffix(name, "Test") {
			return nil
		}
	}

	body := node.ChildByFieldName("body")
	if body == nil {
		return nil
	}

	var tests []domain.Test
	for i := 0; i < int(body.ChildCount()); i++ {
		child := body.Child(i)

		switch child.Type() {
		case pyast.NodeFunctionDefinition:
			if test := parseTestMethod(child, source, filename); test != nil {
				// Inherit class status if method has default (active) status
				if test.Status == domain.TestStatusActive && classStatus != domain.TestStatusActive {
					test.Status = classStatus
				}
				tests = append(tests, *test)
			}

		case pyast.NodeDecoratedDefinition:
			definition := pyast.GetDecoratedDefinition(child)
			if definition == nil || definition.Type() != pyast.NodeFunctionDefinition {
				continue
			}

			decorators := pyast.GetDecorators(child)
			status := getStatusFromDecorators(decorators, source)
			// Inherit class status if method has default (active) status
			if status == domain.TestStatusActive && classStatus != domain.TestStatusActive {
				status = classStatus
			}

			if test := parseTestMethodWithStatus(definition, source, filename, status); test != nil {
				tests = append(tests, *test)
			}
		}
	}

	if len(tests) == 0 {
		return nil
	}

	return &domain.TestSuite{
		Name:     name,
		Status:   classStatus,
		Location: parser.GetLocation(node, filename),
		Tests:    tests,
	}
}

func parseTestMethod(node *sitter.Node, source []byte, filename string) *domain.Test {
	return parseTestMethodWithStatus(node, source, filename, domain.TestStatusActive)
}

func parseTestMethodWithStatus(node *sitter.Node, source []byte, filename string, status domain.TestStatus) *domain.Test {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := parser.GetNodeText(nameNode, source)
	if !strings.HasPrefix(name, "test") {
		return nil
	}

	return &domain.Test{
		Name:     name,
		Status:   status,
		Location: parser.GetLocation(node, filename),
	}
}

func isTestCaseClass(node *sitter.Node, source []byte) bool {
	superclassNode := node.ChildByFieldName("superclasses")
	if superclassNode == nil {
		return false
	}

	text := parser.GetNodeText(superclassNode, source)
	return strings.Contains(text, "TestCase") || strings.Contains(text, "unittest.TestCase")
}

func getStatusFromDecorators(decorators []*sitter.Node, source []byte) domain.TestStatus {
	for _, dec := range decorators {
		text := parser.GetNodeText(dec, source)

		switch {
		case strings.Contains(text, "unittest.skip"):
			return domain.TestStatusSkipped
		case strings.Contains(text, "unittest.skipIf"):
			return domain.TestStatusSkipped
		case strings.Contains(text, "unittest.skipUnless"):
			return domain.TestStatusSkipped
		case strings.Contains(text, "unittest.expectedFailure"):
			return domain.TestStatusXfail
		}
	}
	return domain.TestStatusActive
}
