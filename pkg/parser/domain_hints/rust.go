package domain_hints

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/tspool"
)

// RustExtractor extracts domain hints from Rust source code.
type RustExtractor struct{}

const (
	// use std::collections::HashMap;
	// use crate::models::User;
	// mod tests;
	rustUseQuery = `(use_declaration) @use`
	rustModQuery = `(mod_item name: (identifier) @mod)`

	// Method calls: obj.method(), Type::method()
	rustCallQuery = `
		(call_expression
			function: [
				(identifier) @call
				(scoped_identifier) @call
				(field_expression) @call
			]
		)
	`
)

func (e *RustExtractor) Extract(ctx context.Context, source []byte) *domain.DomainHints {
	tree, err := tspool.Parse(ctx, domain.LanguageRust, source)
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

func (e *RustExtractor) extractImports(root *sitter.Node, source []byte) []string {
	seen := make(map[string]struct{})
	var imports []string

	// Extract use declarations
	useResults, err := tspool.QueryWithCache(root, source, domain.LanguageRust, rustUseQuery)
	if err == nil {
		for _, r := range useResults {
			if node, ok := r.Captures["use"]; ok {
				path := extractRustUsePath(node, source)
				if path != "" {
					if _, exists := seen[path]; !exists {
						seen[path] = struct{}{}
						imports = append(imports, path)
					}
				}
			}
		}
	}

	// Extract mod declarations
	modResults, err := tspool.QueryWithCache(root, source, domain.LanguageRust, rustModQuery)
	if err == nil {
		for _, r := range modResults {
			if node, ok := r.Captures["mod"]; ok {
				modName := getNodeText(node, source)
				if modName != "" {
					if _, exists := seen[modName]; !exists {
						seen[modName] = struct{}{}
						imports = append(imports, modName)
					}
				}
			}
		}
	}

	return imports
}

func (e *RustExtractor) extractCalls(root *sitter.Node, source []byte) []string {
	results, err := tspool.QueryWithCache(root, source, domain.LanguageRust, rustCallQuery)
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

			// Convert :: to . for normalization
			call = strings.ReplaceAll(call, "::", ".")
			call = normalizeCall(call)
			if call == "" {
				continue
			}

			if isRustTestFrameworkCall(call) {
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

// extractRustUsePath extracts the path from a use_declaration node.
// Handles: use std::collections::HashMap;
//          use crate::models::{User, Order};
//          use super::helpers;
func extractRustUsePath(node *sitter.Node, source []byte) string {
	// Get the full use declaration text and extract the path
	text := getNodeText(node, source)
	if text == "" {
		return ""
	}

	// Remove "use " prefix and trailing ";"
	text = strings.TrimPrefix(text, "use ")
	text = strings.TrimSuffix(text, ";")
	text = strings.TrimSpace(text)

	// Handle use lists: use crate::{a, b} -> crate
	if idx := strings.Index(text, "::{"); idx > 0 {
		text = text[:idx]
	}

	// Handle use as: use std::collections::HashMap as Map -> std::collections::HashMap
	if idx := strings.Index(text, " as "); idx > 0 {
		text = text[:idx]
	}

	// Handle wildcard: use std::* -> std
	text = strings.TrimSuffix(text, "::*")

	// Convert :: to / for consistency with other languages
	text = strings.ReplaceAll(text, "::", "/")

	return text
}

// rustTestFrameworkCalls contains patterns from Rust test frameworks
// that should be excluded from domain hints.
var rustTestFrameworkCalls = map[string]struct{}{
	// Standard test macros
	"assert":         {},
	"assert_eq":      {},
	"assert_ne":      {},
	"debug_assert":   {},
	"panic":          {},
	"unreachable":    {},
	"todo":           {},
	"unimplemented":  {},
	// Common test utilities
	"println":        {},
	"print":          {},
	"eprintln":       {},
	"eprint":         {},
	"dbg":            {},
	"format":         {},
	"vec":            {},
	// tokio-test
	"tokio.test":     {},
	// proptest
	"proptest":       {},
	"prop_assert":    {},
	"prop_assert_eq": {},
}

func isRustTestFrameworkCall(call string) bool {
	baseName := call
	if idx := strings.Index(call, "."); idx > 0 {
		baseName = call[:idx]
	}
	// Also check the full call for macros
	_, existsBase := rustTestFrameworkCalls[baseName]
	_, existsFull := rustTestFrameworkCalls[call]
	return existsBase || existsFull
}
