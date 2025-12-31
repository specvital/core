// Package kotlinast provides shared Kotlin AST traversal utilities for test framework parsers.
package kotlinast

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// Kotlin AST node types.
const (
	NodeClassDeclaration       = "class_declaration"
	NodeObjectDeclaration      = "object_declaration"
	NodeFunctionDeclaration    = "function_declaration"
	NodeCallExpression         = "call_expression"
	NodeNavigationExpression   = "navigation_expression"
	NodeClassBody              = "class_body"
	NodeFunctionBody           = "function_body"
	NodeLambdaLiteral          = "lambda_literal"
	NodeAnnotation             = "annotation"
	NodeModifiers              = "modifiers"
	NodeIdentifier             = "simple_identifier"
	NodeStringLiteral          = "string_literal"
	NodeLineStringLiteral      = "line_string_literal"
	NodeMultiLineStringLiteral = "multi_line_string_literal"
	NodeLineStringContent      = "line_string_content"
	NodeCallSuffix             = "call_suffix"
	NodeValueArguments         = "value_arguments"
	NodeValueArgument          = "value_argument"
	NodeDelegationSpecifiers   = "delegation_specifiers"
	NodeDelegationSpecifier    = "delegation_specifier"
	NodeConstructorInvocation  = "constructor_invocation"
	NodeUserType               = "user_type"
	NodeTypeIdentifier         = "type_identifier"
	NodePrimaryConstructor     = "primary_constructor"
	NodeStatements             = "statements"
	NodeAnnotatedLambda        = "annotated_lambda"
	NodeInfixExpression        = "infix_expression"
	NodePostfixExpression      = "postfix_expression"
	NodeAdditiveExpression     = "additive_expression"
)

// Kotest spec styles (alphabetically ordered).
const (
	SpecAnnotationSpec = "AnnotationSpec"
	SpecBehaviorSpec   = "BehaviorSpec"
	SpecDescribeSpec   = "DescribeSpec"
	SpecExpectSpec     = "ExpectSpec"
	SpecFeatureSpec    = "FeatureSpec"
	SpecFreeSpec       = "FreeSpec"
	SpecFunSpec        = "FunSpec"
	SpecShouldSpec     = "ShouldSpec"
	SpecStringSpec     = "StringSpec"
	SpecWordSpec       = "WordSpec"
)

// GetClassName extracts the class name from a class_declaration node.
func GetClassName(node *sitter.Node, source []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeTypeIdentifier || child.Type() == NodeIdentifier {
			return child.Content(source)
		}
	}
	return ""
}

// GetClassBody returns the class body node from a class_declaration.
func GetClassBody(node *sitter.Node) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeClassBody {
			return child
		}
	}
	return nil
}

// GetConstructorLambda returns the lambda from a Kotest-style constructor invocation.
// Kotest specs typically use: class MyTest : FunSpec({ ... })
// The lambda is passed as an argument to the constructor invocation.
func GetConstructorLambda(node *sitter.Node) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeDelegationSpecifier {
			return getLambdaFromDelegationSpecifier(child)
		}
		if child.Type() == NodeDelegationSpecifiers {
			for j := 0; j < int(child.ChildCount()); j++ {
				spec := child.Child(j)
				if lambda := getLambdaFromDelegationSpecifier(spec); lambda != nil {
					return lambda
				}
			}
		}
	}
	return nil
}

func getLambdaFromDelegationSpecifier(node *sitter.Node) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeConstructorInvocation {
			return getLambdaFromConstructorInvocation(child)
		}
	}
	return nil
}

func getLambdaFromConstructorInvocation(node *sitter.Node) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeValueArguments {
			return getLambdaFromValueArguments(child)
		}
	}
	return nil
}

func getLambdaFromValueArguments(node *sitter.Node) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeValueArgument {
			for j := 0; j < int(child.ChildCount()); j++ {
				arg := child.Child(j)
				if arg.Type() == NodeLambdaLiteral {
					return arg
				}
			}
		}
		if child.Type() == NodeLambdaLiteral {
			return child
		}
	}
	return nil
}

