package jest

import (
	"regexp"
	"strconv"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/domain"
	"github.com/specvital/core/parser"
)

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

var skippedFunctionAliases = map[string]string{
	"xdescribe": funcDescribe,
	"xit":       funcIt,
	"xtest":     funcTest,
}

var focusedFunctionAliases = map[string]string{
	"fdescribe": funcDescribe,
	"fit":       funcIt,
}

var jestPlaceholderPattern = regexp.MustCompile(`%[sdpji#%]`)

func unquoteString(text string) string {
	if len(text) < 2 {
		return text
	}

	// Handle template literals, which are not supported by strconv.Unquote.
	if text[0] == '`' && text[len(text)-1] == '`' {
		return text[1 : len(text)-1]
	}

	// Handle single-quoted strings (JavaScript style).
	// strconv.Unquote only handles double-quoted strings and Go-style rune literals.
	if text[0] == '\'' && text[len(text)-1] == '\'' {
		inner := text[1 : len(text)-1]
		// Handle escaped single quotes before converting to a double-quoted string.
		inner = strings.ReplaceAll(inner, `\'`, `'`)
		// Escape any unescaped double quotes in the content for strconv.Unquote
		escaped := strings.ReplaceAll(inner, `"`, `\"`)
		converted := `"` + escaped + `"`
		if s, err := strconv.Unquote(converted); err == nil {
			return s
		}
		// Fallback to original text on failure to avoid returning a partially processed string.
		return text
	}

	if s, err := strconv.Unquote(text); err == nil {
		return s
	}

	return text
}

func formatEachName(template, data string) string {
	args := strings.Split(data, ", ")
	argIndex := 0

	result := jestPlaceholderPattern.ReplaceAllStringFunc(template, func(match string) string {
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

func extractArrayElements(arrayNode *sitter.Node, source []byte) []string {
	var elements []string

	for i := 0; i < int(arrayNode.ChildCount()); i++ {
		elem := arrayNode.Child(i)
		switch elem.Type() {
		case "array":
			elements = append(elements, extractArrayContent(elem, source))
		case "string":
			elements = append(elements, unquoteString(parser.GetNodeText(elem, source)))
		case "number":
			elements = append(elements, parser.GetNodeText(elem, source))
		}
	}

	return elements
}

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

func parseFunctionName(node *sitter.Node, source []byte) (string, domain.TestStatus) {
	switch node.Type() {
	case "identifier":
		return parseIdentifierFunction(node, source)
	case "member_expression":
		return parseMemberExpressionFunction(node, source)
	default:
		return "", domain.TestStatusPending
	}
}
