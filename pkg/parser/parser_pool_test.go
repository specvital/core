package parser

import (
	"context"
	"sync"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/tspool"
)

func TestParserPool_RaceFree(t *testing.T) {
	const goroutines = 100
	const iterations = 10

	source := []byte(`
package main

import "testing"

func TestExample(t *testing.T) {
	t.Run("subtest", func(t *testing.T) {
		// test code
	})
}
`)

	var wg sync.WaitGroup
	ctx := context.Background()

	// Test concurrent access to parser pool
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				tree, err := tspool.Parse(ctx, domain.LanguageGo, source)
				if err != nil {
					t.Errorf("goroutine %d iteration %d: parse failed: %v", id, j, err)
					return
				}

				if tree == nil {
					t.Errorf("goroutine %d iteration %d: tree is nil", id, j)
					return
				}

				tree.Close()
			}
		}(i)
	}

	wg.Wait()
}

func TestParserPool_MultipleLanguages(t *testing.T) {
	const goroutines = 50

	sources := map[domain.Language][]byte{
		domain.LanguageGo: []byte(`
package main
func TestExample(t *testing.T) {}
`),
		domain.LanguageJavaScript: []byte(`
describe('test', () => {
	it('works', () => {});
});
`),
		domain.LanguageTypeScript: []byte(`
describe('test', () => {
	it('works', () => {});
});
`),
	}

	var wg sync.WaitGroup
	ctx := context.Background()

	// Test that different language pools work independently
	for lang, source := range sources {
		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(l domain.Language, src []byte) {
				defer wg.Done()

				tree, err := tspool.Parse(ctx, l, src)
				if err != nil {
					t.Errorf("language %v: parse failed: %v", l, err)
					return
				}
				defer tree.Close()

				if tree == nil {
					t.Errorf("language %v: tree is nil", l)
				}
			}(lang, source)
		}
	}

	wg.Wait()
}

func TestQueryCache_RaceFree(t *testing.T) {
	const goroutines = 100

	source := []byte(`
package main

import "testing"

func TestExample(t *testing.T) {
	t.Run("subtest", func(t *testing.T) {
		// test code
	})
}
`)

	queryStr := `
(function_declaration
  name: (identifier) @name
  parameters: (parameter_list
    (parameter_declaration
      type: (pointer_type
        (qualified_type) @param_type))))
`

	ctx := context.Background()

	var wg sync.WaitGroup

	// Test concurrent query compilation and caching
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			tree, err := tspool.Parse(ctx, domain.LanguageGo, source)
			if err != nil {
				t.Errorf("goroutine %d: parse failed: %v", id, err)
				return
			}
			defer tree.Close()

			results, err := QueryWithCache(tree.RootNode(), source, domain.LanguageGo, queryStr)
			if err != nil {
				t.Errorf("goroutine %d: query failed: %v", id, err)
				return
			}

			if len(results) == 0 {
				t.Errorf("goroutine %d: expected query results", id)
			}
		}(i)
	}

	wg.Wait()
}

func TestQueryCache_SameQueryReused(t *testing.T) {
	defer ClearQueryCache()

	source := []byte(`package main`)
	queryStr := `(package_clause)`

	ctx := context.Background()
	lang := domain.LanguageGo

	// First call - should compile query
	tree1, err := tspool.Parse(ctx, lang, source)
	if err != nil {
		t.Fatalf("parse 1 failed: %v", err)
	}
	defer tree1.Close()

	results1, err := QueryWithCache(tree1.RootNode(), source, lang, queryStr)
	if err != nil {
		t.Fatalf("query 1 failed: %v", err)
	}

	// Second call - should use cached query
	tree2, err := tspool.Parse(ctx, lang, source)
	if err != nil {
		t.Fatalf("parse 2 failed: %v", err)
	}
	defer tree2.Close()

	results2, err := QueryWithCache(tree2.RootNode(), source, lang, queryStr)
	if err != nil {
		t.Fatalf("query 2 failed: %v", err)
	}

	// Results should be identical (cached query works)
	if len(results1) != len(results2) {
		t.Errorf("result count mismatch: got %d and %d", len(results1), len(results2))
	}
}

