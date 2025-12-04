package gotesting

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser"
	"github.com/specvital/core/pkg/parser/strategies"
)

const (
	frameworkName = "go-testing"

	// AST node types
	nodeCallExpression       = "call_expression"
	nodeFunctionDeclaration  = "function_declaration"
	nodeParameterDeclaration = "parameter_declaration"
	nodePointerType          = "pointer_type"
	nodeQualifiedType        = "qualified_type"
	nodeSelectorExpression   = "selector_expression"

	// String literal types
	nodeInterpretedStringLiteral = "interpreted_string_literal"
	nodeRawStringLiteral         = "raw_string_literal"

	// Go test identifiers
	methodRun        = "Run"
	typeTestingParam = "testing.T"
)

type Strategy struct{}

func NewStrategy() *Strategy {
	return &Strategy{}
}

func RegisterDefault() {
	strategies.Register(NewStrategy())
}

func (s *Strategy) Name() string {
	return frameworkName
}

func (s *Strategy) Priority() int {
	return strategies.DefaultPriority
}

func (s *Strategy) Languages() []domain.Language {
	return []domain.Language{domain.LanguageGo}
}

func (s *Strategy) CanHandle(filename string, _ []byte) bool {
	return isGoTestFile(filename)
}

func (s *Strategy) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
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

// Helper functions (alphabetically ordered)

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
			Status:   "", // Go tests have no skip/only modifiers
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

func isGoTestFile(filename string) bool {
	base := filepath.Base(filename)
	return strings.HasSuffix(base, "_test.go")
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
				Status:   "", // Go tests have no skip/only modifiers
				Location: parser.GetLocation(child, filename),
				Tests:    subtests,
			}
			suites = append(suites, suite)
		} else {
			test := domain.Test{
				Name:     name,
				Status:   "", // Go tests have no skip/only modifiers
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
	// Fallback for invalid literals, e.g. from incomplete code.
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
