package detection

import (
	"context"
	"regexp"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
	"github.com/specvital/core/pkg/parser/framework/matchers"
)

// TestDetector_Detect_ScopeBasedDetection tests highest confidence detection via config scope.
func TestDetector_Detect_ScopeBasedDetection(t *testing.T) {
	registry := framework.NewRegistry()
	registry.Register(&framework.Definition{
		Name:      "jest",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("@jest/globals", "@jest/globals/"),
		},
	})

	detector := NewDetector(registry)

	// Setup project scope with Jest config
	projectScope := framework.NewProjectScope()
	configScope := &framework.ConfigScope{
		ConfigPath:  "/project/jest.config.js",
		BaseDir:     "/project",
		Framework:   "jest",
		GlobalsMode: true,
	}
	projectScope.AddConfig("/project/jest.config.js", configScope)
	detector.SetProjectScope(projectScope)

	// Test file within scope
	content := []byte(`
describe('test suite', () => {
  it('should work', () => {
    expect(true).toBe(true);
  });
});
`)

	result := detector.Detect(context.Background(), "/project/src/test.spec.ts", content)

	if result.Framework != "jest" {
		t.Errorf("expected framework 'jest', got '%s'", result.Framework)
	}

	// Should have high confidence from scope + globals mode
	if result.Confidence < 80 {
		t.Errorf("expected confidence >= 80, got %d", result.Confidence)
	}

	if !result.IsDefinite() {
		t.Errorf("expected definite confidence level, got %s", result.ConfidenceLevel())
	}

	// Verify evidence
	hasScope := false
	hasGlobals := false
	for _, ev := range result.Evidence {
		if ev.Source == "config-scope" {
			hasScope = true
		}
		if ev.Source == "globals-mode" {
			hasGlobals = true
		}
	}

	if !hasScope {
		t.Error("expected config-scope evidence")
	}
	if !hasGlobals {
		t.Error("expected globals-mode evidence")
	}
}

// TestDetector_Detect_ImportBasedDetection tests high confidence detection via imports.
func TestDetector_Detect_ImportBasedDetection(t *testing.T) {
	registry := framework.NewRegistry()
	registry.Register(&framework.Definition{
		Name:      "vitest",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("vitest", "vitest/"),
		},
	})

	detector := NewDetector(registry)

	content := []byte(`
import { describe, it, expect } from 'vitest';

describe('test suite', () => {
  it('should work', () => {
    expect(true).toBe(true);
  });
});
`)

	result := detector.Detect(context.Background(), "/project/test.spec.ts", content)

	if result.Framework != "vitest" {
		t.Errorf("expected framework 'vitest', got '%s'", result.Framework)
	}

	// Should have high confidence from import (60 points = moderate level)
	if result.Confidence < 60 {
		t.Errorf("expected confidence >= 60, got %d", result.Confidence)
	}

	if !result.IsModerate() && !result.IsDefinite() {
		t.Errorf("expected moderate or definite confidence level, got %s", result.ConfidenceLevel())
	}

	// Verify evidence
	hasImport := false
	for _, ev := range result.Evidence {
		if ev.Source == "import" && !ev.Negative {
			hasImport = true
		}
	}

	if !hasImport {
		t.Error("expected import evidence")
	}
}

// TestDetector_Detect_ContentBasedDetection tests moderate confidence detection via content patterns.
func TestDetector_Detect_ContentBasedDetection(t *testing.T) {
	registry := framework.NewRegistry()
	registry.Register(&framework.Definition{
		Name:      "playwright",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("@playwright/test", "@playwright/test/"),
			matchers.NewContentMatcherFromStrings(`test\.describe\(`),
		},
	})

	detector := NewDetector(registry)

	// No imports, but has content pattern
	content := []byte(`
test.describe('test suite', () => {
  test('should work', async ({ page }) => {
    await page.goto('https://example.com');
  });
});
`)

	result := detector.Detect(context.Background(), "/project/e2e/test.spec.ts", content)

	if result.Framework != "playwright" {
		t.Errorf("expected framework 'playwright', got '%s'", result.Framework)
	}

	// Should have moderate confidence from content only
	if result.Confidence < 40 {
		t.Errorf("expected confidence >= 40, got %d", result.Confidence)
	}

	if !result.IsModerate() && !result.IsDefinite() {
		t.Errorf("expected moderate or definite confidence level, got %s", result.ConfidenceLevel())
	}

	// Verify evidence
	hasContent := false
	for _, ev := range result.Evidence {
		if ev.Source == "content" && !ev.Negative {
			hasContent = true
		}
	}

	if !hasContent {
		t.Error("expected content evidence")
	}
}

