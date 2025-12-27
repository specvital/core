// Package phpast provides shared PHP AST traversal utilities for test framework parsers.
package phpast

import (
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// PHP AST node types.
const (
	NodeClassDeclaration   = "class_declaration"
	NodeMethodDeclaration  = "method_declaration"
	NodeDeclarationList    = "declaration_list"
	NodeAttributeList      = "attribute_list"
	NodeAttributeGroup     = "attribute_group"
	NodeAttribute          = "attribute"
	NodeComment            = "comment"
	NodeName               = "name"
	NodeVisibilityModifier = "visibility_modifier"
	NodeQualifiedName      = "qualified_name"
	NodeNamespaceUse       = "namespace_use_declaration"
	NodeBaseClause         = "base_clause"
)

// GetClassName extracts the class name from a class_declaration node.
func GetClassName(node *sitter.Node, source []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeName {
			return child.Content(source)
		}
	}
	return ""
}

// GetMethodName extracts the method name from a method_declaration node.
func GetMethodName(node *sitter.Node, source []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeName {
			return child.Content(source)
		}
	}
	return ""
}

// GetDeclarationList returns the declaration_list (class body) from a class_declaration.
func GetDeclarationList(node *sitter.Node) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeDeclarationList {
			return child
		}
	}
	return nil
}

// GetAttributes extracts PHP 8 attribute nodes from a method declaration.
func GetAttributes(node *sitter.Node) []*sitter.Node {
	var attrs []*sitter.Node
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeAttributeList {
			for j := 0; j < int(child.ChildCount()); j++ {
				group := child.Child(j)
				if group.Type() == NodeAttributeGroup {
					for k := 0; k < int(group.ChildCount()); k++ {
						attr := group.Child(k)
						if attr.Type() == NodeAttribute {
							attrs = append(attrs, attr)
						}
					}
				}
			}
		}
	}
	return attrs
}

// GetAttributeName extracts the attribute name (e.g., "Test" from #[Test]).
func GetAttributeName(attr *sitter.Node, source []byte) string {
	for i := 0; i < int(attr.ChildCount()); i++ {
		child := attr.Child(i)
		if child.Type() == NodeName || child.Type() == NodeQualifiedName {
			text := child.Content(source)
			// Handle qualified names like PHPUnit\Framework\Attributes\Test
			if idx := strings.LastIndex(text, "\\"); idx >= 0 {
				return text[idx+1:]
			}
			return text
		}
	}
	return ""
}

// testAnnotationPattern matches @test annotation in docblocks.
var testAnnotationPattern = regexp.MustCompile(`@test\b`)

// HasTestAnnotation checks if a docblock contains @test annotation.
func HasTestAnnotation(comment string) bool {
	return testAnnotationPattern.MatchString(comment)
}

// HasTestAttribute checks if a method has #[Test] attribute (PHP 8+).
func HasTestAttribute(attrs []*sitter.Node, source []byte) bool {
	for _, attr := range attrs {
		name := GetAttributeName(attr, source)
		if name == "Test" {
			return true
		}
	}
	return false
}

// HasSkipAttribute checks if a method has skip-related attributes.
func HasSkipAttribute(attrs []*sitter.Node, source []byte) (bool, string) {
	for _, attr := range attrs {
		name := GetAttributeName(attr, source)
		switch name {
		case "Skip", "Ignore":
			return true, "#[" + name + "]"
		}
	}
	return false, ""
}

// IsPHPTestFileName checks if the filename follows PHPUnit test naming conventions.
func IsPHPTestFileName(filename string) bool {
	base := filename
	if idx := strings.LastIndex(filename, "/"); idx >= 0 {
		base = filename[idx+1:]
	}

	if !strings.HasSuffix(base, ".php") {
		return false
	}

	name := strings.TrimSuffix(base, ".php")

	// *Test.php, *Tests.php, Test*.php
	return strings.HasSuffix(name, "Test") ||
		strings.HasSuffix(name, "Tests") ||
		strings.HasPrefix(name, "Test")
}

// ExtendsTestCase checks if a class extends a TestCase base class.
// Matches: TestCase, BaseTestCase, *TestCase, *Test (suffix match for indirect inheritance)
// Does NOT match: TestCaseHelper, TestHelper (not valid test base class suffix)
func ExtendsTestCase(node *sitter.Node, source []byte) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeBaseClause {
			baseName := extractBaseClassName(child, source)
			return strings.HasSuffix(baseName, "TestCase") || strings.HasSuffix(baseName, "Test")
		}
	}
	return false
}

// extractBaseClassName extracts the base class name from a base_clause node.
// For qualified names like \PHPUnit\Framework\TestCase, returns "TestCase".
func extractBaseClassName(baseClause *sitter.Node, source []byte) string {
	for i := 0; i < int(baseClause.ChildCount()); i++ {
		child := baseClause.Child(i)
		switch child.Type() {
		case NodeName:
			return child.Content(source)
		case NodeQualifiedName:
			text := child.Content(source)
			if idx := strings.LastIndex(text, "\\"); idx >= 0 {
				return text[idx+1:]
			}
			return text
		}
	}
	return ""
}
