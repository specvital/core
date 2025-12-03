package strategies

import (
	"sort"
	"sync"

	"github.com/specvital/core/domain"
)

const DefaultPriority = 100

type Strategy interface {
	Name() string
	Priority() int
	Languages() []domain.Language
	CanHandle(filename string, content []byte) bool
	Parse(source []byte, filename string) (*domain.TestFile, error)
}

type Registry struct {
	mu         sync.RWMutex
	strategies []Strategy
}

func NewRegistry() *Registry {
	return &Registry{}
}

var defaultRegistry = &Registry{}

func DefaultRegistry() *Registry {
	return defaultRegistry
}

func Register(s Strategy) {
	defaultRegistry.Register(s)
}

func GetStrategies() []Strategy {
	return defaultRegistry.GetStrategies()
}

func FindStrategy(filename string, content []byte) Strategy {
	return defaultRegistry.FindStrategy(filename, content)
}

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

func (r *Registry) GetStrategies() []Strategy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Strategy, len(r.strategies))
	copy(result, r.strategies)
	return result
}

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

func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.strategies = nil
}
