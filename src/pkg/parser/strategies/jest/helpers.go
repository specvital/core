package jest

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/domain"
	"github.com/specvital/core/parser"
)

// Jest function names and modifiers.
const (
	frameworkName = "jest"

	funcDescribe = "describe"
	funcIt       = "it"
	funcTest     = "test"

	modifierSkip = "skip"
	modifierOnly = "only"
	modifierEach = "each"
	modifierTodo = "todo"

	dynamicCasesSuffix = " (dynamic cases)"
)

// Skipped function aliases (x-prefixed). cspell:disable
var skippedFunctionAliases = map[string]string{
	"xdescribe": funcDescribe,
	"xit":       funcIt,
	"xtest":     funcTest,
}

// Focused function aliases (f-prefixed).
var focusedFunctionAliases = map[string]string{
	"fdescribe": funcDescribe,
	"fit":       funcIt,
} // cspell:enable

// Jest placeholder formats for .each() templates.
var eachPlaceholders = []string{"%s", "%d", "%p", "%j", "%i", "%#", "%%"}

// parseFunctionName extracts function name and test status from call expression.
func parseFunctionName(node *sitter.Node, source []byte) (name string, status domain.TestStatus) {
	switch node.Type() {
	case "identifier":
		return parseIdentifierFunction(node, source)
	case "member_expression":
		return parseMemberExpressionFunction(node, source)
	default:
		return "", domain.TestStatusPending
	}
}

// parseIdentifierFunction parses simple function identifiers like describe, it, xdescribe.
func parseIdentifierFunction(node *sitter.Node, source []byte) (string, domain.TestStatus) {
	name := parser.GetNodeText(node, source)

	if baseName, ok := skippedFunctionAliases[name]; ok {
		return baseName, domain.TestStatusSkipped
	}

	if baseName, ok := focusedFunctionAliases[name]; ok {
		return baseName, domain.TestStatusOnly
	}

	return name, domain.TestStatusPending
}

// parseMemberExpressionFunction parses member expressions like describe.skip, it.each.
func parseMemberExpressionFunction(node *sitter.Node, source []byte) (string, domain.TestStatus) {
	obj := node.ChildByFieldName("object")
	prop := node.ChildByFieldName("property")

	if obj == nil || prop == nil {
		return "", domain.TestStatusPending
	}

	if obj.Type() == "member_expression" {
		return parseNestedMemberExpression(obj, prop, source)
	}

	return parseSimpleMemberExpression(obj, prop, source)
}

// parseNestedMemberExpression handles describe.skip.each, it.only.each patterns.
func parseNestedMemberExpression(obj, prop *sitter.Node, source []byte) (string, domain.TestStatus) {
	innerObj := obj.ChildByFieldName("object")
	innerProp := obj.ChildByFieldName("property")

	if innerObj == nil || innerProp == nil {
		return "", domain.TestStatusPending
	}

	objName := parser.GetNodeText(innerObj, source)
	middleProp := parser.GetNodeText(innerProp, source)
	propName := parser.GetNodeText(prop, source)

	status := parseModifierStatus(middleProp)

	if propName == modifierEach {
		return objName + "." + modifierEach, status
	}

	return "", status
}

// parseSimpleMemberExpression handles describe.skip, it.only, test.each patterns.
func parseSimpleMemberExpression(obj, prop *sitter.Node, source []byte) (string, domain.TestStatus) {
	objName := parser.GetNodeText(obj, source)
	propName := parser.GetNodeText(prop, source)

	switch propName {
	case modifierSkip, modifierTodo:
		return objName, domain.TestStatusSkipped
	case modifierOnly:
		return objName, domain.TestStatusOnly
	case modifierEach:
		return objName + "." + modifierEach, domain.TestStatusPending
	default:
		return "", domain.TestStatusPending
	}
}

// parseModifierStatus returns the test status for a modifier.
func parseModifierStatus(modifier string) domain.TestStatus {
	switch modifier {
	case modifierSkip, modifierTodo:
		return domain.TestStatusSkipped
	case modifierOnly:
		return domain.TestStatusOnly
	default:
		return domain.TestStatusPending
	}
}

// extractTestName extracts the test name from arguments.
func extractTestName(args *sitter.Node, source []byte) string {
	for i := 0; i < int(args.ChildCount()); i++ {
		child := args.Child(i)
		switch child.Type() {
		case "string", "template_string":
			return unquoteString(parser.GetNodeText(child, source))
		}
	}
	return ""
}

// unquoteString removes surrounding quotes from a string.
func unquoteString(text string) string {
	const minQuotedLen = 2
	if len(text) < minQuotedLen {
		return text
	}

	first, last := text[0], text[len(text)-1]
	if (first == '"' && last == '"') ||
		(first == '\'' && last == '\'') ||
		(first == '`' && last == '`') {
		return text[1 : len(text)-1]
	}

	return text
}

// findCallback finds the callback function in arguments.
func findCallback(args *sitter.Node) *sitter.Node {
	for i := 0; i < int(args.ChildCount()); i++ {
		child := args.Child(i)
		switch child.Type() {
		case "arrow_function", "function_expression", "function":
			return child
		}
	}
	return nil
}

// extractEachTestCases extracts test case data from .each() call arguments.
func extractEachTestCases(args *sitter.Node, source []byte) []string {
	var cases []string

	for i := 0; i < int(args.ChildCount()); i++ {
		child := args.Child(i)
		if child.Type() == "array" {
			cases = extractArrayElements(child, source)
			break
		}
	}

	return cases
}

// extractArrayElements extracts elements from an array node.
func extractArrayElements(arrayNode *sitter.Node, source []byte) []string {
	var elements []string

	for i := 0; i < int(arrayNode.ChildCount()); i++ {
		elem := arrayNode.Child(i)
		switch elem.Type() {
		case "array":
			elements = append(elements, extractArrayContent(elem, source))
		case "string", "number":
			elements = append(elements, parser.GetNodeText(elem, source))
		}
	}

	return elements
}

// extractArrayContent extracts content from nested array for .each formatting.
func extractArrayContent(node *sitter.Node, source []byte) string {
	var parts []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "string":
			parts = append(parts, unquoteString(parser.GetNodeText(child, source)))
		case "number":
			parts = append(parts, parser.GetNodeText(child, source))
		}
	}

	return strings.Join(parts, ", ")
}

// formatEachName formats test name template with data.
func formatEachName(template, data string) string {
	for _, placeholder := range eachPlaceholders {
		if strings.Contains(template, placeholder) {
			return strings.Replace(template, placeholder, data, 1)
		}
	}
	return template
}