func TestGetParser_ReturnsValidParser(t *testing.T) {
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
			parser := tspool.Get(tt.lang)
			if parser == nil {
				t.Fatal("parser is nil")
			}
			defer parser.Close()

			// Verify parser can be used
			ctx := context.Background()
			source := []byte("package main")
			if tt.lang == domain.LanguageJavaScript || tt.lang == domain.LanguageTypeScript {
				source = []byte("console.log('test');")
			}

			tree, err := parser.ParseCtx(ctx, nil, source)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			defer tree.Close()
		})
	}
}

func TestClearQueryCache(t *testing.T) {
	source := []byte(`package main`)
	queryStr := `(package_clause)`
	lang := domain.LanguageGo
	ctx := context.Background()

	tree, err := tspool.Parse(ctx, lang, source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	defer tree.Close()

	// Add query to cache via QueryWithCache
	_, err = QueryWithCache(tree.RootNode(), source, lang, queryStr)
	if err != nil {
		t.Fatalf("failed to cache query: %v", err)
	}

	// Clear cache
	ClearQueryCache()

	// Query should be recompiled (no error expected, just testing it works)
	_, err = QueryWithCache(tree.RootNode(), source, lang, queryStr)
	if err != nil {
		t.Fatalf("failed to recompile query after clear: %v", err)
	}
}

// Benchmark parser pool vs direct creation
func BenchmarkParser_Direct(b *testing.B) {
	source := []byte(`
package main

import "testing"

func TestExample(t *testing.T) {
	t.Run("subtest", func(t *testing.T) {
		// test code
	})
}
`)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := NewTSParser(domain.LanguageGo)
		tree, err := p.Parse(ctx, source)
		if err != nil {
			b.Fatal(err)
		}
		tree.Close()
	}
}

func BenchmarkParser_Pooled(b *testing.B) {
	source := []byte(`
package main

import "testing"

func TestExample(t *testing.T) {
	t.Run("subtest", func(t *testing.T) {
		// test code
	})
}
`)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree, err := tspool.Parse(ctx, domain.LanguageGo, source)
		if err != nil {
			b.Fatal(err)
		}
		tree.Close()
	}
}

func BenchmarkParser_PooledParallel(b *testing.B) {
	source := []byte(`
package main

import "testing"

func TestExample(t *testing.T) {
	t.Run("subtest", func(t *testing.T) {
		// test code
	})
}
`)
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tree, err := tspool.Parse(ctx, domain.LanguageGo, source)
			if err != nil {
				b.Fatal(err)
			}
			tree.Close()
		}
	})
}

func BenchmarkQuery_Direct(b *testing.B) {
	source := []byte(`
package main

import "testing"

func TestExample(t *testing.T) {
	t.Run("subtest", func(t *testing.T) {})
}
`)
	queryStr := `
(function_declaration
  name: (identifier) @name
  parameters: (parameter_list))
`
	ctx := context.Background()
	lang := domain.LanguageGo

	tree, err := tspool.Parse(ctx, lang, source)
	if err != nil {
		b.Fatal(err)
	}
	defer tree.Close()
	root := tree.RootNode()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Query(root, source, lang, queryStr)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQuery_Cached(b *testing.B) {
	source := []byte(`
package main

import "testing"

func TestExample(t *testing.T) {
	t.Run("subtest", func(t *testing.T) {})
}
`)
	queryStr := `
(function_declaration
  name: (identifier) @name
  parameters: (parameter_list))
`
	ctx := context.Background()
	lang := domain.LanguageGo

	tree, err := tspool.Parse(ctx, lang, source)
	if err != nil {
		b.Fatal(err)
	}
	defer tree.Close()
	root := tree.RootNode()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := QueryWithCache(root, source, lang, queryStr)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQuery_CachedParallel(b *testing.B) {
	source := []byte(`
package main

import "testing"

func TestExample(t *testing.T) {
	t.Run("subtest", func(t *testing.T) {})
}
`)
	queryStr := `
(function_declaration
  name: (identifier) @name
  parameters: (parameter_list))
`
	ctx := context.Background()
	lang := domain.LanguageGo

	tree, err := tspool.Parse(ctx, lang, source)
	if err != nil {
		b.Fatal(err)
	}
	defer tree.Close()
	root := tree.RootNode()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := QueryWithCache(root, source, lang, queryStr)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