// TestDetector_Detect_MultipleSignals tests confidence accumulation from multiple sources.
func TestDetector_Detect_MultipleSignals(t *testing.T) {
	registry := framework.NewRegistry()
	registry.Register(&framework.Definition{
		Name:      "jest",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("@jest/globals", "@jest/globals/"),
			matchers.NewContentMatcherFromStrings(`describe\(`, `it\(`, `expect\(`),
		},
	})

	detector := NewDetector(registry)

	// Setup scope
	projectScope := framework.NewProjectScope()
	configScope := &framework.ConfigScope{
		ConfigPath:  "/project/jest.config.js",
		BaseDir:     "/project",
		Framework:   "jest",
		GlobalsMode: false,
	}
	projectScope.AddConfig("/project/jest.config.js", configScope)
	detector.SetProjectScope(projectScope)

	// File with scope + import + content
	content := []byte(`
import { describe, it, expect } from '@jest/globals';

describe('test suite', () => {
  it('should work', () => {
    expect(true).toBe(true);
  });
});
`)

	result := detector.Detect(context.Background(), "/project/src/test.spec.ts", content)

	if result.Framework != "jest" {
		t.Errorf("expected framework 'jest', got '%s'", result.Framework)
	}

	// Should have very high confidence (capped at 100)
	if result.Confidence < 90 {
		t.Errorf("expected confidence >= 90, got %d", result.Confidence)
	}

	if result.Confidence > 100 {
		t.Errorf("expected confidence capped at 100, got %d", result.Confidence)
	}

	// Verify multiple evidence sources
	sources := make(map[string]bool)
	for _, ev := range result.Evidence {
		if !ev.Negative {
			sources[ev.Source] = true
		}
	}

	if !sources["config-scope"] {
		t.Error("expected config-scope evidence")
	}
	if !sources["import"] {
		t.Error("expected import evidence")
	}
	if !sources["content"] {
		t.Error("expected content evidence")
	}
}

// TestDetector_Detect_NegativeEvidence tests that negative evidence excludes frameworks.
func TestDetector_Detect_NegativeEvidence(t *testing.T) {
	registry := framework.NewRegistry()

	// Register Jest
	registry.Register(&framework.Definition{
		Name:      "jest",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("@jest/globals", "@jest/globals/"),
			matchers.NewContentMatcherFromStrings(`describe\(`, `it\(`, `expect\(`),
		},
	})

	// Register Vitest with negative matcher for Jest imports
	registry.Register(&framework.Definition{
		Name:      "vitest",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("vitest", "vitest/"),
			&negativeMatcher{pattern: "@jest/globals"},
		},
	})

	detector := NewDetector(registry)

	// File imports from Jest, which should exclude Vitest
	content := []byte(`
import { describe, it, expect } from '@jest/globals';

describe('test suite', () => {
  it('should work', () => {
    expect(true).toBe(true);
  });
});
`)

	result := detector.Detect(context.Background(), "/project/test.spec.ts", content)

	if result.Framework != "jest" {
		t.Errorf("expected framework 'jest', got '%s'", result.Framework)
	}

	// Vitest should not be detected despite content match
	if result.Framework == "vitest" {
		t.Error("vitest should be excluded due to negative evidence")
	}
}

// TestDetector_Detect_NoMatch tests detection when no framework matches.
func TestDetector_Detect_NoMatch(t *testing.T) {
	registry := framework.NewRegistry()
	detector := NewDetector(registry)

	content := []byte(`
console.log('hello world');
`)

	result := detector.Detect(context.Background(), "/project/test.ts", content)

	if result.Framework != "" {
		t.Errorf("expected no framework, got '%s'", result.Framework)
	}

	if result.Confidence != 0 {
		t.Errorf("expected confidence 0, got %d", result.Confidence)
	}

	if result.ConfidenceLevel() != "none" {
		t.Errorf("expected confidence level 'none', got '%s'", result.ConfidenceLevel())
	}
}

