package detection

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/detection/config"
	"github.com/specvital/core/pkg/parser/detection/extraction"
	"github.com/specvital/core/pkg/parser/detection/matchers"
)

type mockMatcher struct {
	configPatterns []string
	extractFunc    func(context.Context, []byte) []string
	languages      []domain.Language
	matchImports   []string
	name           string
}

func (m *mockMatcher) Name() string                 { return m.name }
func (m *mockMatcher) Languages() []domain.Language { return m.languages }
func (m *mockMatcher) ConfigPatterns() []string     { return m.configPatterns }
func (m *mockMatcher) MatchImport(importPath string) bool {
	return slices.Contains(m.matchImports, importPath)
}
func (m *mockMatcher) ExtractImports(ctx context.Context, content []byte) []string {
	if m.extractFunc != nil {
		return m.extractFunc(ctx, content)
	}
	return nil
}

func TestDetector_Detect_Level1_Import(t *testing.T) {
	t.Parallel()

	registry := matchers.NewRegistry()
	registry.Register(&mockMatcher{
		name:         "vitest",
		languages:    []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		matchImports: []string{"vitest"},
		extractFunc:  extraction.ExtractJSImports,
	})
	registry.Register(&mockMatcher{
		name:         "playwright",
		languages:    []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		matchImports: []string{"@playwright/test"},
		extractFunc:  extraction.ExtractJSImports,
	})
	registry.Register(&mockMatcher{
		name:         "jest",
		languages:    []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		matchImports: []string{"@jest/globals"},
		extractFunc:  extraction.ExtractJSImports,
	})

	detector := NewDetector(registry, nil)

	tests := []struct {
		name          string
		content       string
		wantFramework string
		wantSource    Source
	}{
		{
			name:          "vitest import",
			content:       `import { describe, it } from 'vitest';`,
			wantFramework: "vitest",
			wantSource:    SourceImport,
		},
		{
			name:          "playwright import",
			content:       `import { test, expect } from '@playwright/test';`,
			wantFramework: "playwright",
			wantSource:    SourceImport,
		},
		{
			name:          "jest globals import",
			content:       `import { jest } from '@jest/globals';`,
			wantFramework: "jest",
			wantSource:    SourceImport,
		},
		{
			name:          "require syntax",
			content:       `const { test } = require('@playwright/test');`,
			wantFramework: "playwright",
			wantSource:    SourceImport,
		},
		{
			name:          "no framework import",
			content:       `describe('test', () => { it('works', () => {}) });`,
			wantFramework: "unknown",
			wantSource:    SourceUnknown,
		},
		{
			name:          "unrelated import",
			content:       `import lodash from 'lodash';`,
			wantFramework: "unknown",
			wantSource:    SourceUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := detector.Detect(context.Background(), "test.ts", []byte(tt.content))

			if result.Framework != tt.wantFramework {
				t.Errorf("Framework = %q, want %q", result.Framework, tt.wantFramework)
			}
			if result.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", result.Source, tt.wantSource)
			}
		})
	}
}

func TestDetector_Detect_Level2_ScopeConfig(t *testing.T) {
	// Create temp directory structure
	tempDir := t.TempDir()

	// Create monorepo structure:
	// tempDir/
	// ├── apps/
	// │   ├── web/
	// │   │   ├── jest.config.js
	// │   │   └── __tests__/user.test.ts
	// │   └── api/
	// │       ├── vitest.config.ts
	// │       └── __tests__/handler.test.ts
	// └── e2e/
	//     ├── playwright.config.ts
	//     └── login.spec.ts

	dirs := []string{
		filepath.Join(tempDir, "apps", "web", "__tests__"),
		filepath.Join(tempDir, "apps", "api", "__tests__"),
		filepath.Join(tempDir, "e2e"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	configs := map[string]string{
		filepath.Join(tempDir, "apps", "web", "jest.config.js"):   "",
		filepath.Join(tempDir, "apps", "api", "vitest.config.ts"): "",
		filepath.Join(tempDir, "e2e", "playwright.config.ts"):     "",
	}
	for path, content := range configs {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Setup registry and detector
	registry := matchers.NewRegistry()
	registry.Register(&mockMatcher{
		name:           "jest",
		configPatterns: []string{"jest.config.js", "jest.config.ts"},
	})
	registry.Register(&mockMatcher{
		name:           "vitest",
		configPatterns: []string{"vitest.config.js", "vitest.config.ts"},
	})
	registry.Register(&mockMatcher{
		name:           "playwright",
		configPatterns: []string{"playwright.config.js", "playwright.config.ts"},
	})

	cache := config.NewCache()
	resolver := config.NewResolver(cache, 10)
	detector := NewDetector(registry, resolver)

	tests := []struct {
		name          string
		filePath      string
		wantFramework string
		wantSource    Source
	}{
		{
			name:          "web test uses jest",
			filePath:      filepath.Join(tempDir, "apps", "web", "__tests__", "user.test.ts"),
			wantFramework: "jest",
			wantSource:    SourceScopeConfig,
		},
		{
			name:          "api test uses vitest",
			filePath:      filepath.Join(tempDir, "apps", "api", "__tests__", "handler.test.ts"),
			wantFramework: "vitest",
			wantSource:    SourceScopeConfig,
		},
		{
			name:          "e2e test uses playwright",
			filePath:      filepath.Join(tempDir, "e2e", "login.spec.ts"),
			wantFramework: "playwright",
			wantSource:    SourceScopeConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// No import in content, should fallback to scope config
			result := detector.Detect(context.Background(), tt.filePath, []byte(`describe('test', () => {})`))

			if result.Framework != tt.wantFramework {
				t.Errorf("Framework = %q, want %q", result.Framework, tt.wantFramework)
			}
			if result.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", result.Source, tt.wantSource)
			}
		})
	}
}

func TestDetector_Detect_ImportTakesPrecedence(t *testing.T) {
	// Even if config file exists, import should take precedence

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "jest.config.js")
	if err := os.WriteFile(configPath, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	registry := matchers.NewRegistry()
	registry.Register(&mockMatcher{
		name:           "jest",
		languages:      []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		matchImports:   []string{"@jest/globals"},
		configPatterns: []string{"jest.config.js"},
		extractFunc:    extraction.ExtractJSImports,
	})
	registry.Register(&mockMatcher{
		name:           "vitest",
		languages:      []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		matchImports:   []string{"vitest"},
		configPatterns: []string{"vitest.config.ts"},
		extractFunc:    extraction.ExtractJSImports,
	})

	cache := config.NewCache()
	resolver := config.NewResolver(cache, 10)
	detector := NewDetector(registry, resolver)

	// File is in jest config directory but has vitest import
	content := `import { describe } from 'vitest';`
	result := detector.Detect(context.Background(), filepath.Join(tempDir, "test.ts"), []byte(content))

	if result.Framework != "vitest" {
		t.Errorf("Framework = %q, want vitest (import should take precedence)", result.Framework)
	}
	if result.Source != SourceImport {
		t.Errorf("Source = %q, want %q", result.Source, SourceImport)
	}
}

func TestResult_IsUnknown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result Result
		want   bool
	}{
		{"unknown framework", Result{Framework: "unknown"}, true},
		{"empty framework", Result{Framework: ""}, true},
		{"zero confidence", Result{Framework: "jest", Confidence: ConfidenceUnknown}, true},
		{"known framework", Result{Framework: "jest", Confidence: ConfidenceMedium}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.result.IsUnknown(); got != tt.want {
				t.Errorf("IsUnknown() = %v, want %v", got, tt.want)
			}
		})
	}
}
