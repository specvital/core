package detection

import (
	"context"
	"path/filepath"
	"slices"

	"github.com/specvital/core/pkg/parser/detection/matchers"
)

// ProjectContextBuilder constructs ProjectContext from file lists and config contents.
type ProjectContextBuilder struct {
	ctx      *ProjectContext
	registry *matchers.Registry
}

func NewProjectContextBuilder(registry *matchers.Registry) *ProjectContextBuilder {
	return &ProjectContextBuilder{
		ctx:      NewProjectContext(),
		registry: registry,
	}
}

// AddConfigFiles filters and registers config file paths from a file list.
func (b *ProjectContextBuilder) AddConfigFiles(filePaths []string) *ProjectContextBuilder {
	configPatterns := b.collectConfigPatterns()

	for _, path := range filePaths {
		baseName := filepath.Base(path)
		if configPatterns[baseName] {
			b.ctx.AddConfigFile(path)
		}
	}

	return b
}

// ParseConfigContent parses config content using the appropriate matcher.
// Matchers are sorted by priority (highest first) to ensure consistent behavior
// when multiple matchers could handle the same config pattern.
func (b *ProjectContextBuilder) ParseConfigContent(ctx context.Context, configPath string, content []byte) *ProjectContextBuilder {
	baseName := filepath.Base(configPath)

	matcherList := sortedByPriority(b.registry.All())

	for _, matcher := range matcherList {
		patterns := matcher.ConfigPatterns()
		if len(patterns) == 0 {
			continue
		}
		if slices.Contains(patterns, baseName) {
			if info := matcher.ParseConfig(ctx, content); info != nil {
				b.ctx.SetConfigContent(configPath, info)
			}
			return b
		}
	}

	return b
}

func (b *ProjectContextBuilder) Build() *ProjectContext {
	return b.ctx
}

func (b *ProjectContextBuilder) collectConfigPatterns() map[string]bool {
	patterns := make(map[string]bool)

	for _, matcher := range b.registry.All() {
		for _, pattern := range matcher.ConfigPatterns() {
			patterns[pattern] = true
		}
	}

	return patterns
}
