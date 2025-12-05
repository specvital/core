// Package strategies provides the strategy pattern implementation for test file parsing.
// Each test framework (Jest, Vitest, Playwright, Go testing) has its own strategy.
package strategies

import (
	"context"
	"sort"
	"sync"

	"github.com/specvital/core/pkg/domain"
)

// DefaultPriority is the default priority for strategies.
// Higher priority strategies are checked first.
const DefaultPriority = 100

var defaultRegistry = &Registry{}

// Strategy defines the interface for test framework-specific parsers.
type Strategy interface {
	// Name returns the strategy identifier (e.g., "jest", "vitest").
	Name() string
	// Priority returns the strategy priority (higher = checked first).
	Priority() int
	// Languages returns the languages this strategy supports.
	Languages() []domain.Language
	// CanHandle returns true if this strategy can parse the given file.
	CanHandle(filename string, content []byte) bool
	// Parse parses the source code and extracts test definitions.
	Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error)
}

// Registry manages registered strategies.
type Registry struct {
	mu         sync.RWMutex
	strategies []Strategy
}

// NewRegistry creates a new empty strategy registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// DefaultRegistry returns the global default registry.
func DefaultRegistry() *Registry {
	return defaultRegistry
}

// Register adds a strategy to the default registry.
func Register(s Strategy) {
	defaultRegistry.Register(s)
}

// GetStrategies returns all registered strategies from the default registry.
func GetStrategies() []Strategy {
	return defaultRegistry.GetStrategies()
}

// FindStrategy returns the first matching strategy for the given file.
func FindStrategy(filename string, content []byte) Strategy {
	return defaultRegistry.FindStrategy(filename, content)
}

// Register adds a strategy to the registry.
func (r *Registry) Register(s Strategy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.strategies = append(r.strategies, s)
	r.sortByPriority()
}

func (r *Registry) sortByPriority() {
	sort.Slice(r.strategies, func(i, j int) bool {
		return r.strategies[i].Priority() > r.strategies[j].Priority()
	})
}

// GetStrategies returns a copy of all registered strategies.
func (r *Registry) GetStrategies() []Strategy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Strategy, len(r.strategies))
	copy(result, r.strategies)
	return result
}

// FindStrategy returns the first strategy that can handle the given file.
func (r *Registry) FindStrategy(filename string, content []byte) Strategy {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, s := range r.strategies {
		if s.CanHandle(filename, content) {
			return s
		}
	}
	return nil
}

// Clear removes all registered strategies.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.strategies = nil
}

// FindByName returns the strategy with the given name.
func (r *Registry) FindByName(name string) Strategy {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, s := range r.strategies {
		if s.Name() == name {
			return s
		}
	}
	return nil
}

// FindStrategyByName returns the strategy with the given name from the default registry.
func FindStrategyByName(name string) Strategy {
	return defaultRegistry.FindByName(name)
}
