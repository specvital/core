// Package strategies provides test framework parsing strategies.
package strategies

import (
	"sort"
	"sync"

	"github.com/specvital/core/domain"
)

// DefaultPriority is the default priority for strategies.
const DefaultPriority = 100

// Strategy defines the interface for test framework parsers.
type Strategy interface {
	// Name returns the framework name (e.g., "jest", "vitest").
	Name() string

	// Priority returns the strategy priority. Higher values take precedence.
	// Default priority is 100. Use higher values for more specific strategies.
	Priority() int

	// Languages returns supported languages.
	Languages() []domain.Language

	// CanHandle checks if this strategy can parse the given file.
	// filename is the relative path, content is the file content.
	CanHandle(filename string, content []byte) bool

	// Parse extracts test information from source code.
	Parse(source []byte, filename string) (*domain.TestFile, error)
}

// Registry manages registered strategies.
type Registry struct {
	mu         sync.RWMutex
	strategies []Strategy
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

var defaultRegistry = &Registry{}

// DefaultRegistry returns the default global registry.
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

// FindStrategy finds a matching strategy for the given file.
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

// sortByPriority sorts strategies by priority (higher first).
func (r *Registry) sortByPriority() {
	sort.Slice(r.strategies, func(i, j int) bool {
		return r.strategies[i].Priority() > r.strategies[j].Priority()
	})
}

// GetStrategies returns all registered strategies.
func (r *Registry) GetStrategies() []Strategy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Strategy, len(r.strategies))
	copy(result, r.strategies)
	return result
}

// FindStrategy finds a matching strategy for the given file.
// Strategies are checked in priority order (highest first).
// Returns nil if no strategy matches.
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

// Clear removes all strategies from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.strategies = nil
}
