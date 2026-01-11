package domain_hints

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/strategies/shared/dotnetast"
	"github.com/specvital/core/pkg/parser/tspool"
)

// CSharpExtractor extracts domain hints from C# source code.
type CSharpExtractor struct{}

const (
	// using directives: using Namespace; using Alias = Namespace; using static Namespace.Type;
	csharpUsingQuery = `
		(using_directive) @using
	`

	// Method invocations: obj.Method(), ClassName.StaticMethod()
	csharpCallQuery = `
		(invocation_expression) @call
	`
)

func (e *CSharpExtractor) Extract(ctx context.Context, source []byte) *domain.DomainHints {
	// Sanitize source to handle NULL bytes
	source = dotnetast.SanitizeSource(source)

	tree, err := tspool.Parse(ctx, domain.LanguageCSharp, source)
	if err != nil {
		return nil
	}
	defer tree.Close()

	root := tree.RootNode()

	hints := &domain.DomainHints{
		Imports: e.extractUsings(root, source),
		Calls:   e.extractCalls(root, source),
	}

	if len(hints.Imports) == 0 && len(hints.Calls) == 0 {
		return nil
	}

	return hints
}

func (e *CSharpExtractor) extractUsings(root *sitter.Node, source []byte) []string {
	results, err := tspool.QueryWithCache(root, source, domain.LanguageCSharp, csharpUsingQuery)
	if err != nil {
		return nil
	}

	seen := make(map[string]struct{})
	var usings []string

	for _, r := range results {
		if node, ok := r.Captures["using"]; ok {
			usingPath := extractCSharpUsingPath(node, source)
			if usingPath != "" {
				if _, exists := seen[usingPath]; !exists {
					seen[usingPath] = struct{}{}
					usings = append(usings, usingPath)
				}
			}
		}
	}

	return usings
}

// extractCSharpUsingPath extracts the namespace from a using_directive node.
// Handles:
//   - using Namespace;
//   - using Namespace.SubNamespace;
//   - using Alias = Namespace;
//   - using static Namespace.Type;
//   - global using Namespace;
func extractCSharpUsingPath(node *sitter.Node, source []byte) string {
	// Check for alias pattern: using Alias = Namespace;
	// Structure: using identifier = qualified_name ;
	hasEquals := false
	for i := 0; i < int(node.ChildCount()); i++ {
		if node.Child(i).Type() == dotnetast.NodeEquals {
			hasEquals = true
			break
		}
	}

	if hasEquals {
		// Alias using: extract the qualified_name (the right side of =)
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == dotnetast.NodeQualifiedName {
				return child.Content(source)
			}
		}
		return ""
	}

	// Regular using: extract qualified_name or identifier
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		childType := child.Type()

		switch childType {
		case dotnetast.NodeQualifiedName:
			return child.Content(source)
		case dotnetast.NodeIdentifier:
			// Skip keywords like "using", "global", "static"
			content := child.Content(source)
			if content != "using" && content != "global" && content != "static" {
				return content
			}
		}
	}

	return ""
}

