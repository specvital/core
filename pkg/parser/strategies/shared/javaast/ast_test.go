package javaast

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/tspool"
)

func TestGetAnnotations(t *testing.T) {
	t.Run("nil modifiers returns nil", func(t *testing.T) {
		result := GetAnnotations(nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("extracts annotations from modifiers", func(t *testing.T) {
		source := []byte(`
public class Test {
    @Test
    @DisplayName("test")
    public void testMethod() {}
}
`)
		tree, err := tspool.Parse(context.Background(), domain.LanguageJava, source)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		defer tree.Close()

		methodNode := findNodeByType(tree.RootNode(), NodeMethodDeclaration)
		if methodNode == nil {
			t.Fatal("method declaration not found")
		}

		modifiers := GetModifiers(methodNode)
		annotations := GetAnnotations(modifiers)

		if len(annotations) != 2 {
			t.Errorf("expected 2 annotations, got %d", len(annotations))
		}
	})
}

func TestGetAnnotationName(t *testing.T) {
	t.Run("nil annotation returns empty", func(t *testing.T) {
		result := GetAnnotationName(nil, nil)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("simple annotation", func(t *testing.T) {
		source := []byte(`
class Test {
    @Test
    void testMethod() {}
}
`)
		tree, err := tspool.Parse(context.Background(), domain.LanguageJava, source)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		defer tree.Close()

		methodNode := findNodeByType(tree.RootNode(), NodeMethodDeclaration)
		modifiers := GetModifiers(methodNode)
		annotations := GetAnnotations(modifiers)

		if len(annotations) == 0 {
			t.Fatal("no annotations found")
		}

		name := GetAnnotationName(annotations[0], source)
		if name != "Test" {
			t.Errorf("expected 'Test', got %q", name)
		}
	})

	t.Run("scoped annotation extracts simple name", func(t *testing.T) {
		source := []byte(`
class Test {
    @org.junit.jupiter.api.Test
    void testMethod() {}
}
`)
		tree, err := tspool.Parse(context.Background(), domain.LanguageJava, source)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		defer tree.Close()

		methodNode := findNodeByType(tree.RootNode(), NodeMethodDeclaration)
		modifiers := GetModifiers(methodNode)
		annotations := GetAnnotations(modifiers)

		if len(annotations) == 0 {
			t.Fatal("no annotations found")
		}

		name := GetAnnotationName(annotations[0], source)
		if name != "Test" {
			t.Errorf("expected 'Test', got %q", name)
		}
	})
}

func TestGetMethodName(t *testing.T) {
	source := []byte(`
class Test {
    void myTestMethod() {}
}
`)
	tree, err := tspool.Parse(context.Background(), domain.LanguageJava, source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	defer tree.Close()

	methodNode := findNodeByType(tree.RootNode(), NodeMethodDeclaration)
	if methodNode == nil {
		t.Fatal("method not found")
	}

	name := GetMethodName(methodNode, source)
	if name != "myTestMethod" {
		t.Errorf("expected 'myTestMethod', got %q", name)
	}
}

func TestGetClassName(t *testing.T) {
	source := []byte(`
class MyTestClass {
}
`)
	tree, err := tspool.Parse(context.Background(), domain.LanguageJava, source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	defer tree.Close()

	classNode := findNodeByType(tree.RootNode(), NodeClassDeclaration)
	if classNode == nil {
		t.Fatal("class not found")
	}

	name := GetClassName(classNode, source)
	if name != "MyTestClass" {
		t.Errorf("expected 'MyTestClass', got %q", name)
	}
}

func TestHasAnnotation(t *testing.T) {
	source := []byte(`
class Test {
    @Test
    @Disabled
    void testMethod() {}
}
`)
	tree, err := tspool.Parse(context.Background(), domain.LanguageJava, source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	defer tree.Close()

	methodNode := findNodeByType(tree.RootNode(), NodeMethodDeclaration)
	modifiers := GetModifiers(methodNode)

	if !HasAnnotation(modifiers, source, "Test") {
		t.Error("expected to find @Test annotation")
	}

	if !HasAnnotation(modifiers, source, "Disabled") {
		t.Error("expected to find @Disabled annotation")
	}

	if HasAnnotation(modifiers, source, "NotPresent") {
		t.Error("should not find @NotPresent annotation")
	}

	if HasAnnotation(nil, source, "Test") {
		t.Error("nil modifiers should return false")
	}
}

func TestGetAnnotationArgument(t *testing.T) {
	t.Run("string argument", func(t *testing.T) {
		source := []byte(`
class Test {
    @DisplayName("My Test Name")
    void testMethod() {}
}
`)
		tree, err := tspool.Parse(context.Background(), domain.LanguageJava, source)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		defer tree.Close()

		methodNode := findNodeByType(tree.RootNode(), NodeMethodDeclaration)
		modifiers := GetModifiers(methodNode)
		annotations := GetAnnotations(modifiers)

		if len(annotations) == 0 {
			t.Fatal("no annotations found")
		}

		arg := GetAnnotationArgument(annotations[0], source)
		if arg != "My Test Name" {
			t.Errorf("expected 'My Test Name', got %q", arg)
		}
	})

	t.Run("no argument returns empty", func(t *testing.T) {
		source := []byte(`
class Test {
    @Test
    void testMethod() {}
}
`)
		tree, err := tspool.Parse(context.Background(), domain.LanguageJava, source)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		defer tree.Close()

		methodNode := findNodeByType(tree.RootNode(), NodeMethodDeclaration)
		modifiers := GetModifiers(methodNode)
		annotations := GetAnnotations(modifiers)

		if len(annotations) == 0 {
			t.Fatal("no annotations found")
		}

		arg := GetAnnotationArgument(annotations[0], source)
		if arg != "" {
			t.Errorf("expected empty string, got %q", arg)
		}
	})
}

// findNodeByType recursively finds the first node of the given type.
func findNodeByType(node *sitter.Node, nodeType string) *sitter.Node {
	if node.Type() == nodeType {
		return node
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if result := findNodeByType(child, nodeType); result != nil {
			return result
		}
	}
	return nil
}

func TestSanitizeSource(t *testing.T) {
	t.Run("returns original if no NULL bytes", func(t *testing.T) {
		source := []byte("public class Test { void test() {} }")
		result := SanitizeSource(source)

		// Should return same slice (no copy)
		if &result[0] != &source[0] {
			t.Error("expected same slice when no NULL bytes")
		}
	})

	t.Run("replaces NULL bytes with spaces", func(t *testing.T) {
		source := []byte("class Test { String s = \"hello\x00world\"; }")
		result := SanitizeSource(source)

		expected := []byte("class Test { String s = \"hello world\"; }")
		if string(result) != string(expected) {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("handles multiple NULL bytes", func(t *testing.T) {
		source := []byte("a\x00b\x00c\x00d")
		result := SanitizeSource(source)

		expected := []byte("a b c d")
		if string(result) != string(expected) {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("enables tree-sitter parsing of files with NULL bytes", func(t *testing.T) {
		// Simulate OSS-Fuzz test file with NULL bytes in string literal
		source := []byte(`
public class OssFuzzTest {
    @Test
    public void testMethod() {
        String data = "test` + "\x00\x00\x00" + `data";
    }
}
`)
		// Without sanitization, tree-sitter would produce ERROR root
		cleanSource := SanitizeSource(source)

		tree, err := tspool.Parse(context.Background(), domain.LanguageJava, cleanSource)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		defer tree.Close()

		root := tree.RootNode()
		if root.Type() != "program" {
			t.Errorf("expected root type 'program', got %q", root.Type())
		}

		// Should find class_declaration
		classNode := findNodeByType(root, NodeClassDeclaration)
		if classNode == nil {
			t.Error("expected to find class_declaration after sanitization")
		}
	})
}
