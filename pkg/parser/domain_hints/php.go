package domain_hints

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/strategies/shared/phpast"
	"github.com/specvital/core/pkg/parser/tspool"
)

// PHPExtractor extracts domain hints from PHP source code.
type PHPExtractor struct{}

const (
	// use Namespace\Class;
	phpUseQuery = `(namespace_use_declaration) @use`

	// require/include statements (each has a different node type in tree-sitter-php)
	phpIncludeQuery = `
		(include_expression) @include
		(include_once_expression) @include
		(require_expression) @include
		(require_once_expression) @include
	`

	// Method/function calls: $obj->method(), Class::staticMethod(), function()
	phpCallQuery = `
		(function_call_expression) @call
		(member_call_expression) @call
		(scoped_call_expression) @call
	`
)

func (e *PHPExtractor) Extract(ctx context.Context, source []byte) *domain.DomainHints {
	tree, err := tspool.Parse(ctx, domain.LanguagePHP, source)
	if err != nil {
		return nil
	}
	defer tree.Close()

	root := tree.RootNode()

	hints := &domain.DomainHints{
		Imports: e.extractImports(root, source),
		Calls:   e.extractCalls(root, source),
	}

	if len(hints.Imports) == 0 && len(hints.Calls) == 0 {
		return nil
	}

	return hints
}

func (e *PHPExtractor) extractImports(root *sitter.Node, source []byte) []string {
	seen := make(map[string]struct{})
	var imports []string

	// Extract 'use' statements
	useResults, err := tspool.QueryWithCache(root, source, domain.LanguagePHP, phpUseQuery)
	if err == nil {
		for _, r := range useResults {
			if node, ok := r.Captures["use"]; ok {
				path := extractPHPUsePath(node, source)
				if path != "" {
					if _, exists := seen[path]; !exists {
						seen[path] = struct{}{}
						imports = append(imports, path)
					}
				}
			}
		}
	}

	// Extract include/require statements
	includeResults, err := tspool.QueryWithCache(root, source, domain.LanguagePHP, phpIncludeQuery)
	if err == nil {
		for _, r := range includeResults {
			if node, ok := r.Captures["include"]; ok {
				path := extractPHPIncludePath(node, source)
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

// extractPHPUsePath extracts the namespace from a namespace_use_declaration node.
// Handles: use Namespace\Class; use Namespace\Class as Alias;
func extractPHPUsePath(node *sitter.Node, source []byte) string {
	// Look for namespace_use_clause children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "namespace_use_clause" {
			// Get the qualified_name or name from the clause
			for j := 0; j < int(child.ChildCount()); j++ {
				subChild := child.Child(j)
				switch subChild.Type() {
				case phpast.NodeQualifiedName, phpast.NodeName:
					return subChild.Content(source)
				}
			}
		}
	}
	return ""
}

// extractPHPIncludePath extracts the file path from include/require expressions.
func extractPHPIncludePath(node *sitter.Node, source []byte) string {
	// include_expression has children: include/require keyword + expression
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "string", "encapsed_string":
			return trimPHPQuotes(child.Content(source))
		case "parenthesized_expression":
			// require('path.php') syntax
			for j := 0; j < int(child.ChildCount()); j++ {
				inner := child.Child(j)
				if inner.Type() == "string" || inner.Type() == "encapsed_string" {
					return trimPHPQuotes(inner.Content(source))
				}
			}
		}
	}
	return ""
}

func (e *PHPExtractor) extractCalls(root *sitter.Node, source []byte) []string {
	results, err := tspool.QueryWithCache(root, source, domain.LanguagePHP, phpCallQuery)
	if err != nil {
		return nil
	}

	seen := make(map[string]struct{})
	calls := make([]string, 0, len(results))

	for _, r := range results {
		if node, ok := r.Captures["call"]; ok {
			call := extractPHPCall(node, source)
			if call == "" {
				continue
			}
			call = normalizeCall(call)
			if call == "" {
				continue
			}
			if isPHPTestFrameworkCall(call) {
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

// extractPHPCall extracts method/function call names.
// Handles: function(), $obj->method(), Class::staticMethod()
func extractPHPCall(node *sitter.Node, source []byte) string {
	switch node.Type() {
	case "function_call_expression":
		// function() or ClassName()
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			switch child.Type() {
			case phpast.NodeName, phpast.NodeQualifiedName:
				name := child.Content(source)
				// Get the last segment for qualified names
				if idx := strings.LastIndex(name, "\\"); idx >= 0 {
					return name[idx+1:]
				}
				return name
			}
		}

	case "member_call_expression":
		// $obj->method()
		var objectName, methodName string
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			switch child.Type() {
			case "variable_name":
				objectName = strings.TrimPrefix(child.Content(source), "$")
			case phpast.NodeName:
				methodName = child.Content(source)
			}
		}
		if objectName != "" && methodName != "" {
			return objectName + "." + methodName
		}

	case "scoped_call_expression":
		// Class::staticMethod()
		var className, methodName string
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			switch child.Type() {
			case phpast.NodeName, phpast.NodeQualifiedName:
				if className == "" {
					className = child.Content(source)
					if idx := strings.LastIndex(className, "\\"); idx >= 0 {
						className = className[idx+1:]
					}
				} else {
					methodName = child.Content(source)
				}
			}
		}
		if className != "" && methodName != "" {
			return className + "." + methodName
		}
	}

	return ""
}

// trimPHPQuotes removes string delimiters from PHP strings.
func trimPHPQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return s
	}
	first, last := s[0], s[len(s)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}

// phpTestFrameworkCalls contains base names from PHP test frameworks
// that should be excluded from domain hints.
var phpTestFrameworkCalls = map[string]struct{}{
	// PHPUnit assertions
	"this":        {},
	"self":        {},
	"Assert":      {},
	"assertSame":  {},
	"assertEquals": {},
	"assertTrue":  {},
	"assertFalse": {},
	// PHPUnit setup/teardown
	"setUp":          {},
	"tearDown":       {},
	"setUpBeforeClass": {},
	"tearDownAfterClass": {},
	// Mockery
	"Mockery":   {},
	"mock":      {},
	"spy":       {},
	"shouldReceive": {},
	// Prophecy
	"prophesize": {},
	"reveal":     {},
	// Pest
	"test":       {},
	"it":         {},
	"describe":   {},
	"beforeEach": {},
	"afterEach":  {},
	"expect":     {},
}

func isPHPTestFrameworkCall(call string) bool {
	baseName := call
	if idx := strings.Index(call, "."); idx > 0 {
		baseName = call[:idx]
	}
	_, exists := phpTestFrameworkCalls[baseName]
	return exists
}
