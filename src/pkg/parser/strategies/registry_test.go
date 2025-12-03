package strategies

import (
	"testing"

	"github.com/specvital/core/domain"
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

func (m *mockStrategy) CanHandle(filename string, content []byte) bool {
	return m.canHandleOk
}

func (m *mockStrategy) Parse(source []byte, filename string) (*domain.TestFile, error) {
	return &domain.TestFile{Framework: m.name}, nil
}

func TestRegistry_Register(t *testing.T) {
	r := &Registry{}

	s1 := &mockStrategy{name: "mock1"}
	s2 := &mockStrategy{name: "mock2"}

	r.Register(s1)
	r.Register(s2)

	strategies := r.GetStrategies()
	if len(strategies) != 2 {
		t.Errorf("len(strategies) = %d, want 2", len(strategies))
	}

	if strategies[0].Name() != "mock1" {
		t.Errorf("strategies[0].Name() = %q, want %q", strategies[0].Name(), "mock1")
	}

	if strategies[1].Name() != "mock2" {
		t.Errorf("strategies[1].Name() = %q, want %q", strategies[1].Name(), "mock2")
	}
}

func TestRegistry_FindStrategy(t *testing.T) {
	r := &Registry{}

	s1 := &mockStrategy{name: "nomatch", canHandleOk: false}
	s2 := &mockStrategy{name: "match", canHandleOk: true}

	r.Register(s1)
	r.Register(s2)

	found := r.FindStrategy("test.ts", nil)
	if found == nil {
		t.Fatal("FindStrategy returned nil")
	}

	if found.Name() != "match" {
		t.Errorf("found.Name() = %q, want %q", found.Name(), "match")
	}
}

func TestRegistry_FindStrategy_NotFound(t *testing.T) {
	r := &Registry{}

	s1 := &mockStrategy{name: "nomatch", canHandleOk: false}
	r.Register(s1)

	found := r.FindStrategy("test.ts", nil)
	if found != nil {
		t.Errorf("FindStrategy returned %v, want nil", found)
	}
}

func TestRegistry_Clear(t *testing.T) {
	r := &Registry{}

	r.Register(&mockStrategy{name: "test"})
	r.Clear()

	if len(r.GetStrategies()) != 0 {
		t.Error("Clear did not remove strategies")
	}
}

func TestRegistry_FindStrategy_Priority(t *testing.T) {
	r := NewRegistry()

	lowPriority := &mockStrategy{name: "low", priority: 50, canHandleOk: true}
	highPriority := &mockStrategy{name: "high", priority: 150, canHandleOk: true}
	defaultPriority := &mockStrategy{name: "default", canHandleOk: true}

	r.Register(lowPriority)
	r.Register(defaultPriority)
	r.Register(highPriority)

	found := r.FindStrategy("test.ts", nil)
	if found == nil {
		t.Fatal("FindStrategy returned nil")
	}

	if found.Name() != "high" {
		t.Errorf("expected high priority strategy, got %q", found.Name())
	}
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}

	if len(r.GetStrategies()) != 0 {
		t.Error("new registry should be empty")
	}
}