// GetSuperTypes returns the super types from a class_declaration.
func GetSuperTypes(node *sitter.Node, source []byte) []string {
	var types []string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeDelegationSpecifier {
			typeName := extractTypeFromDelegationSpecifier(child, source)
			if typeName != "" {
				types = append(types, typeName)
			}
		}
		if child.Type() == NodeDelegationSpecifiers {
			for j := 0; j < int(child.ChildCount()); j++ {
				spec := child.Child(j)
				typeName := extractTypeFromDelegationSpecifier(spec, source)
				if typeName != "" {
					types = append(types, typeName)
				}
			}
		}
	}
	return types
}

func extractTypeFromDelegationSpecifier(node *sitter.Node, source []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeConstructorInvocation {
			return extractTypeFromConstructorInvocation(child, source)
		}
		if child.Type() == NodeUserType {
			return extractTypeName(node, source)
		}
	}
	return ""
}

func extractTypeFromConstructorInvocation(node *sitter.Node, source []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeUserType {
			for j := 0; j < int(child.ChildCount()); j++ {
				typeChild := child.Child(j)
				if typeChild.Type() == NodeTypeIdentifier || typeChild.Type() == NodeIdentifier {
					return typeChild.Content(source)
				}
			}
		}
	}
	return ""
}

func extractTypeName(node *sitter.Node, source []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeUserType {
			for j := 0; j < int(child.ChildCount()); j++ {
				typeChild := child.Child(j)
				if typeChild.Type() == NodeTypeIdentifier || typeChild.Type() == NodeIdentifier {
					return typeChild.Content(source)
				}
			}
		}
		if child.Type() == NodeTypeIdentifier || child.Type() == NodeIdentifier {
			return child.Content(source)
		}
	}
	return ""
}

// IsKotestSpec checks if a class extends a Kotest spec style.
func IsKotestSpec(node *sitter.Node, source []byte) (bool, string) {
	supers := GetSuperTypes(node, source)
	for _, s := range supers {
		if isKotestSpecStyle(s) {
			return true, s
		}
	}
	return false, ""
}

func isKotestSpecStyle(name string) bool {
	switch name {
	case SpecFunSpec, SpecStringSpec, SpecBehaviorSpec, SpecDescribeSpec,
		SpecWordSpec, SpecFreeSpec, SpecFeatureSpec, SpecExpectSpec,
		SpecShouldSpec, SpecAnnotationSpec:
		return true
	}
	return false
}

// GetModifiers returns the modifiers node from a class or function declaration.
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
	if modifiers == nil {
		return false
	}
	for i := 0; i < int(modifiers.ChildCount()); i++ {
		child := modifiers.Child(i)
		if child.Type() == NodeAnnotation {
			name := GetAnnotationName(child, source)
			if name == annotationName {
				return true
			}
		}
	}
	return false
}

// GetAnnotationName extracts the annotation name from an annotation node.
// Handles both simple annotations (@Test) and annotations with arguments (@Disabled("reason")).
func GetAnnotationName(annotation *sitter.Node, source []byte) string {
	for i := 0; i < int(annotation.ChildCount()); i++ {
		child := annotation.Child(i)
		if child.Type() == NodeUserType {
			return extractTypeName(annotation, source)
		}
		if child.Type() == NodeIdentifier || child.Type() == NodeTypeIdentifier {
			return child.Content(source)
		}
		// Handle annotations with arguments: @Disabled("reason") has constructor_invocation
		if child.Type() == NodeConstructorInvocation {
			return extractTypeFromConstructorInvocation(child, source)
		}
	}
	return ""
}

// GetCallExpressionName extracts the function name from a call_expression.
func GetCallExpressionName(node *sitter.Node, source []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeIdentifier {
			return child.Content(source)
		}
		if child.Type() == NodeNavigationExpression {
			return getNavigationExpressionName(child, source)
		}
	}
	return ""
}

func getNavigationExpressionName(node *sitter.Node, source []byte) string {
	for i := int(node.ChildCount()) - 1; i >= 0; i-- {
		child := node.Child(i)
		if child.Type() == NodeIdentifier {
			return child.Content(source)
		}
	}
	return ""
}

// GetCallArguments returns the value arguments from a call_expression.
func GetCallArguments(node *sitter.Node) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeCallSuffix {
			for j := 0; j < int(child.ChildCount()); j++ {
				suffix := child.Child(j)
				if suffix.Type() == NodeValueArguments {
					return suffix
				}
			}
		}
		if child.Type() == NodeValueArguments {
			return child
		}
	}
	return nil
}

