package parser

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/domain"
)

func TestNewTSParser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		lang domain.Language
	}{
		{"should create parser for TypeScript", domain.LanguageTypeScript},
		{"should create parser for JavaScript", domain.LanguageJavaScript},
		{"should default to TypeScript for unknown language", domain.LanguageGo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// When
			p := NewTSParser(tt.lang)
			defer p.Close()

			// Then
			if p == nil {
				t.Fatal("NewTSParser returned nil")
			}
			if p.lang != tt.lang {
				t.Errorf("lang = %q, want %q", p.lang, tt.lang)
			}
		})
	}
}

func TestTSParser_Parse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		lang     domain.Language
		source   string
		wantType string
	}{
		{
			name:     "should parse TypeScript code",
			lang:     domain.LanguageTypeScript,
			source:   `const x = 1;`,
			wantType: "program",
		},
		{
			name:     "should parse JavaScript code",
			lang:     domain.LanguageJavaScript,
			source:   `function hello() { return 1; }`,
			wantType: "program",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Given
			p := NewTSParser(tt.lang)
			defer p.Close()

			// When
			root, err := p.Parse([]byte(tt.source))

			// Then
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if root == nil {
				t.Fatal("Parse returned nil root")
			}
			if root.Type() != tt.wantType {
				t.Errorf("Root type = %q, want %q", root.Type(), tt.wantType)
			}
		})
	}
}

func TestGetNodeText(t *testing.T) {
	t.Parallel()

	// Given
	p := NewTSParser(domain.LanguageTypeScript)
	defer p.Close()
	source := []byte(`const hello = "world";`)
	root, _ := p.Parse(source)

	// When
	text := GetNodeText(root, source)

	// Then
	want := `const hello = "world";`
	if text != want {
		t.Errorf("GetNodeText = %q, want %q", text, want)
	}
}

func TestGetLocation(t *testing.T) {
	t.Parallel()

	// Given
	p := NewTSParser(domain.LanguageTypeScript)
	defer p.Close()
	source := []byte("line1\nline2\nline3")
	root, _ := p.Parse(source)

	// When
	loc := GetLocation(root, "test.ts")

	// Then
	if loc.File != "test.ts" {
		t.Errorf("File = %q, want %q", loc.File, "test.ts")
	}
	if loc.StartLine != 1 {
		t.Errorf("StartLine = %d, want 1", loc.StartLine)
	}
	if loc.EndLine != 3 {
		t.Errorf("EndLine = %d, want 3", loc.EndLine)
	}
}

func TestFindChildByType(t *testing.T) {
	t.Parallel()

	// Given
	p := NewTSParser(domain.LanguageTypeScript)
	defer p.Close()
	source := []byte(`const x = 1;`)
	root, _ := p.Parse(source)

	t.Run("should find existing child type", func(t *testing.T) {
		t.Parallel()

		// When
		child := FindChildByType(root, "lexical_declaration")

		// Then
		if child == nil {
			t.Fatal("FindChildByType returned nil")
		}
		if child.Type() != "lexical_declaration" {
			t.Errorf("Type = %q, want %q", child.Type(), "lexical_declaration")
		}
	})

	t.Run("should return nil for nonexistent type", func(t *testing.T) {
		t.Parallel()

		// When
		notFound := FindChildByType(root, "nonexistent")

		// Then
		if notFound != nil {
			t.Error("Expected nil for nonexistent type")
		}
	})
}

func TestFindChildrenByType(t *testing.T) {
	t.Parallel()

	// Given
	p := NewTSParser(domain.LanguageTypeScript)
	defer p.Close()
	source := []byte(`const x = 1; const y = 2;`)
	root, _ := p.Parse(source)

	// When
	children := FindChildrenByType(root, "lexical_declaration")

	// Then
	if len(children) != 2 {
		t.Errorf("len(children) = %d, want 2", len(children))
	}
}

func TestWalkTree(t *testing.T) {
	t.Parallel()

	// Given
	p := NewTSParser(domain.LanguageTypeScript)
	defer p.Close()
	source := []byte(`const x = 1;`)
	root, _ := p.Parse(source)

	t.Run("should visit all nodes", func(t *testing.T) {
		t.Parallel()

		// When
		var visited []string
		WalkTree(root, func(node *sitter.Node) bool {
			visited = append(visited, node.Type())
			return true
		})

		// Then
		if len(visited) == 0 {
			t.Error("WalkTree visited no nodes")
		}
		if visited[0] != "program" {
			t.Errorf("First visited = %q, want %q", visited[0], "program")
		}
	})

	t.Run("should stop descending when visitor returns false", func(t *testing.T) {
		t.Parallel()

		// When
		var visited []string
		WalkTree(root, func(node *sitter.Node) bool {
			visited = append(visited, node.Type())
			return node.Type() == "program"
		})

		// Then
		if len(visited) < 2 {
			t.Errorf("Should visit at least 2 nodes, got %d", len(visited))
		}
		if visited[0] != "program" {
			t.Errorf("First visited = %q, want %q", visited[0], "program")
		}
	})
}

func TestQuery(t *testing.T) {
	t.Parallel()

	t.Run("should find matches in TypeScript", func(t *testing.T) {
		t.Parallel()

		// Given
		p := NewTSParser(domain.LanguageTypeScript)
		defer p.Close()
		source := []byte(`
describe('test', () => {});
it('works', () => {});
`)
		root, _ := p.Parse(source)
		queryStr := `(call_expression function: (identifier) @func)`

		// When
		results, err := Query(root, source, domain.LanguageTypeScript, queryStr)

		// Then
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("len(results) = %d, want 2", len(results))
		}

		funcNode := results[0].Captures["func"]
		if funcNode == nil {
			t.Fatal("Expected 'func' capture")
		}
		if name := GetNodeText(funcNode, source); name != "describe" {
			t.Errorf("funcName = %q, want %q", name, "describe")
		}
	})

	t.Run("should find matches in JavaScript", func(t *testing.T) {
		t.Parallel()

		// Given
		p := NewTSParser(domain.LanguageJavaScript)
		defer p.Close()
		source := []byte(`function foo() {}`)
		root, _ := p.Parse(source)
		queryStr := `(function_declaration name: (identifier) @name)`

		// When
		results, err := Query(root, source, domain.LanguageJavaScript, queryStr)

		// Then
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("len(results) = %d, want 1", len(results))
		}
	})

	t.Run("should return error for invalid query", func(t *testing.T) {
		t.Parallel()

		// Given
		p := NewTSParser(domain.LanguageTypeScript)
		defer p.Close()
		source := []byte(`const x = 1;`)
		root, _ := p.Parse(source)

		// When
		_, err := Query(root, source, domain.LanguageTypeScript, "(invalid query syntax")

		// Then
		if err == nil {
			t.Error("Expected error for invalid query")
		}
	})
}