// TestDetector_Detect_UnsupportedLanguage tests detection for unsupported file types.
func TestDetector_Detect_UnsupportedLanguage(t *testing.T) {
	registry := framework.NewRegistry()
	registry.Register(&framework.Definition{
		Name:      "jest",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers:  []framework.Matcher{},
	})

	detector := NewDetector(registry)

	content := []byte(`print("hello world")`)

	result := detector.Detect(context.Background(), "/project/test.py", content)

	if result.Framework != "" {
		t.Errorf("expected no framework for Python file, got '%s'", result.Framework)
	}
}

// TestResult_ConfidenceLevels tests confidence level classification.
func TestResult_ConfidenceLevels(t *testing.T) {
	tests := []struct {
		name       string
		confidence int
		wantLevel  string
		wantDef    bool
		wantMod    bool
		wantWeak   bool
	}{
		{"definite high", 100, "definite", true, false, false},
		{"definite low", 71, "definite", true, false, false},
		{"moderate high", 70, "moderate", false, true, false},
		{"moderate low", 31, "moderate", false, true, false},
		{"weak high", 30, "weak", false, false, true},
		{"weak low", 1, "weak", false, false, true},
		{"none", 0, "none", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Result{
				Framework:  "test",
				Confidence: tt.confidence,
			}

			if got := result.ConfidenceLevel(); got != tt.wantLevel {
				t.Errorf("ConfidenceLevel() = %v, want %v", got, tt.wantLevel)
			}

			if got := result.IsDefinite(); got != tt.wantDef {
				t.Errorf("IsDefinite() = %v, want %v", got, tt.wantDef)
			}

			if got := result.IsModerate(); got != tt.wantMod {
				t.Errorf("IsModerate() = %v, want %v", got, tt.wantMod)
			}

			if got := result.IsWeak(); got != tt.wantWeak {
				t.Errorf("IsWeak() = %v, want %v", got, tt.wantWeak)
			}
		})
	}
}

// TestDetectLanguage tests language detection from file extensions.
func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path string
		want domain.Language
	}{
		{"/project/test.ts", domain.LanguageTypeScript},
		{"/project/test.tsx", domain.LanguageTypeScript},
		{"/project/test.js", domain.LanguageJavaScript},
		{"/project/test.jsx", domain.LanguageJavaScript},
		{"/project/test.mjs", domain.LanguageJavaScript},
		{"/project/test.cjs", domain.LanguageJavaScript},
		{"/project/test.go", domain.LanguageGo},
		{"/project/test.py", ""},
		{"/project/test.txt", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := detectLanguage(tt.path); got != tt.want {
				t.Errorf("detectLanguage() = %v, want %v", got, tt.want)
			}
		})
	}
}

// negativeMatcher is a test helper that produces negative matches for a pattern.
type negativeMatcher struct {
	pattern string
}

func (m *negativeMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalImport {
		return framework.NoMatch()
	}

	// Check if import contains the pattern
	matched, err := regexp.MatchString(m.pattern, signal.Value)
	if err != nil || !matched {
		return framework.NoMatch()
	}

	// Return negative match
	return framework.NegativeMatch("conflicting import: " + signal.Value)
}

// TestDetector_Detect_GoFileNotMatchedByJSFramework tests that Go files are not detected by JS frameworks.
// This is a regression test for Bug #1: Go files being detected as vitest/jest.
func TestDetector_Detect_GoFileNotMatchedByJSFramework(t *testing.T) {
	registry := framework.NewRegistry()

	// Register Vitest (JS framework)
	registry.Register(&framework.Definition{
		Name:      "vitest",
		Languages: []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("vitest", "vitest/"),
		},
	})

	// Register Go testing
	registry.Register(&framework.Definition{
		Name:      "go-testing",
		Languages: []domain.Language{domain.LanguageGo},
		Matchers:  []framework.Matcher{},
	})

	detector := NewDetector(registry)

	// Setup vitest config scope that covers the entire project
	projectScope := framework.NewProjectScope()
	vitestScope := &framework.ConfigScope{
		ConfigPath:  "/project/vitest.config.ts",
		BaseDir:     "/project",
		Framework:   "vitest",
		GlobalsMode: false,
	}
	projectScope.AddConfig("/project/vitest.config.ts", vitestScope)
	detector.SetProjectScope(projectScope)

	// Go test file within the vitest scope
	content := []byte(`
package env

import "testing"

func TestEnv(t *testing.T) {
	t.Run("should work", func(t *testing.T) {
		// test code
	})
}
`)

	result := detector.Detect(context.Background(), "/project/src/go/libs/env/env_test.go", content)

	// Go file should NOT be detected as vitest
	if result.Framework == "vitest" {
		t.Errorf("Go file should not be detected as vitest, got framework '%s'", result.Framework)
	}
}