func (e *CSharpExtractor) extractCalls(root *sitter.Node, source []byte) []string {
	results, err := tspool.QueryWithCache(root, source, domain.LanguageCSharp, csharpCallQuery)
	if err != nil {
		return nil
	}

	seen := make(map[string]struct{})
	calls := make([]string, 0, len(results))

	for _, r := range results {
		if node, ok := r.Captures["call"]; ok {
			call := extractCSharpMethodCall(node, source)
			if call == "" {
				continue
			}
			// Normalize to 2 segments
			call = normalizeCall(call)
			if call == "" {
				continue
			}
			// Skip test framework calls
			if isCSharpTestFrameworkCall(call) {
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

// extractCSharpMethodCall extracts the method call expression.
// Handles: obj.Method(), ClassName.StaticMethod(), Method()
func extractCSharpMethodCall(node *sitter.Node, source []byte) string {
	// invocation_expression structure: function(arguments)
	// function can be: member_access_expression, identifier, etc.

	var parts []string

	// First child is usually the function being called
	if node.ChildCount() == 0 {
		return ""
	}

	funcNode := node.Child(0)
	if funcNode == nil {
		return ""
	}

	switch funcNode.Type() {
	case dotnetast.NodeMemberAccessExpression:
		// obj.Method or Obj.Prop.Method
		parts = extractMemberAccessParts(funcNode, source)
	case dotnetast.NodeIdentifier:
		// Simple function call: Method()
		parts = []string{funcNode.Content(source)}
	case dotnetast.NodeGenericName:
		// Generic method: Method<T>()
		for i := 0; i < int(funcNode.ChildCount()); i++ {
			child := funcNode.Child(i)
			if child.Type() == dotnetast.NodeIdentifier {
				parts = []string{child.Content(source)}
				break
			}
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, ".")
}

// extractMemberAccessParts recursively extracts parts from member_access_expression.
func extractMemberAccessParts(node *sitter.Node, source []byte) []string {
	var parts []string

	// member_access_expression has: expression . name
	exprNode := node.ChildByFieldName("expression")
	nameNode := node.ChildByFieldName("name")

	if exprNode != nil {
		switch exprNode.Type() {
		case dotnetast.NodeMemberAccessExpression:
			parts = append(parts, extractMemberAccessParts(exprNode, source)...)
		case dotnetast.NodeIdentifier:
			parts = append(parts, exprNode.Content(source))
		case dotnetast.NodeThisExpression:
			parts = append(parts, "this")
		case dotnetast.NodeInvocationExpression:
			// Method chaining: a.B().C() - extract the base
			if exprNode.ChildCount() > 0 {
				firstChild := exprNode.Child(0)
				if firstChild.Type() == dotnetast.NodeMemberAccessExpression {
					parts = append(parts, extractMemberAccessParts(firstChild, source)...)
				} else if firstChild.Type() == dotnetast.NodeIdentifier {
					parts = append(parts, firstChild.Content(source))
				}
			}
		default:
			// For other types, try to get the content
			content := exprNode.Content(source)
			if content != "" && !strings.Contains(content, "(") {
				parts = append(parts, content)
			}
		}
	}

	if nameNode != nil {
		switch nameNode.Type() {
		case dotnetast.NodeIdentifier:
			parts = append(parts, nameNode.Content(source))
		case dotnetast.NodeGenericName:
			for i := 0; i < int(nameNode.ChildCount()); i++ {
				child := nameNode.Child(i)
				if child.Type() == dotnetast.NodeIdentifier {
					parts = append(parts, child.Content(source))
					break
				}
			}
		}
	}

	return parts
}

// csharpTestFrameworkCalls contains base names from C# test frameworks
// that should be excluded from domain hints.
var csharpTestFrameworkCalls = map[string]struct{}{
	// NUnit assertions
	"Assert": {}, "Assume": {}, "Warn": {},
	// xUnit assertions
	"Xunit": {},
	// FluentAssertions
	"Should": {}, "BeEquivalentTo": {}, "Be": {}, "HaveCount": {},
	// MSTest assertions
	"CollectionAssert": {}, "StringAssert": {},
	// Common assertion patterns
	"Is": {}, "Has": {}, "Does": {}, "Contains": {}, "Throws": {},
	// Moq
	"Mock": {}, "Setup": {}, "Verify": {}, "Returns": {}, "Callback": {},
	"It": {}, "Times": {},
	// NSubstitute
	"Substitute": {}, "Received": {}, "DidNotReceive": {},
	// AutoFixture
	"Fixture": {}, "Create": {}, "Build": {}, "Freeze": {},
}

func isCSharpTestFrameworkCall(call string) bool {
	baseName := call
	if idx := strings.Index(call, "."); idx > 0 {
		baseName = call[:idx]
	}
	_, exists := csharpTestFrameworkCalls[baseName]
	return exists
}
