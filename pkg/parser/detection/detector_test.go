package detection

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
	"github.com/specvital/core/pkg/parser/framework/matchers"
)

// TestDetector_EarlyReturn_ImportWins tests that import detection returns immediately.
func TestDetector_EarlyReturn_ImportWins(t *testing.T) {
	registry := framework.NewRegistry()
	registry.Register(&framework.Definition{
		Name:      "vitest",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("vitest", "vitest/"),
		},
	})

	detector := NewDetector(registry)

	// Setup scope for jest (should be ignored due to import)
	projectScope := framework.NewProjectScope()
	jestScope := &framework.ConfigScope{
		ConfigPath:  "/project/jest.config.js",
		BaseDir:     "/project",
		Framework:   "jest",
		GlobalsMode: true,
	}
	projectScope.AddConfig("/project/jest.config.js", jestScope)
	detector.SetProjectScope(projectScope)

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

	if result.Source != SourceImport {
		t.Errorf("expected source 'import', got '%s'", result.Source)
	}
}

// TestDetector_EarlyReturn_FallbackToScope tests scope detection when no imports found.
func TestDetector_EarlyReturn_FallbackToScope(t *testing.T) {
	registry := framework.NewRegistry()
	registry.Register(&framework.Definition{
		Name:      "jest",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("@jest/globals", "@jest/globals/"),
		},
	})

	detector := NewDetector(registry)

	projectScope := framework.NewProjectScope()
	jestScope := &framework.ConfigScope{
		ConfigPath:  "/project/jest.config.js",
		BaseDir:     "/project",
		Framework:   "jest",
		GlobalsMode: true,
	}
	projectScope.AddConfig("/project/jest.config.js", jestScope)
	detector.SetProjectScope(projectScope)

	// No imports - should fall back to scope
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

	if result.Source != SourceConfigScope {
		t.Errorf("expected source 'config-scope', got '%s'", result.Source)
	}

	if result.Scope == nil {
		t.Error("expected scope to be set")
	}
}

// TestDetector_EarlyReturn_FallbackToContent tests content detection as last resort.
func TestDetector_EarlyReturn_FallbackToContent(t *testing.T) {
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

	// No imports, no scope - should use content pattern
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

	if result.Source != SourceContentPattern {
		t.Errorf("expected source 'content-pattern', got '%s'", result.Source)
	}
}

// TestDetector_EarlyReturn_Unknown tests unknown result when nothing matches.
func TestDetector_EarlyReturn_Unknown(t *testing.T) {
	registry := framework.NewRegistry()
	detector := NewDetector(registry)

	content := []byte(`
console.log('hello world');
`)

	result := detector.Detect(context.Background(), "/project/test.ts", content)

	if result.Framework != "" {
		t.Errorf("expected no framework, got '%s'", result.Framework)
	}

	if result.Source != SourceUnknown {
		t.Errorf("expected source 'unknown', got '%s'", result.Source)
	}

	if result.IsDetected() {
		t.Error("expected IsDetected() to return false")
	}
}

// TestDetector_GoFileNotMatchedByJSFramework tests that Go files are not detected by JS frameworks.
func TestDetector_GoFileNotMatchedByJSFramework(t *testing.T) {
	registry := framework.NewRegistry()

	registry.Register(&framework.Definition{
		Name:      "vitest",
		Languages: []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("vitest", "vitest/"),
		},
	})

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