// TestDetector_Detect_ImportPriorityOverScope tests that import evidence takes priority over scope.
// This is a regression test for Bug #2: Playwright files being detected as vitest.
func TestDetector_Detect_ImportPriorityOverScope(t *testing.T) {
	registry := framework.NewRegistry()

	// Register Vitest
	registry.Register(&framework.Definition{
		Name:      "vitest",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("vitest", "vitest/"),
		},
	})

	// Register Playwright
	registry.Register(&framework.Definition{
		Name:      "playwright",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("@playwright/test", "@playwright/test/"),
		},
	})

	detector := NewDetector(registry)

	// Setup overlapping scopes - vitest scope is broader
	projectScope := framework.NewProjectScope()
	vitestScope := &framework.ConfigScope{
		ConfigPath:  "/project/src/extension/vitest.config.ts",
		BaseDir:     "/project/src", // Covers all of src/
		Framework:   "vitest",
		GlobalsMode: false,
	}
	playwrightScope := &framework.ConfigScope{
		ConfigPath:  "/project/src/view/playwright.config.ts",
		BaseDir:     "/project/src/view", // More specific, covers src/view/
		Framework:   "playwright",
		GlobalsMode: false,
	}
	projectScope.AddConfig("/project/src/extension/vitest.config.ts", vitestScope)
	projectScope.AddConfig("/project/src/view/playwright.config.ts", playwrightScope)
	detector.SetProjectScope(projectScope)

	// Playwright test file with @playwright/test import
	content := []byte(`
import { expect, test } from "@playwright/test";

test("should work", async ({ page }) => {
	await page.goto("/");
	await expect(page).toHaveTitle("Test");
});
`)

	result := detector.Detect(context.Background(), "/project/src/view/e2e/add-group-command.spec.ts", content)

	// Should be detected as playwright due to import, not vitest
	if result.Framework != "playwright" {
		t.Errorf("expected framework 'playwright', got '%s'", result.Framework)
	}

	// Verify import evidence exists
	hasImport := false
	for _, ev := range result.Evidence {
		if ev.Source == "import" && !ev.Negative {
			hasImport = true
			break
		}
	}
	if !hasImport {
		t.Error("expected import evidence for playwright")
	}
}

// TestDetector_Detect_DeeperScopePriority tests that more specific (deeper) scope takes priority.
func TestDetector_Detect_DeeperScopePriority(t *testing.T) {
	registry := framework.NewRegistry()

	// Register Vitest
	registry.Register(&framework.Definition{
		Name:      "vitest",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("vitest", "vitest/"),
		},
	})

	// Register Playwright
	registry.Register(&framework.Definition{
		Name:      "playwright",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("@playwright/test", "@playwright/test/"),
		},
	})

	detector := NewDetector(registry)

	// Setup overlapping scopes
	projectScope := framework.NewProjectScope()
	vitestScope := &framework.ConfigScope{
		ConfigPath:  "/project/vitest.config.ts",
		BaseDir:     "/project", // Shallow - covers entire project
		Framework:   "vitest",
		GlobalsMode: true,
	}
	playwrightScope := &framework.ConfigScope{
		ConfigPath:  "/project/e2e/playwright.config.ts",
		BaseDir:     "/project/e2e", // Deeper - more specific
		Framework:   "playwright",
		GlobalsMode: false,
	}
	projectScope.AddConfig("/project/vitest.config.ts", vitestScope)
	projectScope.AddConfig("/project/e2e/playwright.config.ts", playwrightScope)
	detector.SetProjectScope(projectScope)

	// Test file without imports - should use deeper scope
	content := []byte(`
test("should work", async ({ page }) => {
	await page.goto("/");
});
`)

	result := detector.Detect(context.Background(), "/project/e2e/login.spec.ts", content)

	// Should be detected as playwright due to deeper scope
	if result.Framework != "playwright" {
		t.Errorf("expected framework 'playwright' (deeper scope), got '%s'", result.Framework)
	}
}
