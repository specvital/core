// Package javaast provides shared Java AST traversal utilities for test framework parsers.
package javaast

import (
	"bytes"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// Java AST node types.
const (
	NodeClassDeclaration       = "class_declaration"
	NodeMethodDeclaration      = "method_declaration"
	NodeAnnotation             = "annotation"
	NodeMarkerAnnotation       = "marker_annotation"
	NodeModifiers              = "modifiers"
	NodeIdentifier             = "identifier"
	NodeFormalParameters       = "formal_parameters"
	NodeClassBody              = "class_body"
	NodeAnnotationArgumentList = "annotation_argument_list"
)

// GetAnnotations extracts all annotation nodes from a modifiers node.
func GetAnnotations(modifiers *sitter.Node) []*sitter.Node {
	if modifiers == nil {
		return nil
	}

	var annotations []*sitter.Node
	for i := 0; i < int(modifiers.ChildCount()); i++ {
		child := modifiers.Child(i)
		if child.Type() == NodeAnnotation || child.Type() == NodeMarkerAnnotation {
			annotations = append(annotations, child)
		}
	}
	return annotations
}

// GetAnnotationName extracts the simple annotation name (e.g., "Test" from @Test or @org.junit.jupiter.api.Test).
// For scoped identifiers, returns only the simple name (last segment).
func GetAnnotationName(annotation *sitter.Node, source []byte) string {
	if annotation == nil {
		return ""
	}

	for i := 0; i < int(annotation.ChildCount()); i++ {
		child := annotation.Child(i)
		if child.Type() == NodeIdentifier {
			return child.Content(source)
		}
		// Handle scoped identifier like @org.junit.jupiter.api.Test
		// Extract simple name: "org.junit.jupiter.api.Test" → "Test"
		if child.Type() == "scoped_identifier" {
			fullName := child.Content(source)
			if idx := strings.LastIndex(fullName, "."); idx >= 0 {
				return fullName[idx+1:]
			}
			return fullName
		}
	}
	return ""
}

// GetMethodName extracts the method name from a method_declaration node.
func GetMethodName(node *sitter.Node, source []byte) string {
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		return nameNode.Content(source)
	}
	return ""
}

// GetClassName extracts the class name from a class_declaration node.
func GetClassName(node *sitter.Node, source []byte) string {
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		return nameNode.Content(source)
	}
	return ""
}

// GetClassBody returns the class body node from a class_declaration.
func GetClassBody(node *sitter.Node) *sitter.Node {
	return node.ChildByFieldName("body")
}

// GetModifiers returns the modifiers node from a method or class declaration.
func GetModifiers(node *sitter.Node) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeModifiers {
			return child
		}
	}
	return nil
}

// HasAnnotation checks if the modifiers contain a specific annotation.
func HasAnnotation(modifiers *sitter.Node, source []byte, annotationName string) bool {
	annotations := GetAnnotations(modifiers)
	for _, ann := range annotations {
		name := GetAnnotationName(ann, source)
		if name == annotationName {
			return true
		}
	}
	return false
}

// GetAnnotationArgument extracts the annotation argument value.
// For @DisplayName("My Test"), returns "My Test".
func GetAnnotationArgument(annotation *sitter.Node, source []byte) string {
	for i := 0; i < int(annotation.ChildCount()); i++ {
		child := annotation.Child(i)
		if child.Type() == NodeAnnotationArgumentList {
			// Get the first argument value
			for j := 0; j < int(child.ChildCount()); j++ {
				arg := child.Child(j)
				if arg.Type() == "string_literal" {
					text := arg.Content(source)
					// Remove surrounding quotes
					if len(text) >= 2 {
						return text[1 : len(text)-1]
					}
				}
			}
		}
	}
	return ""
}

// SanitizeSource removes NULL bytes from source code that would cause tree-sitter parsing failures.
// Some files (e.g., OSS-Fuzz test data) contain NULL bytes in string literals which cause
// tree-sitter to produce ERROR nodes instead of valid AST.
func SanitizeSource(source []byte) []byte {
	if !bytes.Contains(source, []byte{0}) {
		return source
	}
	return bytes.ReplaceAll(source, []byte{0}, []byte{' '})
}