// TestDetector_ImportPriorityOverScope tests that import takes priority over scope.
func TestDetector_ImportPriorityOverScope(t *testing.T) {
	registry := framework.NewRegistry()

	registry.Register(&framework.Definition{
		Name:      "vitest",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("vitest", "vitest/"),
		},
	})

	registry.Register(&framework.Definition{
		Name:      "playwright",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("@playwright/test", "@playwright/test/"),
		},
	})

	detector := NewDetector(registry)

	// Setup vitest scope that covers entire project
	projectScope := framework.NewProjectScope()
	vitestScope := &framework.ConfigScope{
		ConfigPath:  "/project/vitest.config.ts",
		BaseDir:     "/project",
		Framework:   "vitest",
		GlobalsMode: false,
	}
	projectScope.AddConfig("/project/vitest.config.ts", vitestScope)
	detector.SetProjectScope(projectScope)

	// Playwright test file with @playwright/test import
	content := []byte(`
import { expect, test } from "@playwright/test";

test("should work", async ({ page }) => {
	await page.goto("/");
	await expect(page).toHaveTitle("Test");
});
`)

	result := detector.Detect(context.Background(), "/project/src/view/e2e/test.spec.ts", content)

	// Import should win over scope
	if result.Framework != "playwright" {
		t.Errorf("expected framework 'playwright', got '%s'", result.Framework)
	}

	if result.Source != SourceImport {
		t.Errorf("expected source 'import', got '%s'", result.Source)
	}
}

// TestDetector_DeeperScopePriority tests that more specific scope wins.
func TestDetector_DeeperScopePriority(t *testing.T) {
	registry := framework.NewRegistry()

	registry.Register(&framework.Definition{
		Name:      "vitest",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("vitest", "vitest/"),
		},
	})

	registry.Register(&framework.Definition{
		Name:      "playwright",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("@playwright/test", "@playwright/test/"),
		},
	})

	detector := NewDetector(registry)

	projectScope := framework.NewProjectScope()
	vitestScope := &framework.ConfigScope{
		ConfigPath:  "/project/vitest.config.ts",
		BaseDir:     "/project", // Shallow
		Framework:   "vitest",
		GlobalsMode: true,
	}
	playwrightScope := &framework.ConfigScope{
		ConfigPath:  "/project/e2e/playwright.config.ts",
		BaseDir:     "/project/e2e", // Deeper
		Framework:   "playwright",
		GlobalsMode: false,
	}
	projectScope.AddConfig("/project/vitest.config.ts", vitestScope)
	projectScope.AddConfig("/project/e2e/playwright.config.ts", playwrightScope)
	detector.SetProjectScope(projectScope)

	// No imports - should use deeper scope
	content := []byte(`
test("should work", async ({ page }) => {
	await page.goto("/");
});
`)

	result := detector.Detect(context.Background(), "/project/e2e/login.spec.ts", content)

	if result.Framework != "playwright" {
		t.Errorf("expected framework 'playwright' (deeper scope), got '%s'", result.Framework)
	}

	if result.Source != SourceConfigScope {
		t.Errorf("expected source 'config-scope', got '%s'", result.Source)
	}
}

