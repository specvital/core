package jstest

import (
	"strconv"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser"
)

func UnquoteString(text string) string {
	if len(text) < 2 {
		return text
	}

	if text[0] == '`' && text[len(text)-1] == '`' {
		return text[1 : len(text)-1]
	}

	// Handle single-quoted JavaScript strings.
	// Go's strconv.Unquote only handles double-quoted strings, so we need to
	// convert single-quoted strings to double-quoted format first:
	// 1. Remove outer single quotes and get the inner content
	// 2. Unescape JavaScript's escaped single quotes (\' -> ')
	// 3. Escape any double quotes for Go's strconv.Unquote
	// 4. Wrap in double quotes and parse with strconv.Unquote
	if text[0] == '\'' && text[len(text)-1] == '\'' {
		inner := text[1 : len(text)-1]
		inner = strings.ReplaceAll(inner, `\'`, `'`)
		escaped := strings.ReplaceAll(inner, `"`, `\"`)
		converted := `"` + escaped + `"`
		if s, err := strconv.Unquote(converted); err == nil {
			return s
		}
		return text
	}

	if s, err := strconv.Unquote(text); err == nil {
		return s
	}

	return text
}

func FormatEachName(template, data string) string {
	args := strings.Split(data, ", ")
	argIndex := 0

	result := JestPlaceholderPattern.ReplaceAllStringFunc(template, func(match string) string {
		if match == "%%" {
			return "%"
		}
		if argIndex < len(args) {
			arg := args[argIndex]
			argIndex++
			return arg
		}
		return match
	})

	return result
}

// ExtractStringValue extracts a string value from a node if it's a string literal.
// Returns empty string for non-string nodes (identifiers, expressions, etc.).
func ExtractStringValue(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}

	switch node.Type() {
	case "string", "template_string":
		return UnquoteString(parser.GetNodeText(node, source))
	default:
		return ""
	}
}

func ExtractArrayContent(node *sitter.Node, source []byte) string {
	var parts []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "string":
			parts = append(parts, UnquoteString(parser.GetNodeText(child, source)))
		case "number":
			parts = append(parts, parser.GetNodeText(child, source))
		}
	}

	return strings.Join(parts, ", ")
}

func ExtractArrayElements(arrayNode *sitter.Node, source []byte) []string {
	var elements []string

	for i := 0; i < int(arrayNode.ChildCount()); i++ {
		elem := arrayNode.Child(i)
		switch elem.Type() {
		case "array":
			elements = append(elements, ExtractArrayContent(elem, source))
		case "number":
			elements = append(elements, parser.GetNodeText(elem, source))
		case "object":
			elements = append(elements, ObjectPlaceholder)
		case "string":
			elements = append(elements, UnquoteString(parser.GetNodeText(elem, source)))
		}
	}

	return elements
}

func ExtractEachTestCases(args *sitter.Node, source []byte) []string {
	var cases []string

	for i := 0; i < int(args.ChildCount()); i++ {
		child := args.Child(i)
		if child.Type() == "array" {
			cases = ExtractArrayElements(child, source)
			break
		}
	}

	return cases
}

func FindCallback(args *sitter.Node) *sitter.Node {
	for i := 0; i < int(args.ChildCount()); i++ {
		child := args.Child(i)
		switch child.Type() {
		case "arrow_function", "function_expression", "function":
			return child
		}
	}
	return nil
}

// FindLastCallback finds the last argument that is a function (arrow_function, function_expression, function).
// This is used to detect callbacks in custom wrapper functions like describeMatrix() or describeIf().
func FindLastCallback(args *sitter.Node) *sitter.Node {
	var lastCallback *sitter.Node
	for i := 0; i < int(args.ChildCount()); i++ {
		child := args.Child(i)
		switch child.Type() {
		case "arrow_function", "function_expression", "function":
			lastCallback = child
		}
	}
	return lastCallback
}

func ExtractTestName(args *sitter.Node, source []byte) string {
	for i := 0; i < int(args.ChildCount()); i++ {
		child := args.Child(i)
		switch child.Type() {
		case "string", "template_string":
			return UnquoteString(parser.GetNodeText(child, source))
		case "identifier", "binary_expression", "call_expression":
			return DynamicNamePlaceholder
		}
	}
	return ""
}