// GetLambdaFromCall returns the lambda block from a call_expression.
func GetLambdaFromCall(node *sitter.Node) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeCallSuffix {
			for j := 0; j < int(child.ChildCount()); j++ {
				suffix := child.Child(j)
				if suffix.Type() == NodeAnnotatedLambda {
					return getLambdaFromAnnotated(suffix)
				}
				if suffix.Type() == NodeLambdaLiteral {
					return suffix
				}
			}
		}
		if child.Type() == NodeLambdaLiteral {
			return child
		}
	}
	return nil
}

func getLambdaFromAnnotated(node *sitter.Node) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeLambdaLiteral {
			return child
		}
	}
	return nil
}

// GetLambdaFromAnnotatedLambda extracts lambda_literal from an annotated_lambda node.
func GetLambdaFromAnnotatedLambda(node *sitter.Node) *sitter.Node {
	return getLambdaFromAnnotated(node)
}

// GetLambdaFromCallSuffix extracts lambda from a call_suffix node.
func GetLambdaFromCallSuffix(node *sitter.Node) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeLambdaLiteral {
			return child
		}
		if child.Type() == NodeAnnotatedLambda {
			return getLambdaFromAnnotated(child)
		}
	}
	return nil
}

// ExtractStringContent extracts content from a string literal node.
func ExtractStringContent(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}

	switch node.Type() {
	case NodeStringLiteral, NodeLineStringLiteral:
		text := node.Content(source)
		if len(text) >= 2 {
			if text[0] == '"' && text[len(text)-1] == '"' {
				return text[1 : len(text)-1]
			}
		}
		if strings.HasPrefix(text, `"""`) && strings.HasSuffix(text, `"""`) {
			return text[3 : len(text)-3]
		}
		return text
	case NodeLineStringContent:
		return node.Content(source)
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeLineStringContent {
			return child.Content(source)
		}
		if child.Type() == NodeLineStringLiteral || child.Type() == NodeStringLiteral {
			return ExtractStringContent(child, source)
		}
	}

	return ""
}

// GetFirstStringArgument extracts the first string argument from a call_expression.
func GetFirstStringArgument(node *sitter.Node, source []byte) string {
	args := GetCallArguments(node)
	if args == nil {
		return ""
	}

	for i := 0; i < int(args.ChildCount()); i++ {
		arg := args.Child(i)
		if arg.Type() == NodeValueArgument {
			for j := 0; j < int(arg.ChildCount()); j++ {
				val := arg.Child(j)
				if val.Type() == NodeStringLiteral || val.Type() == NodeLineStringLiteral {
					return ExtractStringContent(val, source)
				}
			}
		}
		if arg.Type() == NodeStringLiteral || arg.Type() == NodeLineStringLiteral {
			return ExtractStringContent(arg, source)
		}
	}

	return ""
}

// IsKotlinTestFile checks if the path matches Kotlin test file patterns.
// It checks both filename conventions and directory-based patterns.
func IsKotlinTestFile(path string) bool {
	// Normalize path separators
	normalizedPath := strings.ReplaceAll(path, "\\", "/")

	// Extract base filename
	base := normalizedPath
	if idx := strings.LastIndex(normalizedPath, "/"); idx >= 0 {
		base = normalizedPath[idx+1:]
	}

	// Must be a Kotlin file
	if !strings.HasSuffix(base, ".kt") && !strings.HasSuffix(base, ".kts") {
		return false
	}

	name := strings.TrimSuffix(strings.TrimSuffix(base, ".kts"), ".kt")

	// Kotlin test naming conventions: *Test.kt, *Tests.kt, *Spec.kt, Test*.kt
	if strings.HasSuffix(name, "Test") || strings.HasSuffix(name, "Tests") ||
		strings.HasSuffix(name, "Spec") || strings.HasPrefix(name, "Test") {
		return true
	}

	// Directory-based detection: files in test directories
	if strings.Contains(normalizedPath, "/test/") ||
		strings.Contains(normalizedPath, "/tests/") ||
		strings.Contains(normalizedPath, "/src/test/") {
		return true
	}

	return false
}
