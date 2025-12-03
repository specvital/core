package jstest

import (
	"strconv"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/domain"
	"github.com/specvital/core/parser"
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
		case "string":
			elements = append(elements, UnquoteString(parser.GetNodeText(elem, source)))
		case "number":
			elements = append(elements, parser.GetNodeText(elem, source))
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

func ExtractTestName(args *sitter.Node, source []byte) string {
	for i := 0; i < int(args.ChildCount()); i++ {
		child := args.Child(i)
		switch child.Type() {
		case "string", "template_string":
			return UnquoteString(parser.GetNodeText(child, source))
		}
	}
	return ""
}

func ParseSimpleMemberExpression(obj, prop *sitter.Node, source []byte) (string, domain.TestStatus) {
	objName := parser.GetNodeText(obj, source)
	propName := parser.GetNodeText(prop, source)

	switch propName {
	case ModifierSkip, ModifierTodo:
		return objName, domain.TestStatusSkipped
	case ModifierOnly:
		return objName, domain.TestStatusOnly
	case ModifierEach:
		return objName + "." + ModifierEach, domain.TestStatusPending
	default:
		return "", domain.TestStatusPending
	}
}

func ParseNestedMemberExpression(obj, prop *sitter.Node, source []byte) (string, domain.TestStatus) {
	innerObj := obj.ChildByFieldName("object")
	innerProp := obj.ChildByFieldName("property")

	if innerObj == nil || innerProp == nil {
		return "", domain.TestStatusPending
	}

	objName := parser.GetNodeText(innerObj, source)
	middleProp := parser.GetNodeText(innerProp, source)
	propName := parser.GetNodeText(prop, source)

	status := ParseModifierStatus(middleProp)

	if propName == ModifierEach {
		return objName + "." + ModifierEach, status
	}

	return "", status
}

func ParseMemberExpressionFunction(node *sitter.Node, source []byte) (string, domain.TestStatus) {
	obj := node.ChildByFieldName("object")
	prop := node.ChildByFieldName("property")

	if obj == nil || prop == nil {
		return "", domain.TestStatusPending
	}

	if obj.Type() == "member_expression" {
		return ParseNestedMemberExpression(obj, prop, source)
	}

	return ParseSimpleMemberExpression(obj, prop, source)
}

func ParseIdentifierFunction(node *sitter.Node, source []byte) (string, domain.TestStatus) {
	name := parser.GetNodeText(node, source)

	if baseName, ok := SkippedFunctionAliases[name]; ok {
		return baseName, domain.TestStatusSkipped
	}

	if baseName, ok := FocusedFunctionAliases[name]; ok {
		return baseName, domain.TestStatusOnly
	}

	return name, domain.TestStatusPending
}

func ParseFunctionName(node *sitter.Node, source []byte) (string, domain.TestStatus) {
	switch node.Type() {
	case "identifier":
		return ParseIdentifierFunction(node, source)
	case "member_expression":
		return ParseMemberExpressionFunction(node, source)
	default:
		return "", domain.TestStatusPending
	}
}
