package phpast

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/php"
)

func TestExtendsTestCase(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "extends TestCase",
			content:  `<?php class MyTest extends TestCase {}`,
			expected: true,
		},
		{
			name:     "extends BaseTestCase",
			content:  `<?php class MyTest extends BaseTestCase {}`,
			expected: true,
		},
		{
			name:     "extends qualified PHPUnit TestCase",
			content:  `<?php class MyTest extends \PHPUnit\Framework\TestCase {}`,
			expected: true,
		},
		{
			name:     "extends SomeTestCaseHelper (not suffix)",
			content:  `<?php class MyTest extends SomeTestCaseHelper {}`,
			expected: false,
		},
		{
			name:     "extends TestCaseMixin (not suffix)",
			content:  `<?php class MyTest extends TestCaseMixin {}`,
			expected: false,
		},
		{
			name:     "no extends clause",
			content:  `<?php class MyTest {}`,
			expected: false,
		},
		{
			name:     "extends unrelated class",
			content:  `<?php class MyTest extends Controller {}`,
			expected: false,
		},
	}

	parser := sitter.NewParser()
	parser.SetLanguage(php.GetLanguage())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := []byte(tt.content)
			tree, err := parser.ParseCtx(context.Background(), nil, source)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			root := tree.RootNode()
			var classNode *sitter.Node
			for i := 0; i < int(root.ChildCount()); i++ {
				child := root.Child(i)
				if child.Type() == NodeClassDeclaration {
					classNode = child
					break
				}
			}
			if classNode == nil {
				t.Fatal("class node not found")
			}

			result := ExtendsTestCase(classNode, source)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestExtractBaseClassName(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple name",
			content:  `<?php class MyTest extends TestCase {}`,
			expected: "TestCase",
		},
		{
			name:     "qualified name",
			content:  `<?php class MyTest extends \PHPUnit\Framework\TestCase {}`,
			expected: "TestCase",
		},
		{
			name:     "no backslash prefix",
			content:  `<?php class MyTest extends PHPUnit\Framework\TestCase {}`,
			expected: "TestCase",
		},
	}

	parser := sitter.NewParser()
	parser.SetLanguage(php.GetLanguage())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := []byte(tt.content)
			tree, err := parser.ParseCtx(context.Background(), nil, source)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			root := tree.RootNode()
			var baseClause *sitter.Node
			for i := 0; i < int(root.ChildCount()); i++ {
				child := root.Child(i)
				if child.Type() == NodeClassDeclaration {
					for j := 0; j < int(child.ChildCount()); j++ {
						grandchild := child.Child(j)
						if grandchild.Type() == NodeBaseClause {
							baseClause = grandchild
							break
						}
					}
					break
				}
			}
			if baseClause == nil {
				t.Fatal("base_clause not found")
			}

			result := extractBaseClassName(baseClause, source)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
