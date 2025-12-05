package matchers

import (
	"sync"

	"github.com/specvital/core/pkg/domain"
)

var defaultRegistry = &Registry{}

type Registry struct {
	mu       sync.RWMutex
	matchers []Matcher
}

func NewRegistry() *Registry {
	return &Registry{}
}

func DefaultRegistry() *Registry {
	return defaultRegistry
}

func Register(m Matcher) {
	defaultRegistry.Register(m)
}

func (r *Registry) Register(m Matcher) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.matchers = append(r.matchers, m)
}

func (r *Registry) All() []Matcher {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Matcher, len(r.matchers))
	copy(result, r.matchers)
	return result
}

func (r *Registry) FindByName(name string) (Matcher, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.matchers {
		if m.Name() == name {
			return m, true
		}
	}
	return nil, false
}

func (r *Registry) FindByImport(importPath string) (Matcher, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.matchers {
		if m.MatchImport(importPath) {
			return m, true
		}
	}
	return nil, false
}

func (r *Registry) FindByLanguage(lang domain.Language) []Matcher {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []Matcher
	for _, m := range r.matchers {
		for _, l := range m.Languages() {
			if l == lang {
				result = append(result, m)
				break
			}
		}
	}
	return result
}

func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.matchers = nil
}
