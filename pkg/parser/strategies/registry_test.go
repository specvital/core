package strategies

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
)

type mockStrategy struct {
	name        string
	priority    int
	canHandleOk bool
}

func (m *mockStrategy) Name() string { return m.name }

func (m *mockStrategy) Priority() int {
	if m.priority == 0 {
		return DefaultPriority
	}
	return m.priority
}

func (m *mockStrategy) Languages() []domain.Language { return nil }

func (m *mockStrategy) CanHandle(string, []byte) bool {
	return m.canHandleOk
}

func (m *mockStrategy) Parse(_ context.Context, _ []byte, _ string) (*domain.TestFile, error) {
	return &domain.TestFile{Framework: m.name}, nil
}

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	// When
	r := NewRegistry()

	// Then
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if len(r.GetStrategies()) != 0 {
		t.Error("new registry should be empty")
	}
}

func TestRegistry_Register(t *testing.T) {
	t.Parallel()

	t.Run("should add strategies in order", func(t *testing.T) {
		t.Parallel()

		// Given
		r := NewRegistry()

		// When
		r.Register(&mockStrategy{name: "first"})
		r.Register(&mockStrategy{name: "second"})

		// Then
		strategies := r.GetStrategies()
		if len(strategies) != 2 {
			t.Fatalf("len(strategies) = %d, want 2", len(strategies))
		}
		if strategies[0].Name() != "first" {
			t.Errorf("strategies[0].Name() = %q, want %q", strategies[0].Name(), "first")
		}
	})

	t.Run("should sort by priority descending", func(t *testing.T) {
		t.Parallel()

		// Given
		r := NewRegistry()

		// When
		r.Register(&mockStrategy{name: "low", priority: 50})
		r.Register(&mockStrategy{name: "high", priority: 150})
		r.Register(&mockStrategy{name: "default"}) // priority 100

		// Then
		strategies := r.GetStrategies()
		if strategies[0].Name() != "high" {
			t.Errorf("First strategy = %q, want high", strategies[0].Name())
		}
		if strategies[1].Name() != "default" {
			t.Errorf("Second strategy = %q, want default", strategies[1].Name())
		}
		if strategies[2].Name() != "low" {
			t.Errorf("Third strategy = %q, want low", strategies[2].Name())
		}
	})
}

func TestRegistry_FindStrategy(t *testing.T) {
	t.Parallel()

	t.Run("should return matching strategy", func(t *testing.T) {
		t.Parallel()

		// Given
		r := NewRegistry()
		r.Register(&mockStrategy{name: "nomatch", canHandleOk: false})
		r.Register(&mockStrategy{name: "match", canHandleOk: true})

		// When
		found := r.FindStrategy("test.ts", nil)

		// Then
		if found == nil {
			t.Fatal("FindStrategy returned nil")
		}
		if found.Name() != "match" {
			t.Errorf("found.Name() = %q, want %q", found.Name(), "match")
		}
	})

	t.Run("should return nil when no strategy matches", func(t *testing.T) {
		t.Parallel()

		// Given
		r := NewRegistry()
		r.Register(&mockStrategy{name: "nomatch", canHandleOk: false})

		// When
		found := r.FindStrategy("test.ts", nil)

		// Then
		if found != nil {
			t.Errorf("FindStrategy returned %v, want nil", found)
		}
	})

	t.Run("should prefer higher priority strategy", func(t *testing.T) {
		t.Parallel()

		// Given
		r := NewRegistry()
		r.Register(&mockStrategy{name: "low", priority: 50, canHandleOk: true})
		r.Register(&mockStrategy{name: "high", priority: 150, canHandleOk: true})

		// When
		found := r.FindStrategy("test.ts", nil)

		// Then
		if found.Name() != "high" {
			t.Errorf("expected high priority, got %q", found.Name())
		}
	})
}

func TestRegistry_Clear(t *testing.T) {
	t.Parallel()

	// Given
	r := NewRegistry()
	r.Register(&mockStrategy{name: "test"})

	// When
	r.Clear()

	// Then
	if len(r.GetStrategies()) != 0 {
		t.Error("Clear did not remove strategies")
	}
}

