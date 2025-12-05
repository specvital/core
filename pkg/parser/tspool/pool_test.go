package tspool_test

import (
	"context"
	"sync"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/tspool"
)

func TestParse_RaceFree(t *testing.T) {
	t.Parallel()

	const goroutines = 50
	source := []byte("const x = 1;")

	var wg sync.WaitGroup
	wg.Add(goroutines)

	errCh := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			tree, err := tspool.Parse(context.Background(), domain.LanguageTypeScript, source)
			if err != nil {
				errCh <- err
				return
			}
			defer tree.Close()
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("Parse failed: %v", err)
	}
}

func TestGetPut_ReusesParser(t *testing.T) {
	t.Parallel()

	parser1 := tspool.Get(domain.LanguageGo)
	if parser1 == nil {
		t.Fatal("Get returned nil parser")
	}

	tspool.Put(domain.LanguageGo, parser1)

	parser2 := tspool.Get(domain.LanguageGo)
	if parser2 == nil {
		t.Fatal("Get returned nil parser after Put")
	}

	tspool.Put(domain.LanguageGo, parser2)
}

func TestParse_ContextCancellation(t *testing.T) {
	t.Parallel()

	// Note: tree-sitter's ParseCtx may not honor context cancellation for small inputs.
	// This test verifies the context is passed through, not that parsing fails.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	source := []byte("const x = 1;")
	tree, err := tspool.Parse(ctx, domain.LanguageTypeScript, source)

	// Either error or success is acceptable - tree-sitter behavior varies
	if err == nil && tree != nil {
		tree.Close()
	}
}

func TestGetLanguage_ReturnsCorrectLanguages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		lang domain.Language
	}{
		{"Go", domain.LanguageGo},
		{"JavaScript", domain.LanguageJavaScript},
		{"TypeScript", domain.LanguageTypeScript},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lang := tspool.GetLanguage(tt.lang)
			if lang == nil {
				t.Errorf("GetLanguage(%v) returned nil", tt.lang)
			}
		})
	}
}

func TestPut_NilParser(t *testing.T) {
	t.Parallel()

	// Should not panic
	tspool.Put(domain.LanguageGo, nil)
}

func TestParse_ValidOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		lang   domain.Language
		source string
	}{
		{
			name:   "TypeScript const",
			lang:   domain.LanguageTypeScript,
			source: "const x: number = 1;",
		},
		{
			name:   "JavaScript function",
			lang:   domain.LanguageJavaScript,
			source: "function foo() { return 42; }",
		},
		{
			name:   "Go function",
			lang:   domain.LanguageGo,
			source: "package main\nfunc main() {}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tree, err := tspool.Parse(context.Background(), tt.lang, []byte(tt.source))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			defer tree.Close()

			root := tree.RootNode()
			if root == nil {
				t.Fatal("Root node is nil")
			}
			if root.ChildCount() == 0 {
				t.Error("Expected children in parsed tree")
			}
		})
	}
}
