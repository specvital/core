package dotnetast

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/csharp"
)

func parseCS(t *testing.T, source string) *sitter.Node {
	t.Helper()
	parser := sitter.NewParser()
	parser.SetLanguage(csharp.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, []byte(source))
	if err != nil {
		t.Fatalf("failed to parse C# source: %v", err)
	}
	return tree.RootNode()
}

func TestGetClassName(t *testing.T) {
	source := `public class MyTestClass { }`
	root := parseCS(t, source)

	var className string
	walkTree(root, func(n *sitter.Node) bool {
		if n.Type() == NodeClassDeclaration {
			className = GetClassName(n, []byte(source))
			return false
		}
		return true
	})

	if className != "MyTestClass" {
		t.Errorf("expected 'MyTestClass', got '%s'", className)
	}
}

func TestGetMethodName(t *testing.T) {
	source := `public class C { public void TestMethod() { } }`
	root := parseCS(t, source)

	var methodName string
	walkTree(root, func(n *sitter.Node) bool {
		if n.Type() == NodeMethodDeclaration {
			methodName = GetMethodName(n, []byte(source))
			return false
		}
		return true
	})

	if methodName != "TestMethod" {
		t.Errorf("expected 'TestMethod', got '%s'", methodName)
	}
}

func TestGetAttributeName(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name:     "simple attribute",
			source:   `public class C { [Fact] public void Test() { } }`,
			expected: "Fact",
		},
		{
			name:     "qualified attribute",
			source:   `public class C { [Xunit.Fact] public void Test() { } }`,
			expected: "Fact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := parseCS(t, tt.source)

			var attrName string
			walkTree(root, func(n *sitter.Node) bool {
				if n.Type() == NodeAttribute {
					attrName = GetAttributeName(n, []byte(tt.source))
					return false
				}
				return true
			})

			if attrName != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, attrName)
			}
		})
	}
}

func TestHasAttribute(t *testing.T) {
	source := `public class C { [Fact] [Theory] public void Test() { } }`
	root := parseCS(t, source)

	var attrLists []*sitter.Node
	walkTree(root, func(n *sitter.Node) bool {
		if n.Type() == NodeMethodDeclaration {
			attrLists = GetAttributeLists(n)
			return false
		}
		return true
	})

	if !HasAttribute(attrLists, []byte(source), "Fact") {
		t.Error("expected to find Fact attribute")
	}
	if !HasAttribute(attrLists, []byte(source), "Theory") {
		t.Error("expected to find Theory attribute")
	}
	if HasAttribute(attrLists, []byte(source), "Skip") {
		t.Error("should not find Skip attribute")
	}
}

func TestGetDeclarationList(t *testing.T) {
	source := `public class C { public void M() { } }`
	root := parseCS(t, source)

	var body *sitter.Node
	walkTree(root, func(n *sitter.Node) bool {
		if n.Type() == NodeClassDeclaration {
			body = GetDeclarationList(n)
			return false
		}
		return true
	})

	if body == nil {
		t.Error("expected non-nil declaration list")
	}
	if body.Type() != NodeDeclarationList {
		t.Errorf("expected declaration_list, got '%s'", body.Type())
	}
}

func walkTree(node *sitter.Node, visitor func(*sitter.Node) bool) {
	if !visitor(node) {
		return
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		walkTree(node.Child(i), visitor)
	}
}

