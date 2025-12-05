package detection

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/detection/config"
	"github.com/specvital/core/pkg/parser/detection/matchers"
)

var langExtMap = map[string]domain.Language{
	".cts": domain.LanguageTypeScript,
	".cjs": domain.LanguageJavaScript,
	".go":  domain.LanguageGo,
	".js":  domain.LanguageJavaScript,
	".jsx": domain.LanguageJavaScript,
	".mjs": domain.LanguageJavaScript,
	".mts": domain.LanguageTypeScript,
	".ts":  domain.LanguageTypeScript,
	".tsx": domain.LanguageTypeScript,
}

type Detector struct {
	matcherRegistry *matchers.Registry
	scopeResolver   *config.Resolver
}

func NewDetector(matcherRegistry *matchers.Registry, scopeResolver *config.Resolver) *Detector {
	return &Detector{
		matcherRegistry: matcherRegistry,
		scopeResolver:   scopeResolver,
	}
}

// Detect performs hierarchical framework detection.
// Level 1: Import statements → Level 2: Scope config files → Level 3: Unknown
func (d *Detector) Detect(ctx context.Context, filePath string, content []byte) Result {
	if result, ok := d.detectFromImports(ctx, filePath, content); ok {
		return result
	}
	if result, ok := d.detectFromScopeConfig(filePath); ok {
		return result
	}
	return Unknown()
}

func (d *Detector) detectFromImports(ctx context.Context, filePath string, content []byte) (Result, bool) {
	lang := detectLanguage(filePath)
	if lang == "" {
		return Result{}, false
	}

	compatibleMatchers := d.matcherRegistry.FindByLanguage(lang)
	if len(compatibleMatchers) == 0 {
		return Result{}, false
	}

	// Extract imports once using the first compatible matcher
	imports := compatibleMatchers[0].ExtractImports(ctx, content)
	if len(imports) == 0 {
		return Result{}, false
	}

	if matcher := findMatchingMatcher(compatibleMatchers, imports); matcher != nil {
		return FromImport(matcher.Name()), true
	}
	return Result{}, false
}

func (d *Detector) detectFromScopeConfig(filePath string) (Result, bool) {
	if d.scopeResolver == nil {
		return Result{}, false
	}

	for _, matcher := range d.matcherRegistry.All() {
		patterns := matcher.ConfigPatterns()
		if len(patterns) == 0 {
			continue
		}
		if configPath, found := d.scopeResolver.ResolveConfig(filePath, patterns); found {
			return FromScopeConfig(matcher.Name(), configPath), true
		}
	}
	return Result{}, false
}

func findMatchingMatcher(matcherList []matchers.Matcher, imports []string) matchers.Matcher {
	for _, matcher := range matcherList {
		for _, importPath := range imports {
			if matcher.MatchImport(importPath) {
				return matcher
			}
		}
	}
	return nil
}

func detectLanguage(filePath string) domain.Language {
	ext := strings.ToLower(filepath.Ext(filePath))
	return langExtMap[ext]
}
