// Package dotnetast provides shared C# AST traversal utilities for .NET test framework parsers.
package dotnetast

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// C# AST node types.
const (
	NodeClassDeclaration      = "class_declaration"
	NodeMethodDeclaration     = "method_declaration"
	NodeAttributeList         = "attribute_list"
	NodeAttribute             = "attribute"
	NodeAttributeArgumentList = "attribute_argument_list"
	NodeAttributeArgument     = "attribute_argument"
	NodeIdentifier            = "identifier"
	NodeDeclarationList       = "declaration_list"
	NodeStringLiteral         = "string_literal"
	NodeVerbatimStringLiteral = "verbatim_string_literal"
	NodeStringLiteralContent  = "string_literal_content"
	NodeInterpolatedString    = "interpolated_string_expression"
	NodeNamespaceDeclaration  = "namespace_declaration"
	NodeFileScopedNamespace   = "file_scoped_namespace_declaration"
	NodeUsingDirective        = "using_directive"
	NodeQualifiedName         = "qualified_name"
	NodeModifier              = "modifier"
	NodeGenericName           = "generic_name"
	NodeAssignmentExpression  = "assignment_expression"
	NodePreprocIf             = "preproc_if"
	NodePreprocElse           = "preproc_else"
	NodePreprocElif           = "preproc_elif"
)

// GetClassName extracts the class name from a class_declaration node.
func GetClassName(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		return nameNode.Content(source)
	}
	return ""
}

// GetMethodName extracts the method name from a method_declaration node.
func GetMethodName(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		return nameNode.Content(source)
	}
	return ""
}

// GetDeclarationList returns the body (declaration_list) from a class_declaration.
func GetDeclarationList(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	return node.ChildByFieldName("body")
}

// GetAttributeLists returns all attribute_list nodes preceding a declaration.
//
// KNOWN LIMITATION: tree-sitter-c-sharp parses preprocessor directives (#if, #else, #elif)
// between attributes as ERROR nodes instead of preproc_if. This means attributes inside
// conditional compilation blocks like:
//
//	[Theory]
//	[InlineData(1)]
//	#if NET6_0_OR_GREATER
//	[InlineData(2)]  // ← Not detected (parsed as ERROR)
//	#endif
//	public void Test(int x) { }
//
// will not be fully detected. This is a tree-sitter grammar limitation, not a parser bug.
// See: https://github.com/tree-sitter/tree-sitter-c-sharp/issues
func GetAttributeLists(node *sitter.Node) []*sitter.Node {
	if node == nil {
		return nil
	}
	var attributes []*sitter.Node
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeAttributeList {
			attributes = append(attributes, child)
		}
	}
	return attributes
}

// GetAttributes extracts all attribute nodes from attribute_list nodes.
func GetAttributes(attributeLists []*sitter.Node) []*sitter.Node {
	var attributes []*sitter.Node
	for _, attrList := range attributeLists {
		for i := 0; i < int(attrList.ChildCount()); i++ {
			child := attrList.Child(i)
			if child.Type() == NodeAttribute {
				attributes = append(attributes, child)
			}
		}
	}
	return attributes
}

// GetAttributeName extracts the attribute name (e.g., "Fact" from [Fact] or "Theory" from [Theory]).
// Handles both simple identifiers and qualified names.
func GetAttributeName(attribute *sitter.Node, source []byte) string {
	if attribute == nil {
		return ""
	}

	nameNode := attribute.ChildByFieldName("name")
	if nameNode == nil {
		return ""
	}

	switch nameNode.Type() {
	case NodeIdentifier:
		return nameNode.Content(source)
	case NodeQualifiedName:
		fullName := nameNode.Content(source)
		if idx := strings.LastIndex(fullName, "."); idx >= 0 {
			return fullName[idx+1:]
		}
		return fullName
	case NodeGenericName:
		for i := 0; i < int(nameNode.ChildCount()); i++ {
			child := nameNode.Child(i)
			if child.Type() == NodeIdentifier {
				return child.Content(source)
			}
		}
	}

	return ""
}

// HasAttribute checks if the attribute lists contain a specific attribute.
func HasAttribute(attributeLists []*sitter.Node, source []byte, attributeName string) bool {
	for _, attr := range GetAttributes(attributeLists) {
		name := GetAttributeName(attr, source)
		if name == attributeName || name == attributeName+"Attribute" {
			return true
		}
	}
	return false
}

