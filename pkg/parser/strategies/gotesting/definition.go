package gotesting

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser"
	"github.com/specvital/core/pkg/parser/framework"
	"github.com/specvital/core/pkg/parser/framework/matchers"
)

const (
	frameworkName                = "go-testing"
	nodeCallExpression           = "call_expression"
	nodeFunctionDeclaration      = "function_declaration"
	nodeParameterDeclaration     = "parameter_declaration"
	nodePointerType              = "pointer_type"
	nodeQualifiedType            = "qualified_type"
	nodeSelectorExpression       = "selector_expression"
	nodeInterpretedStringLiteral = "interpreted_string_literal"
	nodeRawStringLiteral         = "raw_string_literal"
	methodRun                    = "Run"
	typeTestingParam             = "testing.T"
)

func init() {
	framework.Register(NewDefinition())
}

func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      frameworkName,
		Languages: []domain.Language{domain.LanguageGo},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("testing"),
			&GoTestFileMatcher{},
		},
		ConfigParser: nil, // Go doesn't have config files
		Parser:       &GoTestingParser{},
		Priority:     framework.PriorityGeneric,
	}
}

// GoTestFileMatcher matches *_test.go files.
type GoTestFileMatcher struct{}

func (m *GoTestFileMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileName {
		return framework.NoMatch()
	}

	if strings.HasSuffix(signal.Value, "_test.go") {
		return framework.DefiniteMatch("Go test file naming convention: *_test.go")
	}

	return framework.NoMatch()
}

type GoTestingParser struct{}

func (p *GoTestingParser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	tree, err := parser.ParseWithPool(ctx, domain.LanguageGo, source)
	if err != nil {
		return nil, fmt.Errorf("go-testing parser: failed to parse %s: %w", filename, err)
	}
	defer tree.Close()
	root := tree.RootNode()

	suites, tests := parseTestFunctions(root, source, filename)

	testFile := &domain.TestFile{
		Path:      filename,
		Language:  domain.LanguageGo,
		Framework: frameworkName,
		Suites:    suites,
		Tests:     tests,
	}

	return testFile, nil
}

func extractSubtests(body *sitter.Node, source []byte, filename string) []domain.Test {
	var subtests []domain.Test

	parser.WalkTree(body, func(node *sitter.Node) bool {
		if node.Type() != nodeCallExpression {
			return true
		}

		funcNode := node.ChildByFieldName("function")
		if funcNode == nil || funcNode.Type() != nodeSelectorExpression {
			return true
		}

		field := funcNode.ChildByFieldName("field")
		if field == nil || parser.GetNodeText(field, source) != methodRun {
			return true
		}

		args := node.ChildByFieldName("arguments")
		if args == nil {
			return true
		}

		name := extractSubtestName(args, source)
		if name == "" {
			return true
		}

		subtests = append(subtests, domain.Test{
			Name:     name,
			Status:   domain.TestStatusActive,
			Location: parser.GetLocation(node, filename),
		})

		return true
	})

	return subtests
}

func extractSubtestName(args *sitter.Node, source []byte) string {
	for i := 0; i < int(args.ChildCount()); i++ {
		child := args.Child(i)
		switch child.Type() {
		case nodeInterpretedStringLiteral, nodeRawStringLiteral:
			return trimQuotes(parser.GetNodeText(child, source))
		}
	}
	return ""
}

func extractTestName(funcDecl *sitter.Node, source []byte) string {
	nameNode := funcDecl.ChildByFieldName("name")
	if nameNode == nil {
		return ""
	}
	return parser.GetNodeText(nameNode, source)
}

func isTestFunction(name string) bool {
	if !strings.HasPrefix(name, "Test") || len(name) <= 4 {
		return false
	}
	return unicode.IsUpper(rune(name[4]))
}

func parseTestFunctions(root *sitter.Node, source []byte, filename string) ([]domain.TestSuite, []domain.Test) {
	var suites []domain.TestSuite
	var tests []domain.Test

	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		if child.Type() != nodeFunctionDeclaration {
			continue
		}

		name := extractTestName(child, source)
		if !isTestFunction(name) {
			continue
		}

		if !validateTestParams(child, source) {
			continue
		}

		body := child.ChildByFieldName("body")
		var subtests []domain.Test
		if body != nil {
			subtests = extractSubtests(body, source, filename)
		}

		if len(subtests) > 0 {
			suite := domain.TestSuite{
				Name:     name,
				Status:   domain.TestStatusActive,
				Location: parser.GetLocation(child, filename),
				Tests:    subtests,
			}
			suites = append(suites, suite)
		} else {
			test := domain.Test{
				Name:     name,
				Status:   domain.TestStatusActive,
				Location: parser.GetLocation(child, filename),
			}
			tests = append(tests, test)
		}
	}

	return suites, tests
}

func trimQuotes(s string) string {
	if unquoted, err := strconv.Unquote(s); err == nil {
		return unquoted
	}
	// Fallback for invalid literals from incomplete code
	if len(s) >= 2 && s[0] == s[len(s)-1] && (s[0] == '"' || s[0] == '`') {
		return s[1 : len(s)-1]
	}
	return s
}

func validateTestParams(funcDecl *sitter.Node, source []byte) bool {
	params := funcDecl.ChildByFieldName("parameters")
	if params == nil {
		return false
	}

	var paramDecl *sitter.Node
	paramCount := 0
	for i := 0; i < int(params.ChildCount()); i++ {
		child := params.Child(i)
		if child.Type() == nodeParameterDeclaration {
			if paramCount == 0 {
				paramDecl = child
			}
			paramCount++
		}
	}

	if paramCount != 1 {
		return false
	}

	typeNode := paramDecl.ChildByFieldName("type")
	if typeNode == nil || typeNode.Type() != nodePointerType {
		return false
	}

	elem := parser.FindChildByType(typeNode, nodeQualifiedType)
	return elem != nil && parser.GetNodeText(elem, source) == typeTestingParam
}
