package rubyast

import (
	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser"
)

// ExtractStringContent removes surrounding quotes from string nodes.
func ExtractStringContent(node *sitter.Node, source []byte) string {
	text := parser.GetNodeText(node, source)
	if len(text) >= 2 {
		if (text[0] == '"' && text[len(text)-1] == '"') ||
			(text[0] == '\'' && text[len(text)-1] == '\'') {
			return text[1 : len(text)-1]
		}
	}
	return text
}

// ExtractSymbolContent removes leading colon from symbol nodes.
func ExtractSymbolContent(node *sitter.Node, source []byte) string {
	text := parser.GetNodeText(node, source)
	if len(text) > 0 && text[0] == ':' {
		return text[1:]
	}
	return text
}

// FindBlock returns the first block or do_block child of a node.
func FindBlock(node *sitter.Node) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == NodeBlock || child.Type() == NodeDoBlock {
			return child
		}
	}
	return nil
}

// AddTestToTarget adds a test to either a suite or file.
func AddTestToTarget(test domain.Test, parentSuite *domain.TestSuite, file *domain.TestFile) {
	if parentSuite != nil {
		parentSuite.Tests = append(parentSuite.Tests, test)
	} else {
		file.Tests = append(file.Tests, test)
	}
}

// AddSuiteToTarget adds a suite to either a parent suite or file.
func AddSuiteToTarget(suite domain.TestSuite, parentSuite *domain.TestSuite, file *domain.TestFile) {
	if parentSuite != nil {
		parentSuite.Suites = append(parentSuite.Suites, suite)
	} else {
		file.Suites = append(file.Suites, suite)
	}
}
