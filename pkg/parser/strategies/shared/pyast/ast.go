// Package pyast provides shared Python AST traversal utilities for test framework parsers.
package pyast

import sitter "github.com/smacker/go-tree-sitter"

// Python AST node types.
const (
	NodeClassDefinition     = "class_definition"
	NodeDecorator           = "decorator"
	NodeDecoratedDefinition = "decorated_definition"
	NodeFunctionDefinition  = "function_definition"
)

// GetDecoratedDefinition extracts the actual definition from a decorated_definition node.
func GetDecoratedDefinition(node *sitter.Node) *sitter.Node {
	definition := node.ChildByFieldName("definition")
	if definition != nil {
		return definition
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeFunctionDefinition || child.Type() == NodeClassDefinition {
			return child
		}
	}
	return nil
}

// GetDecorators extracts all decorator nodes from a decorated_definition.
func GetDecorators(node *sitter.Node) []*sitter.Node {
	var decorators []*sitter.Node
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeDecorator {
			decorators = append(decorators, child)
		}
	}
	return decorators
}
