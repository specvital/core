package domain_hints

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/tspool"
)

// CppExtractor extracts domain hints from C++ source code.
type CppExtractor struct{}

const (
	// #include <iostream>, #include "myheader.h"
	cppIncludeQuery = `(preproc_include path: (_) @include)`

	// Function/method calls
	cppCallQuery = `
		(call_expression
			function: [
				(identifier) @call
				(qualified_identifier) @call
				(field_expression) @call
			]
		)
	`
)

func (e *CppExtractor) Extract(ctx context.Context, source []byte) *domain.DomainHints {
	tree, err := tspool.Parse(ctx, domain.LanguageCpp, source)
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

func (e *CppExtractor) extractImports(root *sitter.Node, source []byte) []string {
	results, err := tspool.QueryWithCache(root, source, domain.LanguageCpp, cppIncludeQuery)
	if err != nil {
		return nil
	}

	seen := make(map[string]struct{})
	var imports []string

	for _, r := range results {
		if node, ok := r.Captures["include"]; ok {
			path := extractCppIncludePath(node, source)
			if path == "" {
				continue
			}

			if _, exists := seen[path]; exists {
				continue
			}
			seen[path] = struct{}{}
			imports = append(imports, path)
		}
	}

	return imports
}

func (e *CppExtractor) extractCalls(root *sitter.Node, source []byte) []string {
	results, err := tspool.QueryWithCache(root, source, domain.LanguageCpp, cppCallQuery)
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
			// Handle -> operator for pointer access
			call = strings.ReplaceAll(call, "->", ".")
			call = normalizeCall(call)
			if call == "" {
				continue
			}

			if isCppTestFrameworkCall(call) {
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

// extractCppIncludePath extracts the path from an include directive.
// Handles: #include <iostream>
//          #include "myheader.h"
//          #include <gtest/gtest.h>
func extractCppIncludePath(node *sitter.Node, source []byte) string {
	text := getNodeText(node, source)
	if text == "" {
		return ""
	}

	text = strings.TrimSpace(text)

	// Remove angle brackets or quotes
	if len(text) >= 2 {
		if (text[0] == '<' && text[len(text)-1] == '>') ||
			(text[0] == '"' && text[len(text)-1] == '"') {
			text = text[1 : len(text)-1]
		}
	}

	return text
}

// cppTestFrameworkCalls contains patterns from C++ test frameworks
// that should be excluded from domain hints.
var cppTestFrameworkCalls = map[string]struct{}{
	// Google Test
	"EXPECT_TRUE":         {},
	"EXPECT_FALSE":        {},
	"EXPECT_EQ":           {},
	"EXPECT_NE":           {},
	"EXPECT_LT":           {},
	"EXPECT_LE":           {},
	"EXPECT_GT":           {},
	"EXPECT_GE":           {},
	"EXPECT_STREQ":        {},
	"EXPECT_STRNE":        {},
	"EXPECT_THROW":        {},
	"EXPECT_NO_THROW":     {},
	"EXPECT_DEATH":        {},
	"ASSERT_TRUE":         {},
	"ASSERT_FALSE":        {},
	"ASSERT_EQ":           {},
	"ASSERT_NE":           {},
	"ASSERT_LT":           {},
	"ASSERT_LE":           {},
	"ASSERT_GT":           {},
	"ASSERT_GE":           {},
	"ASSERT_STREQ":        {},
	"ASSERT_STRNE":        {},
	"ASSERT_THROW":        {},
	"ASSERT_NO_THROW":     {},
	"ASSERT_DEATH":        {},
	"TEST":                {},
	"TEST_F":              {},
	"TEST_P":              {},
	"TYPED_TEST":          {},
	"TYPED_TEST_SUITE":    {},
	"INSTANTIATE_TEST_SUITE_P": {},
	// Catch2
	"REQUIRE":             {},
	"REQUIRE_FALSE":       {},
	"REQUIRE_THROWS":      {},
	"REQUIRE_NOTHROW":     {},
	"CHECK":               {},
	"CHECK_FALSE":         {},
	"CHECK_THROWS":        {},
	"CHECK_NOTHROW":       {},
	"SECTION":             {},
	"TEST_CASE":           {},
	"SCENARIO":            {},
	"GIVEN":               {},
	"WHEN":                {},
	"THEN":                {},
	// Common utilities
	"std.cout":            {},
	"std.cerr":            {},
	"std.endl":            {},
	"printf":              {},
	"fprintf":             {},
	"cout":                {},
	"cerr":                {},
}

func isCppTestFrameworkCall(call string) bool {
	baseName := call
	if idx := strings.Index(call, "."); idx > 0 {
		baseName = call[:idx]
	}
	_, existsBase := cppTestFrameworkCalls[baseName]
	_, existsFull := cppTestFrameworkCalls[call]
	return existsBase || existsFull
}
