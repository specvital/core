package pytest

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

const frameworkName = "pytest"

func init() {
	framework.Register(NewDefinition())
}

func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      frameworkName,
		Languages: []domain.Language{domain.LanguagePython},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("pytest"),
			matchers.NewConfigMatcher(
				"pytest.ini",
				"conftest.py",
			),
			&PytestConfigContentMatcher{},
			&PytestFileMatcher{},
			&PytestContentMatcher{},
		},
		ConfigParser: nil, // TODO: implement pytest config parsing if needed
		Parser:       &PytestParser{},
		Priority:     framework.PriorityGeneric,
	}
}

// PytestConfigContentMatcher matches pyproject.toml with [tool.pytest] section.
type PytestConfigContentMatcher struct{}

func (m *PytestConfigContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalConfigFile {
		return framework.NoMatch()
	}

	if !strings.HasSuffix(signal.Value, "pyproject.toml") {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		return framework.NoMatch()
	}

	if pytestSectionPattern.Match(content) {
		return framework.DefiniteMatch("pyproject.toml contains [tool.pytest] section")
	}

	return framework.NoMatch()
}

var pytestSectionPattern = regexp.MustCompile(`\[tool\.pytest`)

// PytestFileMatcher matches test_*.py and *_test.py files.
type PytestFileMatcher struct{}

func (m *PytestFileMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileName {
		return framework.NoMatch()
	}

	filename := signal.Value
	base := filename
	if idx := strings.LastIndex(filename, "/"); idx >= 0 {
		base = filename[idx+1:]
	}

	if strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py") {
		return framework.PartialMatch(20, "pytest file naming: test_*.py")
	}

	if strings.HasSuffix(base, "_test.py") {
		return framework.PartialMatch(20, "pytest file naming: *_test.py")
	}

	return framework.NoMatch()
}

// PytestContentMatcher matches pytest-specific patterns.
type PytestContentMatcher struct{}

var pytestPatterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	{regexp.MustCompile(`@pytest\.mark\.\w+`), "@pytest.mark decorator"},
	{regexp.MustCompile(`@pytest\.fixture`), "@pytest.fixture decorator"},
	{regexp.MustCompile(`pytest\.raises\s*\(`), "pytest.raises()"},
	{regexp.MustCompile(`pytest\.warns\s*\(`), "pytest.warns()"},
	{regexp.MustCompile(`pytest\.param\s*\(`), "pytest.param()"},
}

func (m *PytestContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileContent {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		content = []byte(signal.Value)
	}

	for _, p := range pytestPatterns {
		if p.pattern.Match(content) {
			return framework.PartialMatch(40, "Found pytest pattern: "+p.desc)
		}
	}

	return framework.NoMatch()
}

// PytestParser extracts test definitions from Python pytest files.
type PytestParser struct{}

func (p *PytestParser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	tree, err := parser.ParseWithPool(ctx, domain.LanguagePython, source)
	if err != nil {
		return nil, fmt.Errorf("pytest parser: failed to parse %s: %w", filename, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	suites, tests := parseTestModule(root, source, filename)

	return &domain.TestFile{
		Path:      filename,
		Language:  domain.LanguagePython,
		Framework: frameworkName,
		Suites:    suites,
		Tests:     tests,
	}, nil
}

func parseTestModule(root *sitter.Node, source []byte, filename string) ([]domain.TestSuite, []domain.Test) {
	var suites []domain.TestSuite
	var tests []domain.Test

	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)

		switch child.Type() {
		case pyast.NodeFunctionDefinition:
			if test := parseTestFunction(child, source, filename); test != nil {
				tests = append(tests, *test)
			}

		case pyast.NodeClassDefinition:
			if suite := parseTestClass(child, source, filename); suite != nil {
				suites = append(suites, *suite)
			}

		case pyast.NodeDecoratedDefinition:
			// Handle decorated functions and classes
			definition := pyast.GetDecoratedDefinition(child)
			if definition == nil {
				continue
			}

			decorators := pyast.GetDecorators(child)
			status := getStatusFromDecorators(decorators, source)

			switch definition.Type() {
			case pyast.NodeFunctionDefinition:
				if test := parseTestFunctionWithStatus(definition, source, filename, status); test != nil {
					tests = append(tests, *test)
				}
			case pyast.NodeClassDefinition:
				if suite := parseTestClassWithStatus(definition, source, filename, status); suite != nil {
					suites = append(suites, *suite)
				}
			}
		}
	}

	return suites, tests
}

func parseTestFunction(node *sitter.Node, source []byte, filename string) *domain.Test {
	return parseTestFunctionWithStatus(node, source, filename, domain.TestStatusActive)
}

func parseTestFunctionWithStatus(node *sitter.Node, source []byte, filename string, status domain.TestStatus) *domain.Test {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := parser.GetNodeText(nameNode, source)
	if !isTestFunction(name) {
		return nil
	}

	return &domain.Test{
		Name:     name,
		Status:   status,
		Location: parser.GetLocation(node, filename),
	}
}

func parseTestClass(node *sitter.Node, source []byte, filename string) *domain.TestSuite {
	return parseTestClassWithStatus(node, source, filename, domain.TestStatusActive)
}

func parseTestClassWithStatus(node *sitter.Node, source []byte, filename string, classStatus domain.TestStatus) *domain.TestSuite {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := parser.GetNodeText(nameNode, source)
	if !isTestClass(name) {
		return nil
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
			if test := parseTestFunction(child, source, filename); test != nil {
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

			if test := parseTestFunctionWithStatus(definition, source, filename, status); test != nil {
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

func getStatusFromDecorators(decorators []*sitter.Node, source []byte) domain.TestStatus {
	for _, dec := range decorators {
		text := parser.GetNodeText(dec, source)

		switch {
		case strings.Contains(text, "pytest.mark.skip"):
			return domain.TestStatusSkipped
		case strings.Contains(text, "pytest.mark.xfail"):
			return domain.TestStatusXfail
		}
	}
	return domain.TestStatusActive
}

func isTestFunction(name string) bool {
	// pytest convention: test_* is the standard pattern
	return strings.HasPrefix(name, "test_")
}

func isTestClass(name string) bool {
	return strings.HasPrefix(name, "Test")
}