// ExtractStringContent extracts string content from various string literal node types.
// Supports regular string literals ("..."), verbatim strings (@"..."), and interpolated strings ($"...").
// This is exported for reuse by xUnit, NUnit, MSTest, and other .NET framework parsers.
func ExtractStringContent(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}

	switch node.Type() {
	case NodeStringLiteral:
		text := node.Content(source)
		if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
			return text[1 : len(text)-1]
		}
		return text
	case NodeVerbatimStringLiteral:
		text := node.Content(source)
		if len(text) >= 3 && text[0] == '@' && text[1] == '"' && text[len(text)-1] == '"' {
			return text[2 : len(text)-1]
		}
		return text
	case NodeInterpolatedString:
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == NodeStringLiteralContent {
				return child.Content(source)
			}
		}
	}
	return ""
}

// FindAttributeArgumentList finds the attribute_argument_list child of an attribute node.
// Used by xUnit, NUnit, and other .NET framework parsers.
func FindAttributeArgumentList(attr *sitter.Node) *sitter.Node {
	for i := 0; i < int(attr.ChildCount()); i++ {
		child := attr.Child(i)
		if child.Type() == NodeAttributeArgumentList {
			return child
		}
	}
	return nil
}

// ParseAssignmentExpression extracts name and value from an assignment_expression node.
// For "Skip = \"reason\"", returns ("Skip", "reason").
// Used by xUnit (Skip, DisplayName), NUnit (Description), and other .NET framework parsers.
func ParseAssignmentExpression(argNode *sitter.Node, source []byte) (string, string) {
	for i := 0; i < int(argNode.ChildCount()); i++ {
		child := argNode.Child(i)
		if child.Type() == NodeAssignmentExpression {
			var name, value string
			for j := 0; j < int(child.ChildCount()); j++ {
				part := child.Child(j)
				switch part.Type() {
				case NodeIdentifier:
					name = part.Content(source)
				case NodeStringLiteral:
					value = ExtractStringContent(part, source)
				}
			}
			return name, value
		}
	}
	return "", ""
}

// IsCSharpTestFileName checks if a filename follows C# test file naming conventions.
// Matches: *Test.cs, *Tests.cs, Test*.cs, *Spec.cs, *Specs.cs
func IsCSharpTestFileName(filename string) bool {
	// Extract base name
	base := filename
	if idx := strings.LastIndex(filename, "/"); idx >= 0 {
		base = filename[idx+1:]
	}
	if idx := strings.LastIndex(base, "\\"); idx >= 0 {
		base = base[idx+1:]
	}

	if !strings.HasSuffix(base, ".cs") {
		return false
	}

	name := strings.TrimSuffix(base, ".cs")

	// xUnit/NUnit/MSTest conventions
	if strings.HasSuffix(name, "Test") || strings.HasSuffix(name, "Tests") {
		return true
	}
	if strings.HasPrefix(name, "Test") {
		return true
	}
	// BDD-style naming
	if strings.HasSuffix(name, "Spec") || strings.HasSuffix(name, "Specs") {
		return true
	}

	return false
}

// IsPreprocessorDirective checks if the node is a C# preprocessor directive.
func IsPreprocessorDirective(nodeType string) bool {
	return nodeType == NodePreprocIf || nodeType == NodePreprocElse || nodeType == NodePreprocElif
}

// GetDeclarationChildren returns all declarations from a declaration_list,
// including those inside preprocessor directives (#if, #else, #elif).
// This handles C# conditional compilation where tests may be wrapped in
// #if NET6_0_OR_GREATER or similar directives.
func GetDeclarationChildren(body *sitter.Node) []*sitter.Node {
	if body == nil {
		return nil
	}

	var children []*sitter.Node
	collectDeclarations(body, &children)
	return children
}

func collectDeclarations(node *sitter.Node, result *[]*sitter.Node) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		nodeType := child.Type()

		switch nodeType {
		case NodeClassDeclaration, NodeMethodDeclaration:
			*result = append(*result, child)
		case NodePreprocIf, NodePreprocElse, NodePreprocElif:
			collectDeclarations(child, result)
		}
	}
}
