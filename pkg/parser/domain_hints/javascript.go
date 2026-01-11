package domain_hints

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/tspool"
)

// JavaScriptExtractor extracts domain hints from JavaScript/TypeScript source code.
type JavaScriptExtractor struct {
	lang domain.Language
}

const (
	// ES6 imports: import x from 'y', import { x } from 'y', import 'y'
	jsImportQuery = `
		(import_statement
			source: (string) @import
		)
	`

	// CommonJS: require('x'), require("x")
	jsRequireQuery = `
		(call_expression
			function: (identifier) @func (#eq? @func "require")
			arguments: (arguments (string) @import)
		)
	`

	// Function calls: obj.method(), func()
	jsCallQuery = `
		(call_expression
			function: [
				(identifier) @call
				(member_expression) @call
			]
		)
	`

	// Variable declarations: const x = ..., let y = ..., var z = ...
	jsVariableQuery = `
		(variable_declarator
			name: (identifier) @var
		)
	`
)


func (e *JavaScriptExtractor) Extract(ctx context.Context, source []byte) *domain.DomainHints {
	tree, err := tspool.Parse(ctx, e.lang, source)
	if err != nil {
		return nil
	}
	defer tree.Close()

	root := tree.RootNode()

	hints := &domain.DomainHints{
		Imports:   e.extractImports(root, source),
		Calls:     e.extractCalls(root, source),
		Variables: e.extractVariables(root, source),
	}

	if len(hints.Imports) == 0 && len(hints.Calls) == 0 && len(hints.Variables) == 0 {
		return nil
	}

	return hints
}

// extractImports extracts both ES6 and CommonJS imports.
// Uses best-effort extraction: query errors are ignored to allow partial results
// when one import style fails (e.g., ES6 query fails but CommonJS succeeds).
func (e *JavaScriptExtractor) extractImports(root *sitter.Node, source []byte) []string {
	seen := make(map[string]struct{})
	var imports []string

	// ES6 imports (errors ignored for best-effort extraction)
	es6Results, err := tspool.QueryWithCache(root, source, e.lang, jsImportQuery)
	if err == nil {
		for _, r := range es6Results {
			if node, ok := r.Captures["import"]; ok {
				path := trimJSQuotes(getNodeText(node, source))
				if path != "" && !isTypeOnlyImportNode(node) {
					if _, exists := seen[path]; !exists {
						seen[path] = struct{}{}
						imports = append(imports, path)
					}
				}
			}
		}
	}

	// CommonJS require (errors ignored for best-effort extraction)
	requireResults, err := tspool.QueryWithCache(root, source, e.lang, jsRequireQuery)
	if err == nil {
		for _, r := range requireResults {
			if node, ok := r.Captures["import"]; ok {
				path := trimJSQuotes(getNodeText(node, source))
				if path != "" {
					if _, exists := seen[path]; !exists {
						seen[path] = struct{}{}
						imports = append(imports, path)
					}
				}
			}
		}
	}

	return imports
}

func (e *JavaScriptExtractor) extractCalls(root *sitter.Node, source []byte) []string {
	results, err := tspool.QueryWithCache(root, source, e.lang, jsCallQuery)
	if err != nil {
		return nil
	}

	seen := make(map[string]struct{})
	calls := make([]string, 0, len(results))

	for _, r := range results {
		if node, ok := r.Captures["call"]; ok {
			call := getNodeText(node, source)
			if call == "" {
				continue
			}
			// Skip require() calls as they're already captured in imports
			if call == "require" {
				continue
			}
			// Skip test framework calls
			if isTestFrameworkCall(call) {
				continue
			}
			if _, exists := seen[call]; exists {
				continue
			}
			seen[call] = struct{}{}
			calls = append(calls, call)
		}
	}

	return calls
}

func (e *JavaScriptExtractor) extractVariables(root *sitter.Node, source []byte) []string {
	results, err := tspool.QueryWithCache(root, source, e.lang, jsVariableQuery)
	if err != nil {
		return nil
	}

	seen := make(map[string]struct{})
	variables := make([]string, 0)

	for _, r := range results {
		if node, ok := r.Captures["var"]; ok {
			name := getNodeText(node, source)
			if name == "" || name == "_" {
				continue
			}
			if !domainVariablePattern.MatchString(name) {
				continue
			}
			if _, exists := seen[name]; exists {
				continue
			}
			seen[name] = struct{}{}
			variables = append(variables, name)
		}
	}

	return variables
}

func trimJSQuotes(s string) string {
	if len(s) < 2 {
		return s
	}
	first := s[0]
	last := s[len(s)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') || (first == '`' && last == '`') {
		return s[1 : len(s)-1]
	}
	return s
}

func isTypeOnlyImportNode(node *sitter.Node) bool {
	parent := node.Parent()
	if parent == nil {
		return false
	}
	// Check if import_statement has "type" keyword
	if parent.Type() == "import_statement" {
		for i := 0; i < int(parent.ChildCount()); i++ {
			child := parent.Child(i)
			if child != nil && child.Type() == "type" {
				return true
			}
		}
	}
	return false
}

// testFrameworkCalls contains base function names from common test frameworks
// that should be excluded from domain hints. These calls don't provide
// meaningful domain classification signals. Method calls like test.describe()
// or expect().toBe() are filtered by checking the base name before the dot.
var testFrameworkCalls = map[string]struct{}{
	"describe": {}, "it": {}, "test": {}, "expect": {},
	"beforeEach": {}, "afterEach": {}, "beforeAll": {}, "afterAll": {},
	"vi": {}, "jest": {}, "cy": {},
}

func isTestFrameworkCall(call string) bool {
	baseName := call
	if idx := strings.Index(call, "."); idx > 0 {
		baseName = call[:idx]
	}
	_, exists := testFrameworkCalls[baseName]
	return exists
}
