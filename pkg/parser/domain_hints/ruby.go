package domain_hints

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/tspool"
)

// RubyExtractor extracts domain hints from Ruby source code.
type RubyExtractor struct{}

const (
	// require 'path', require "path", require_relative 'path'
	rubyRequireQuery = `
		(call
			method: (identifier) @method
			arguments: (argument_list (string) @path)
		)
	`

	// Method calls: obj.method(), Constant.method()
	rubyMethodCallQuery = `
		(call
			receiver: [
				(identifier) @receiver
				(constant) @receiver
			]
			method: (identifier) @method
		)
	`
)

func (e *RubyExtractor) Extract(ctx context.Context, source []byte) *domain.DomainHints {
	tree, err := tspool.Parse(ctx, domain.LanguageRuby, source)
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

func (e *RubyExtractor) extractImports(root *sitter.Node, source []byte) []string {
	results, err := tspool.QueryWithCache(root, source, domain.LanguageRuby, rubyRequireQuery)
	if err != nil {
		return nil
	}

	seen := make(map[string]struct{})
	var imports []string

	for _, r := range results {
		methodNode, hasMethod := r.Captures["method"]
		pathNode, hasPath := r.Captures["path"]
		if !hasMethod || !hasPath {
			continue
		}

		methodName := getNodeText(methodNode, source)
		if methodName != "require" && methodName != "require_relative" {
			continue
		}

		path := trimRubyQuotes(getNodeText(pathNode, source))
		if path == "" {
			continue
		}

		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		imports = append(imports, path)
	}

	return imports
}

func (e *RubyExtractor) extractCalls(root *sitter.Node, source []byte) []string {
	results, err := tspool.QueryWithCache(root, source, domain.LanguageRuby, rubyMethodCallQuery)
	if err != nil {
		return nil
	}

	seen := make(map[string]struct{})
	calls := make([]string, 0, len(results))

	for _, r := range results {
		receiverNode, hasReceiver := r.Captures["receiver"]
		methodNode, hasMethod := r.Captures["method"]
		if !hasReceiver || !hasMethod {
			continue
		}

		receiver := getNodeText(receiverNode, source)
		method := getNodeText(methodNode, source)
		if receiver == "" || method == "" {
			continue
		}

		call := receiver + "." + method
		call = normalizeCall(call)
		if call == "" {
			continue
		}

		if isRubyTestFrameworkCall(call) {
			continue
		}

		if _, exists := seen[call]; exists {
			continue
		}
		seen[call] = struct{}{}
		calls = append(calls, call)
	}

	return calls
}

// trimRubyQuotes removes string delimiters from Ruby strings.
// Handles: 'single', "double", %q(string), %Q(string), %q[string], %q{string}
func trimRubyQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return s
	}

	// Handle %q and %Q syntax with various delimiters
	if len(s) >= 4 && (strings.HasPrefix(s, "%q") || strings.HasPrefix(s, "%Q")) {
		opener := s[2]
		closer := getMatchingDelimiter(opener)
		if s[len(s)-1] == closer {
			return s[3 : len(s)-1]
		}
	}

	// Handle standard quotes
	first, last := s[0], s[len(s)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
		return s[1 : len(s)-1]
	}

	return s
}

// getMatchingDelimiter returns the closing delimiter for Ruby %q syntax.
func getMatchingDelimiter(opener byte) byte {
	switch opener {
	case '(':
		return ')'
	case '[':
		return ']'
	case '{':
		return '}'
	case '<':
		return '>'
	default:
		return opener // For symmetric delimiters like |, !, etc.
	}
}

// rubyTestFrameworkCalls contains base names from Ruby test frameworks
// that should be excluded from domain hints.
var rubyTestFrameworkCalls = map[string]struct{}{
	// RSpec
	"RSpec":           {},
	"describe":        {},
	"context":         {},
	"it":              {},
	"specify":         {},
	"example":         {},
	"expect":          {},
	"allow":           {},
	"before":          {},
	"after":           {},
	"let":             {},
	"let!":            {},
	"subject":         {},
	"shared_examples": {},
	"include_examples": {},
	"shared_context":  {},
	"include_context": {},
	// Minitest
	"assert":       {},
	"refute":       {},
	"assert_equal": {},
	"refute_equal": {},
	"must_equal":   {},
	"wont_equal":   {},
	// FactoryBot
	"FactoryBot": {},
	"factory":    {},
	"build":      {},
	"create":     {},
	// Ruby core test helpers
	"puts":  {},
	"print": {},
	"raise": {},
	"p":     {},
	"pp":    {},
}

func isRubyTestFrameworkCall(call string) bool {
	baseName := call
	if idx := strings.Index(call, "."); idx > 0 {
		baseName = call[:idx]
	}
	_, exists := rubyTestFrameworkCalls[baseName]
	return exists
}
