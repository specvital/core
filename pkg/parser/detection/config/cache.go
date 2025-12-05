package config

import (
	"sort"
	"strings"
	"sync"
)

type Cache struct {
	mu    sync.RWMutex
	items map[string]string
}

func NewCache() *Cache {
	return &Cache{
		items: make(map[string]string),
	}
}

func (c *Cache) Get(dir string, patterns []string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := makeCacheKey(dir, patterns)
	val, ok := c.items[key]
	return val, ok
}

func (c *Cache) Set(dir string, patterns []string, configPath string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := makeCacheKey(dir, patterns)
	c.items[key] = configPath
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]string)
}

func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

func makeCacheKey(dir string, patterns []string) string {
	sorted := make([]string, len(patterns))
	copy(sorted, patterns)
	sort.Strings(sorted)
	return dir + "|" + strings.Join(sorted, "|")
}
