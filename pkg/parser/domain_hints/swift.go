package domain_hints

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/tspool"
)

// SwiftExtractor extracts domain hints from Swift source code.
type SwiftExtractor struct{}

const (
	// import Foundation, @testable import MyApp
	swiftImportQuery = `(import_declaration) @import`

	// Function/method calls
	swiftCallQuery = `
		(call_expression
			(navigation_expression) @call
		)
	`
)

func (e *SwiftExtractor) Extract(ctx context.Context, source []byte) *domain.DomainHints {
	tree, err := tspool.Parse(ctx, domain.LanguageSwift, source)
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

func (e *SwiftExtractor) extractImports(root *sitter.Node, source []byte) []string {
	results, err := tspool.QueryWithCache(root, source, domain.LanguageSwift, swiftImportQuery)
	if err != nil {
		return nil
	}

	seen := make(map[string]struct{})
	var imports []string

	for _, r := range results {
		if node, ok := r.Captures["import"]; ok {
			path := extractSwiftImportPath(node, source)
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

func (e *SwiftExtractor) extractCalls(root *sitter.Node, source []byte) []string {
	results, err := tspool.QueryWithCache(root, source, domain.LanguageSwift, swiftCallQuery)
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

			call = normalizeCall(call)
			if call == "" {
				continue
			}

			if isSwiftTestFrameworkCall(call) {
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

// extractSwiftImportPath extracts the module name from an import_declaration node.
// Handles: import Foundation
//          @testable import MyApp
//          import UIKit.UIView
func extractSwiftImportPath(node *sitter.Node, source []byte) string {
	text := getNodeText(node, source)
	if text == "" {
		return ""
	}

	// Remove @testable or other attributes
	if idx := strings.Index(text, "import "); idx >= 0 {
		text = text[idx+7:] // len("import ") = 7
	}

	text = strings.TrimSpace(text)

	// Handle dotted imports: import UIKit.UIView -> UIKit.UIView
	// Keep as-is for now, dots are meaningful in Swift

	return text
}

// swiftTestFrameworkCalls contains patterns from Swift test frameworks
// that should be excluded from domain hints.
var swiftTestFrameworkCalls = map[string]struct{}{
	// XCTest
	"XCTAssert":              {},
	"XCTAssertTrue":          {},
	"XCTAssertFalse":         {},
	"XCTAssertEqual":         {},
	"XCTAssertNotEqual":      {},
	"XCTAssertNil":           {},
	"XCTAssertNotNil":        {},
	"XCTAssertThrowsError":   {},
	"XCTAssertNoThrow":       {},
	"XCTFail":                {},
	"XCTSkip":                {},
	"XCTUnwrap":              {},
	"XCTExpectFailure":       {},
	// Swift Testing
	"expect":                 {},
	"require":                {},
	"Issue":                  {},
	"confirmation":           {},
	// Common utilities
	"print":                  {},
	"debugPrint":             {},
	"dump":                   {},
	"fatalError":             {},
	"precondition":           {},
	"preconditionFailure":    {},
	"assertionFailure":       {},
}

func isSwiftTestFrameworkCall(call string) bool {
	baseName := call
	if idx := strings.Index(call, "."); idx > 0 {
		baseName = call[:idx]
	}
	_, existsBase := swiftTestFrameworkCalls[baseName]
	_, existsFull := swiftTestFrameworkCalls[call]
	return existsBase || existsFull
}