func TestGetDeclarationChildren(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		expectedTypes  []string
		expectedNames  []string
	}{
		{
			name: "simple class with methods",
			source: `public class C {
    public void Test1() { }
    public void Test2() { }
}`,
			expectedTypes: []string{"method_declaration", "method_declaration"},
			expectedNames: []string{"Test1", "Test2"},
		},
		{
			name: "class with preprocessor #if directive",
			source: `public class C {
#if NET6_0
    public void Net6Test() { }
#endif
    public void CommonTest() { }
}`,
			expectedTypes: []string{"method_declaration", "method_declaration"},
			expectedNames: []string{"Net6Test", "CommonTest"},
		},
		{
			name: "nested class inside preprocessor",
			source: `public class C {
#if NET6_0
    public class Nested {
        public void Test() { }
    }
#endif
    public void CommonTest() { }
}`,
			expectedTypes: []string{"class_declaration", "method_declaration"},
			expectedNames: []string{"Nested", "CommonTest"},
		},
		{
			name: "preprocessor #if with #else",
			source: `public class C {
#if NETFRAMEWORK
    public void FrameworkTest() { }
#else
    public void CoreTest() { }
#endif
}`,
			expectedTypes: []string{"method_declaration", "method_declaration"},
			expectedNames: []string{"FrameworkTest", "CoreTest"},
		},
		{
			name: "preprocessor #if with #elif and #else",
			source: `public class C {
#if NET8_0
    public void Net8Test() { }
#elif NET6_0
    public void Net6Test() { }
#else
    public void LegacyTest() { }
#endif
}`,
			expectedTypes: []string{"method_declaration", "method_declaration", "method_declaration"},
			expectedNames: []string{"Net8Test", "Net6Test", "LegacyTest"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := parseCS(t, tt.source)

			var body *sitter.Node
			walkTree(root, func(n *sitter.Node) bool {
				if n.Type() == NodeClassDeclaration {
					body = GetDeclarationList(n)
					return false
				}
				return true
			})

			if body == nil {
				t.Fatal("expected non-nil declaration list")
			}

			children := GetDeclarationChildren(body)
			if len(children) != len(tt.expectedTypes) {
				t.Fatalf("expected %d children, got %d", len(tt.expectedTypes), len(children))
			}

			for i, child := range children {
				if child.Type() != tt.expectedTypes[i] {
					t.Errorf("child[%d]: expected type '%s', got '%s'", i, tt.expectedTypes[i], child.Type())
				}

				var name string
				switch child.Type() {
				case NodeMethodDeclaration:
					name = GetMethodName(child, []byte(tt.source))
				case NodeClassDeclaration:
					name = GetClassName(child, []byte(tt.source))
				}
				if name != tt.expectedNames[i] {
					t.Errorf("child[%d]: expected name '%s', got '%s'", i, tt.expectedNames[i], name)
				}
			}
		})
	}
}

func TestIsPreprocessorDirective(t *testing.T) {
	tests := []struct {
		nodeType string
		expected bool
	}{
		{"preproc_if", true},
		{"preproc_else", true},
		{"preproc_elif", true},
		{"class_declaration", false},
		{"method_declaration", false},
		{"preproc_endif", false},
	}

	for _, tt := range tests {
		t.Run(tt.nodeType, func(t *testing.T) {
			result := IsPreprocessorDirective(tt.nodeType)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetAttributeLists_PreprocessorLimitation(t *testing.T) {
	// This test documents the tree-sitter-c-sharp limitation where
	// preprocessor directives between attributes are parsed as ERROR nodes.
	// See GetAttributeLists function comment for details.

	t.Run("attributes without preprocessor work correctly", func(t *testing.T) {
		source := `public class C {
    [Fact]
    [Theory]
    [InlineData(1)]
    public void Test() { }
}`
		root := parseCS(t, source)

		var attrLists []*sitter.Node
		walkTree(root, func(n *sitter.Node) bool {
			if n.Type() == NodeMethodDeclaration {
				attrLists = GetAttributeLists(n)
				return false
			}
			return true
		})

		if len(attrLists) != 3 {
			t.Errorf("expected 3 attribute lists, got %d", len(attrLists))
		}
	})

	t.Run("preprocessor between attributes causes ERROR node (known limitation)", func(t *testing.T) {
		source := `public class C {
    [Theory]
    [InlineData(1)]
#if NET6_0
    [InlineData(2)]
#endif
    public void Test(int x) { }
}`
		root := parseCS(t, source)

		// Verify that tree-sitter parses this with an ERROR node
		hasError := false
		walkTree(root, func(n *sitter.Node) bool {
			if n.Type() == "ERROR" {
				hasError = true
				return false
			}
			return true
		})

		if !hasError {
			t.Error("expected ERROR node in AST due to preprocessor between attributes")
		}

		// GetAttributeLists will only return attributes before the ERROR
		var attrLists []*sitter.Node
		walkTree(root, func(n *sitter.Node) bool {
			if n.Type() == NodeMethodDeclaration {
				attrLists = GetAttributeLists(n)
				return false
			}
			return true
		})

		// Due to the limitation, we only get 2 attributes (before #if)
		if len(attrLists) != 2 {
			t.Errorf("expected 2 attribute lists (limitation), got %d", len(attrLists))
		}
	})
}
