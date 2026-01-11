package domain_hints

import (
	"context"
	"regexp"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/tspool"
)

// GoExtractor extracts domain hints from Go source code.
type GoExtractor struct{}

const (
	goImportQuery = `(import_spec path: (interpreted_string_literal) @import)`

	goCallQuery = `
		(function_declaration
			body: (block
				(expression_statement
					(call_expression
						function: [
							(identifier) @call
							(selector_expression) @call
						]
					)
				)
			)
		)
		(function_declaration
			body: (block
				(short_var_declaration
					right: (expression_list
						(call_expression
							function: [
								(identifier) @call
								(selector_expression) @call
							]
						)
					)
				)
			)
		)
	`

	goVariableQuery = `
		(function_declaration
			body: (block
				(short_var_declaration
					left: (expression_list
						(identifier) @var
					)
				)
			)
		)
		(function_declaration
			body: (block
				(var_declaration
					(var_spec
						name: (identifier) @var
					)
				)
			)
		)
	`
)

var domainVariablePattern = regexp.MustCompile(`(?i)(mock|fake|stub|fixture|test|expected|want|got)`)

func (e *GoExtractor) Extract(ctx context.Context, source []byte) *domain.DomainHints {
	tree, err := tspool.Parse(ctx, domain.LanguageGo, source)
	if err != nil {
		return nil
	}
	defer tree.Close()

	root := tree.RootNode()

	hints := &domain.DomainHints{
		Imports:   extractGoImports(root, source),
		Calls:     extractGoCalls(root, source),
		Variables: extractGoVariables(root, source),
	}

	if len(hints.Imports) == 0 && len(hints.Calls) == 0 && len(hints.Variables) == 0 {
		return nil
	}

	return hints
}

func extractGoImports(root *sitter.Node, source []byte) []string {
	results, err := tspool.QueryWithCache(root, source, domain.LanguageGo, goImportQuery)
	if err != nil {
		return nil
	}

	imports := make([]string, 0, len(results))
	for _, r := range results {
		if node, ok := r.Captures["import"]; ok {
			path := trimQuotes(getNodeText(node, source))
			if path != "" {
				imports = append(imports, path)
			}
		}
	}

	return imports
}

func extractGoCalls(root *sitter.Node, source []byte) []string {
	results, err := tspool.QueryWithCache(root, source, domain.LanguageGo, goCallQuery)
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
			if _, exists := seen[call]; exists {
				continue
			}
			seen[call] = struct{}{}
			calls = append(calls, call)
		}
	}

	return calls
}

func extractGoVariables(root *sitter.Node, source []byte) []string {
	results, err := tspool.QueryWithCache(root, source, domain.LanguageGo, goVariableQuery)
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

func trimQuotes(s string) string {
	if len(s) < 2 {
		return s
	}
	if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '`' && s[len(s)-1] == '`') {
		return s[1 : len(s)-1]
	}
	return s
}

func getNodeText(node *sitter.Node, source []byte) (result string) {
	start := node.StartByte()
	end := node.EndByte()
	sourceLen := uint32(len(source))

	if start > sourceLen || end > sourceLen {
		return ""
	}

	defer func() {
		if r := recover(); r != nil {
			result = ""
		}
	}()

	return node.Content(source)
}