// TestDetector_UnsupportedLanguage tests detection for unsupported file types.
func TestDetector_UnsupportedLanguage(t *testing.T) {
	registry := framework.NewRegistry()
	registry.Register(&framework.Definition{
		Name:      "jest",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers:  []framework.Matcher{},
	})

	detector := NewDetector(registry)

	// Use a truly unsupported file type (.rb)
	content := []byte(`puts "hello world"`)

	result := detector.Detect(context.Background(), "/project/test.rb", content)

	if result.Framework != "" {
		t.Errorf("expected no framework for Ruby file, got '%s'", result.Framework)
	}

	if result.Source != SourceUnknown {
		t.Errorf("expected source 'unknown', got '%s'", result.Source)
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
		{"/project/test.py", domain.LanguagePython},
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

// TestResult_String tests Result string representation.
func TestResult_String(t *testing.T) {
	tests := []struct {
		name   string
		result Result
		want   string
	}{
		{
			name:   "detected",
			result: Confirmed("jest", SourceImport),
			want:   "jest (source: import)",
		},
		{
			name:   "unknown",
			result: Unknown(),
			want:   "no framework detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDetector_GoTestFile tests Go test file detection by naming convention.
func TestDetector_GoTestFile(t *testing.T) {
	registry := framework.NewRegistry()
	registry.Register(&framework.Definition{
		Name:      "go-testing",
		Languages: []domain.Language{domain.LanguageGo},
		Matchers:  []framework.Matcher{},
	})

	detector := NewDetector(registry)

	content := []byte(`
package service

import "testing"

func TestService(t *testing.T) {
	t.Run("should work", func(t *testing.T) {})
}
`)

	result := detector.Detect(context.Background(), "/project/service_test.go", content)

	if result.Framework != "go-testing" {
		t.Errorf("expected framework 'go-testing', got '%s'", result.Framework)
	}

	if result.Source != SourceContentPattern {
		t.Errorf("expected source 'content-pattern', got '%s'", result.Source)
	}
}

// TestDetector_GoNonTestFile tests that non-test Go files are not detected.
func TestDetector_GoNonTestFile(t *testing.T) {
	registry := framework.NewRegistry()
	registry.Register(&framework.Definition{
		Name:      "go-testing",
		Languages: []domain.Language{domain.LanguageGo},
		Matchers:  []framework.Matcher{},
	})

	detector := NewDetector(registry)

	content := []byte(`
package service

func DoSomething() {}
`)

	result := detector.Detect(context.Background(), "/project/service.go", content)

	if result.Framework != "" {
		t.Errorf("expected no framework for non-test Go file, got '%s'", result.Framework)
	}
}

// TestResult_IsDetected tests IsDetected method.
func TestResult_IsDetected(t *testing.T) {
	tests := []struct {
		name   string
		result Result
		want   bool
	}{
		{"import", Confirmed("jest", SourceImport), true},
		{"scope", Confirmed("vitest", SourceConfigScope), true},
		{"content", Confirmed("playwright", SourceContentPattern), true},
		{"unknown", Unknown(), false},
		{"empty framework", Result{Framework: "", Source: SourceImport}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.IsDetected(); got != tt.want {
				t.Errorf("IsDetected() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDetector_DeterministicScopeSelection verifies that when multiple config scopes
// could match a file, the selection is deterministic across multiple runs.
// This test addresses the nondeterministic map iteration bug that caused
// different files to be detected across CI runs.
func TestDetector_DeterministicScopeSelection(t *testing.T) {
	registry := framework.NewRegistry()
	registry.Register(&framework.Definition{
		Name:      "junit5",
		Languages: []domain.Language{domain.LanguageJava},
		Matchers: []framework.Matcher{
			matchers.NewContentMatcherFromStrings(`@Test`),
		},
	})

	detector := NewDetector(registry)

	// Create multiple config scopes that could potentially match the same file
	// (simulating a multi-module Gradle project like testcontainers-java with 90 modules)
	projectScope := framework.NewProjectScope()

	// Add configs in non-alphabetical order to test deterministic sorting
	configs := []string{
		"/project/modules/module-z/build.gradle",
		"/project/modules/module-a/build.gradle",
		"/project/modules/module-m/build.gradle",
		"/project/build.gradle",
		"/project/modules/module-b/build.gradle",
	}

	for _, configPath := range configs {
		scope := framework.NewConfigScope(configPath, "")
		scope.Framework = "junit5"
		projectScope.AddConfig(configPath, scope)
	}

	detector.SetProjectScope(projectScope)

	// Test file in module-a that could potentially match multiple scopes
	testFile := "/project/modules/module-a/src/test/java/TestFile.java"
	content := []byte(`
		@Test
		public void testSomething() {
		}
	`)

	// Run detection 100 times to verify consistent results
	var firstResult Result
	for i := 0; i < 100; i++ {
		result := detector.Detect(context.Background(), testFile, content)

		if i == 0 {
			firstResult = result
		} else {
			// Every iteration must produce the same result
			if result.Framework != firstResult.Framework {
				t.Errorf("iteration %d: expected framework '%s', got '%s'",
					i, firstResult.Framework, result.Framework)
			}
			if result.Scope != firstResult.Scope {
				t.Errorf("iteration %d: scope mismatch", i)
			}
		}
	}

	if firstResult.Framework != "junit5" {
		t.Errorf("expected framework 'junit5', got '%s'", firstResult.Framework)
	}

	// Verify semantic correctness: module-a config should be selected (most specific)
	if firstResult.Scope == nil {
		t.Fatal("expected scope to be set")
	}
	scope, ok := firstResult.Scope.(*framework.ConfigScope)
	if !ok {
		t.Fatal("scope is not *framework.ConfigScope")
	}
	expectedConfig := "/project/modules/module-a/build.gradle"
	if scope.ConfigPath != expectedConfig {
		t.Errorf("expected config '%s', got '%s'", expectedConfig, scope.ConfigPath)
	}
}

// TestDetector_TieBreakingSameDepthPaths verifies deterministic tie-breaking
// when multiple configs have identical depth and could match the same file.
func TestDetector_TieBreakingSameDepthPaths(t *testing.T) {
	registry := framework.NewRegistry()
	registry.Register(&framework.Definition{
		Name:      "vitest",
		Languages: []domain.Language{domain.LanguageTypeScript},
		Matchers: []framework.Matcher{
			matchers.NewContentMatcherFromStrings(`import { describe }`),
		},
	})

	detector := NewDetector(registry)
	projectScope := framework.NewProjectScope()

	// Two configs at same depth with same path length
	configs := []string{
		"/project/packages/zzz-pkg/vitest.config.ts",
		"/project/packages/aaa-pkg/vitest.config.ts",
	}

	for _, configPath := range configs {
		scope := framework.NewConfigScope(configPath, "")
		scope.Framework = "vitest"
		projectScope.AddConfig(configPath, scope)
	}

	detector.SetProjectScope(projectScope)

	// File that could match either config (in parent directory)
	testFile := "/project/packages/test.spec.ts"
	content := []byte(`import { describe } from 'vitest'`)

	// Run 100 times - must always select same config
	var selectedConfig string
	for i := 0; i < 100; i++ {
		result := detector.Detect(context.Background(), testFile, content)

		if result.Scope == nil {
			continue // No scope match expected for this case
		}

		scope, ok := result.Scope.(*framework.ConfigScope)
		if !ok {
			t.Fatal("scope is not *framework.ConfigScope")
		}

		if i == 0 {
			selectedConfig = scope.ConfigPath
		} else if scope.ConfigPath != selectedConfig {
			t.Errorf("iteration %d: expected '%s', got '%s'",
				i, selectedConfig, scope.ConfigPath)
		}
	}
}

// TestDetector_StrongFilename_CypressOverridesPlaywrightScope tests that .cy.tsx files
// are detected as Cypress even when they exist within a Playwright config scope.
// This is the key bug fix - strong filename patterns should override scope detection.
func TestDetector_StrongFilename_CypressOverridesPlaywrightScope(t *testing.T) {
	registry := framework.NewRegistry()

	// Register Cypress with filename matcher that returns DefiniteMatch(100)
	registry.Register(&framework.Definition{
		Name:      "cypress",
		Languages: []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		Matchers: []framework.Matcher{
			&cypressFilenameMatcher{},
		},
	})

	// Register Playwright with scope detection
	registry.Register(&framework.Definition{
		Name:      "playwright",
		Languages: []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("@playwright/test", "@playwright/test/"),
		},
	})

	detector := NewDetector(registry)

	// Setup Playwright scope that covers the entire project
	projectScope := framework.NewProjectScope()
	playwrightScope := &framework.ConfigScope{
		ConfigPath:  "/project/playwright.config.ts",
		BaseDir:     "/project",
		Framework:   "playwright",
		GlobalsMode: false,
	}
	projectScope.AddConfig("/project/playwright.config.ts", playwrightScope)
	detector.SetProjectScope(projectScope)

	// Cypress test file with Cypress-specific content (but no explicit import)
	content := []byte(`
describe('Button', () => {
  it('renders primary button', async () => {
    cy.mount(<CSF3Primary />);
    cy.get('button').should('contain.text', 'foo');
  });
});
`)

	// Test .cy.tsx file - should be detected as Cypress despite Playwright scope
	result := detector.Detect(context.Background(), "/project/stories/Button.cy.tsx", content)

	if result.Framework != "cypress" {
		t.Errorf("expected framework 'cypress', got '%s'", result.Framework)
	}

	if result.Source != SourceStrongFilename {
		t.Errorf("expected source 'strong-filename', got '%s'", result.Source)
	}
}

// TestDetector_StrongFilename_RegularSpecFileUsesScope tests that regular .spec.ts files
// still use scope detection when no strong filename pattern matches.
func TestDetector_StrongFilename_RegularSpecFileUsesScope(t *testing.T) {
	registry := framework.NewRegistry()

	// Register Cypress with filename matcher
	registry.Register(&framework.Definition{
		Name:      "cypress",
		Languages: []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		Matchers: []framework.Matcher{
			&cypressFilenameMatcher{},
		},
	})

	// Register Playwright
	registry.Register(&framework.Definition{
		Name:      "playwright",
		Languages: []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("@playwright/test", "@playwright/test/"),
		},
	})

	detector := NewDetector(registry)

	// Setup Playwright scope
	projectScope := framework.NewProjectScope()
	playwrightScope := &framework.ConfigScope{
		ConfigPath:  "/project/playwright.config.ts",
		BaseDir:     "/project",
		Framework:   "playwright",
		GlobalsMode: true,
	}
	projectScope.AddConfig("/project/playwright.config.ts", playwrightScope)
	detector.SetProjectScope(projectScope)

	// Regular test file (not .cy.ts) with no explicit import
	content := []byte(`
test('should work', async ({ page }) => {
  await page.goto('/');
});
`)

	// Regular .spec.ts file - should use Playwright scope
	result := detector.Detect(context.Background(), "/project/e2e/login.spec.ts", content)

	if result.Framework != "playwright" {
		t.Errorf("expected framework 'playwright', got '%s'", result.Framework)
	}

	if result.Source != SourceConfigScope {
		t.Errorf("expected source 'config-scope', got '%s'", result.Source)
	}
}

// TestDetector_StrongFilename_ImportStillWins tests that import detection still
// takes priority over strong filename detection.
func TestDetector_StrongFilename_ImportStillWins(t *testing.T) {
	registry := framework.NewRegistry()

	// Register Cypress with filename matcher
	registry.Register(&framework.Definition{
		Name:      "cypress",
		Languages: []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		Matchers: []framework.Matcher{
			&cypressFilenameMatcher{},
		},
	})

	// Register Vitest with import matcher
	registry.Register(&framework.Definition{
		Name:      "vitest",
		Languages: []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		Matchers: []framework.Matcher{
			matchers.NewImportMatcher("vitest", "vitest/"),
		},
	})

	detector := NewDetector(registry)

	// File named .cy.tsx but with vitest import - import should win
	content := []byte(`
import { describe, it, expect } from 'vitest';

describe('test', () => {
  it('works', () => {
    expect(true).toBe(true);
  });
});
`)

	result := detector.Detect(context.Background(), "/project/test.cy.tsx", content)

	if result.Framework != "vitest" {
		t.Errorf("expected framework 'vitest' (import wins), got '%s'", result.Framework)
	}

	if result.Source != SourceImport {
		t.Errorf("expected source 'import', got '%s'", result.Source)
	}
}

// cypressFilenameMatcher is a test double that mimics the real Cypress filename matcher.
type cypressFilenameMatcher struct{}

func (m *cypressFilenameMatcher) Match(_ context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileName {
		return framework.NoMatch()
	}

	// Match *.cy.{js,ts,jsx,tsx} pattern
	filename := signal.Value
	if len(filename) > 6 {
		suffix := filename[len(filename)-6:]
		if suffix == ".cy.ts" || suffix == ".cy.js" {
			return framework.DefiniteMatch("filename: *.cy.{js,ts}")
		}
	}
	if len(filename) > 7 {
		suffix := filename[len(filename)-7:]
		if suffix == ".cy.tsx" || suffix == ".cy.jsx" {
			return framework.DefiniteMatch("filename: *.cy.{jsx,tsx}")
		}
	}

	return framework.NoMatch()
}