// IsFirstArgString checks if the first argument in arguments node is a string literal.
// Returns false for expressions like `test.skip(condition, message)`.
// Returns true for test definitions like `test.skip('name', callback)`.
func IsFirstArgString(args *sitter.Node) bool {
	for i := 0; i < int(args.ChildCount()); i++ {
		child := args.Child(i)
		switch child.Type() {
		case "string", "template_string":
			return true
		case "(", ")", ",":
			continue
		default:
			return false
		}
	}
	return false
}

func ParseSimpleMemberExpression(obj, prop *sitter.Node, source []byte) (string, domain.TestStatus, string) {
	objName := parser.GetNodeText(obj, source)
	propName := parser.GetNodeText(prop, source)

	switch propName {
	case ModifierConcurrent:
		return objName, domain.TestStatusActive, ""
	case ModifierEach:
		return objName + "." + ModifierEach, domain.TestStatusActive, ""
	case ModifierOnly:
		return objName, domain.TestStatusFocused, ModifierOnly
	case ModifierSkip:
		return objName, domain.TestStatusSkipped, ModifierSkip
	case ModifierTodo:
		return objName, domain.TestStatusTodo, ModifierTodo
	default:
		return "", domain.TestStatusActive, ""
	}
}

func ParseNestedMemberExpression(obj, prop *sitter.Node, source []byte) (string, domain.TestStatus, string) {
	innerObj := obj.ChildByFieldName("object")
	innerProp := obj.ChildByFieldName("property")

	if innerObj == nil || innerProp == nil {
		return "", domain.TestStatusActive, ""
	}

	objName := parser.GetNodeText(innerObj, source)
	middleProp := parser.GetNodeText(innerProp, source)
	propName := parser.GetNodeText(prop, source)

	// Handle test.concurrent.skip, describe.concurrent.only, etc.
	if middleProp == ModifierConcurrent {
		status := ParseModifierStatus(propName)
		modifier := ""
		if status != domain.TestStatusActive {
			modifier = propName
		}
		if propName == ModifierEach {
			return objName + "." + ModifierEach, domain.TestStatusActive, ""
		}
		return objName, status, modifier
	}

	status := ParseModifierStatus(middleProp)
	modifier := ""
	if status != domain.TestStatusActive {
		modifier = middleProp
	}

	if propName == ModifierEach {
		return objName + "." + ModifierEach, status, modifier
	}

	return "", status, modifier
}

func ParseMemberExpressionFunction(node *sitter.Node, source []byte) (string, domain.TestStatus, string) {
	obj := node.ChildByFieldName("object")
	prop := node.ChildByFieldName("property")

	if obj == nil || prop == nil {
		return "", domain.TestStatusActive, ""
	}

	if obj.Type() == "member_expression" {
		return ParseNestedMemberExpression(obj, prop, source)
	}

	return ParseSimpleMemberExpression(obj, prop, source)
}

func ParseIdentifierFunction(node *sitter.Node, source []byte) (string, domain.TestStatus, string) {
	name := parser.GetNodeText(node, source)

	if baseName, ok := SkippedFunctionAliases[name]; ok {
		return baseName, domain.TestStatusSkipped, name
	}

	if baseName, ok := FocusedFunctionAliases[name]; ok {
		return baseName, domain.TestStatusFocused, name
	}

	return name, domain.TestStatusActive, ""
}

func ParseFunctionName(node *sitter.Node, source []byte) (string, domain.TestStatus, string) {
	switch node.Type() {
	case "identifier":
		return ParseIdentifierFunction(node, source)
	case "member_expression":
		return ParseMemberExpressionFunction(node, source)
	case "parenthesized_expression":
		return parseParenthesizedFunction(node, source)
	default:
		return "", domain.TestStatusActive, ""
	}
}

func parseParenthesizedFunction(node *sitter.Node, source []byte) (string, domain.TestStatus, string) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "ternary_expression":
			return parseConditionalFunction(child, source)
		case "identifier", "member_expression", "parenthesized_expression":
			return ParseFunctionName(child, source)
		}
	}
	return "", domain.TestStatusActive, ""
}

func parseConditionalFunction(node *sitter.Node, source []byte) (string, domain.TestStatus, string) {
	consequence := node.ChildByFieldName("consequence")
	if consequence != nil {
		if name, _, _ := ParseFunctionName(consequence, source); name != "" {
			return name, domain.TestStatusActive, ""
		}
	}

	alternative := node.ChildByFieldName("alternative")
	if alternative != nil {
		if name, _, _ := ParseFunctionName(alternative, source); name != "" {
			return name, domain.TestStatusActive, ""
		}
	}

	return "", domain.TestStatusActive, ""
}