func TestDefaultRegistry(t *testing.T) {
	// NOTE: This test modifies global state, so it cannot run in parallel.
	defaultRegistry.Clear()
	defer defaultRegistry.Clear()

	// When
	r := DefaultRegistry()

	// Then
	if r == nil {
		t.Fatal("DefaultRegistry returned nil")
	}
	if r != defaultRegistry {
		t.Error("DefaultRegistry should return singleton")
	}
}

func TestRegistry_FindByName(t *testing.T) {
	t.Parallel()

	t.Run("should return matching strategy by name", func(t *testing.T) {
		t.Parallel()

		r := NewRegistry()
		r.Register(&mockStrategy{name: "jest"})
		r.Register(&mockStrategy{name: "vitest"})

		found := r.FindByName("vitest")
		if found == nil {
			t.Fatal("FindByName returned nil")
		}
		if found.Name() != "vitest" {
			t.Errorf("found.Name() = %q, want %q", found.Name(), "vitest")
		}
	})

	t.Run("should return nil when no strategy matches", func(t *testing.T) {
		t.Parallel()

		r := NewRegistry()
		r.Register(&mockStrategy{name: "jest"})

		found := r.FindByName("nonexistent")
		if found != nil {
			t.Errorf("FindByName returned %v, want nil", found)
		}
	})

	t.Run("should return first match when duplicate names exist", func(t *testing.T) {
		t.Parallel()

		r := NewRegistry()
		r.Register(&mockStrategy{name: "jest", priority: 100})
		r.Register(&mockStrategy{name: "jest", priority: 50})

		found := r.FindByName("jest")
		if found == nil {
			t.Fatal("FindByName returned nil")
		}
		if found.Priority() != 100 {
			t.Errorf("expected first registered (priority 100), got priority %d", found.Priority())
		}
	})
}

func TestFindStrategyByName(t *testing.T) {
	defaultRegistry.Clear()
	defer defaultRegistry.Clear()

	Register(&mockStrategy{name: "playwright"})

	found := FindStrategyByName("playwright")
	if found == nil {
		t.Fatal("FindStrategyByName returned nil")
	}
	if found.Name() != "playwright" {
		t.Errorf("found.Name() = %q, want %q", found.Name(), "playwright")
	}

	notFound := FindStrategyByName("nonexistent")
	if notFound != nil {
		t.Errorf("FindStrategyByName returned %v, want nil", notFound)
	}
}

func TestGlobalFunctions(t *testing.T) {
	// NOTE: This test modifies global state, so it cannot run in parallel.

	t.Run("Register should add to default registry", func(t *testing.T) {
		defaultRegistry.Clear()
		defer defaultRegistry.Clear()

		// When
		Register(&mockStrategy{name: "global"})

		// Then
		strategies := GetStrategies()
		if len(strategies) != 1 {
			t.Fatalf("len(strategies) = %d, want 1", len(strategies))
		}
		if strategies[0].Name() != "global" {
			t.Errorf("Name = %q, want %q", strategies[0].Name(), "global")
		}
	})

	t.Run("GetStrategies should return from default registry", func(t *testing.T) {
		defaultRegistry.Clear()
		defer defaultRegistry.Clear()

		// Given
		Register(&mockStrategy{name: "s1"})
		Register(&mockStrategy{name: "s2"})

		// When
		strategies := GetStrategies()

		// Then
		if len(strategies) != 2 {
			t.Errorf("len(strategies) = %d, want 2", len(strategies))
		}
	})

	t.Run("FindStrategy should search default registry", func(t *testing.T) {
		defaultRegistry.Clear()
		defer defaultRegistry.Clear()

		// Given
		Register(&mockStrategy{name: "matcher", canHandleOk: true})

		// When
		found := FindStrategy("test.ts", nil)

		// Then
		if found == nil {
			t.Fatal("FindStrategy returned nil")
		}
		if found.Name() != "matcher" {
			t.Errorf("Name = %q, want %q", found.Name(), "matcher")
		}
	})

	t.Run("FindStrategy should return nil when no match", func(t *testing.T) {
		defaultRegistry.Clear()
		defer defaultRegistry.Clear()

		// Given
		Register(&mockStrategy{name: "no-match", canHandleOk: false})

		// When
		found := FindStrategy("test.ts", nil)

		// Then
		if found != nil {
			t.Errorf("Expected nil, got %v", found)
		}
	})
}
