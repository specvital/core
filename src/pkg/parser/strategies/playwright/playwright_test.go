package playwright

import (
	"context"
	"testing"

	"github.com/specvital/core/src/pkg/domain"
	"github.com/specvital/core/src/pkg/parser/strategies"
)

func TestNewStrategy(t *testing.T) {
	t.Parallel()

	// When
	s := NewStrategy()

	// Then
	if s == nil {
		t.Fatal("NewStrategy() returned nil")
	}
}

func TestStrategy_Name(t *testing.T) {
	t.Parallel()

	// Given
	s := NewStrategy()

	// When
	name := s.Name()

	// Then
	if name != "playwright" {
		t.Errorf("Name() = %q, want %q", name, "playwright")
	}
}

func TestStrategy_Priority(t *testing.T) {
	t.Parallel()

	// Given
	s := NewStrategy()

	// When
	priority := s.Priority()

	// Then
	if priority != strategies.DefaultPriority {
		t.Errorf("Priority() = %d, want %d", priority, strategies.DefaultPriority)
	}
}

func TestStrategy_Languages(t *testing.T) {
	t.Parallel()

	// Given
	s := NewStrategy()

	// When
	langs := s.Languages()

	// Then
	if len(langs) != 2 {
		t.Fatalf("len(Languages()) = %d, want 2", len(langs))
	}
	if langs[0] != domain.LanguageTypeScript {
		t.Errorf("Languages()[0] = %q, want %q", langs[0], domain.LanguageTypeScript)
	}
	if langs[1] != domain.LanguageJavaScript {
		t.Errorf("Languages()[1] = %q, want %q", langs[1], domain.LanguageJavaScript)
	}
}

func TestStrategy_CanHandle(t *testing.T) {
	t.Parallel()

	s := &Strategy{}

	tests := []struct {
		name     string
		filename string
		content  string
		want     bool
	}{
		{
			name:     "should handle .test.ts with playwright import",
			filename: "user.test.ts",
			content:  `import { test, expect } from '@playwright/test';`,
			want:     true,
		},
		{
			name:     "should handle .spec.ts with playwright import",
			filename: "user.spec.ts",
			content:  `import { test } from '@playwright/test';`,
			want:     true,
		},
		{
			name:     "should handle __tests__ directory with playwright import",
			filename: "__tests__/user.ts",
			content:  `import { expect } from '@playwright/test';`,
			want:     true,
		},
		{
			name:     "should reject test file without playwright import",
			filename: "user.test.ts",
			content:  `import { test } from 'vitest';`,
			want:     false,
		},
		{
			name:     "should reject non-test file even with playwright import",
			filename: "user.ts",
			content:  `import { test } from '@playwright/test';`,
			want:     false,
		},
		{
			name:     "should handle playwright in require statement",
			filename: "user.test.js",
			content:  `const { test } = require('@playwright/test');`,
			want:     true,
		},
		{
			name:     "should reject tsx files",
			filename: "user.test.tsx",
			content:  `import { test } from '@playwright/test';`,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// When
			got := s.CanHandle(tt.filename, []byte(tt.content))

			// Then
			if got != tt.want {
				t.Errorf("CanHandle(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestStrategy_Parse_BasicStructure(t *testing.T) {
	t.Parallel()

	s := &Strategy{}

	tests := []struct {
		name       string
		source     string
		filename   string
		wantSuites int
		wantTests  int
		wantLang   domain.Language
	}{
		{
			name: "should parse test.describe with tests",
			source: `import { test, expect } from '@playwright/test';
test.describe('Login', () => {
	test('should display login form', async ({ page }) => {});
	test('should handle valid credentials', async ({ page }) => {});
});`,
			filename:   "login.spec.ts",
			wantSuites: 1,
			wantTests:  0,
			wantLang:   domain.LanguageTypeScript,
		},
		{
			name: "should parse nested test.describe",
			source: `import { test } from '@playwright/test';
test.describe('Auth', () => {
	test.describe('Login', () => {
		test('works', async () => {});
	});
});`,
			filename:   "auth.spec.ts",
			wantSuites: 1,
			wantTests:  0,
			wantLang:   domain.LanguageTypeScript,
		},
		{
			name:       "should parse top-level tests",
			source:     `import { test } from '@playwright/test'; test('test1', async () => {}); test('test2', async () => {});`,
			filename:   "basic.spec.ts",
			wantSuites: 0,
			wantTests:  2,
			wantLang:   domain.LanguageTypeScript,
		},
		{
			name:       "should detect JavaScript",
			source:     `import { test } from '@playwright/test'; test.describe('JS', () => { test('works', async () => {}); });`,
			filename:   "basic.spec.js",
			wantSuites: 1,
			wantTests:  0,
			wantLang:   domain.LanguageJavaScript,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// When
			file, err := s.Parse(context.Background(), []byte(tt.source), tt.filename)

			// Then
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if file.Framework != "playwright" {
				t.Errorf("Framework = %q, want %q", file.Framework, "playwright")
			}
			if len(file.Suites) != tt.wantSuites {
				t.Errorf("len(Suites) = %d, want %d", len(file.Suites), tt.wantSuites)
			}
			if len(file.Tests) != tt.wantTests {
				t.Errorf("len(Tests) = %d, want %d", len(file.Tests), tt.wantTests)
			}
			if file.Language != tt.wantLang {
				t.Errorf("Language = %q, want %q", file.Language, tt.wantLang)
			}
		})
	}
}

func TestStrategy_Parse_TestModifiers(t *testing.T) {
	t.Parallel()

	s := &Strategy{}

	tests := []struct {
		name       string
		source     string
		wantName   string
		wantStatus domain.TestStatus
	}{
		{
			name:       "should parse test.skip",
			source:     `test.describe('S', () => { test.skip('skipped test', async () => {}); });`,
			wantName:   "skipped test",
			wantStatus: domain.TestStatusSkipped,
		},
		{
			name:       "should parse test.only",
			source:     `test.describe('S', () => { test.only('focused test', async () => {}); });`,
			wantName:   "focused test",
			wantStatus: domain.TestStatusOnly,
		},
		{
			name:       "should parse test.fixme",
			source:     `test.describe('S', () => { test.fixme('broken test', async () => {}); });`,
			wantName:   "broken test",
			wantStatus: domain.TestStatusFixme,
		},
		{
			name:       "should parse regular test as pending",
			source:     `test.describe('S', () => { test('normal test', async () => {}); });`,
			wantName:   "normal test",
			wantStatus: domain.TestStatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// When
			file, err := s.Parse(context.Background(), []byte(tt.source), "test.spec.ts")

			// Then
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if len(file.Suites) != 1 || len(file.Suites[0].Tests) != 1 {
				t.Fatal("Expected 1 suite with 1 test")
			}
			test := file.Suites[0].Tests[0]
			if test.Name != tt.wantName {
				t.Errorf("Test.Name = %q, want %q", test.Name, tt.wantName)
			}
			if test.Status != tt.wantStatus {
				t.Errorf("Test.Status = %q, want %q", test.Status, tt.wantStatus)
			}
		})
	}
}

func TestStrategy_Parse_DescribeModifiers(t *testing.T) {
	t.Parallel()

	s := &Strategy{}

	tests := []struct {
		name       string
		source     string
		wantName   string
		wantStatus domain.TestStatus
	}{
		{
			name:       "should parse test.describe.skip",
			source:     `test.describe.skip('Skipped Suite', () => {});`,
			wantName:   "Skipped Suite",
			wantStatus: domain.TestStatusSkipped,
		},
		{
			name:       "should parse test.describe.only",
			source:     `test.describe.only('Focused Suite', () => {});`,
			wantName:   "Focused Suite",
			wantStatus: domain.TestStatusOnly,
		},
		{
			name:       "should parse test.describe.fixme",
			source:     `test.describe.fixme('Broken Suite', () => {});`,
			wantName:   "Broken Suite",
			wantStatus: domain.TestStatusFixme,
		},
		{
			name:       "should parse regular test.describe as pending",
			source:     `test.describe('Normal Suite', () => {});`,
			wantName:   "Normal Suite",
			wantStatus: domain.TestStatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// When
			file, err := s.Parse(context.Background(), []byte(tt.source), "test.spec.ts")

			// Then
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if len(file.Suites) != 1 {
				t.Fatal("Expected 1 suite")
			}
			suite := file.Suites[0]
			if suite.Name != tt.wantName {
				t.Errorf("Suite.Name = %q, want %q", suite.Name, tt.wantName)
			}
			if suite.Status != tt.wantStatus {
				t.Errorf("Suite.Status = %q, want %q", suite.Status, tt.wantStatus)
			}
		})
	}
}

func TestStrategy_Parse_NestedStructure(t *testing.T) {
	t.Parallel()

	s := &Strategy{}

	source := `import { test, expect } from '@playwright/test';

test.describe('Authentication', () => {
	test.describe('Login', () => {
		test('should display login form', async ({ page }) => {});
		test('should handle invalid credentials', async ({ page }) => {});
	});

	test.describe('Logout', () => {
		test('should clear session', async ({ page }) => {});
	});
});`

	// When
	file, err := s.Parse(context.Background(), []byte(source), "auth.spec.ts")

	// Then
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Root level: 1 suite (Authentication)
	if len(file.Suites) != 1 {
		t.Fatalf("len(file.Suites) = %d, want 1", len(file.Suites))
	}

	authSuite := file.Suites[0]
	if authSuite.Name != "Authentication" {
		t.Errorf("authSuite.Name = %q, want %q", authSuite.Name, "Authentication")
	}

	// Authentication has 2 nested suites
	if len(authSuite.Suites) != 2 {
		t.Fatalf("len(authSuite.Suites) = %d, want 2", len(authSuite.Suites))
	}

	// Login suite has 2 tests
	loginSuite := authSuite.Suites[0]
	if loginSuite.Name != "Login" {
		t.Errorf("loginSuite.Name = %q, want %q", loginSuite.Name, "Login")
	}
	if len(loginSuite.Tests) != 2 {
		t.Errorf("len(loginSuite.Tests) = %d, want 2", len(loginSuite.Tests))
	}

	// Logout suite has 1 test
	logoutSuite := authSuite.Suites[1]
	if logoutSuite.Name != "Logout" {
		t.Errorf("logoutSuite.Name = %q, want %q", logoutSuite.Name, "Logout")
	}
	if len(logoutSuite.Tests) != 1 {
		t.Errorf("len(logoutSuite.Tests) = %d, want 1", len(logoutSuite.Tests))
	}
}

func TestStrategy_Parse_Location(t *testing.T) {
	t.Parallel()

	s := &Strategy{}

	source := `import { test } from '@playwright/test';

test('first test', async () => {});`

	// When
	file, err := s.Parse(context.Background(), []byte(source), "test.spec.ts")

	// Then
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(file.Tests) != 1 {
		t.Fatal("Expected 1 test")
	}

	test := file.Tests[0]
	if test.Location.File != "test.spec.ts" {
		t.Errorf("Location.File = %q, want %q", test.Location.File, "test.spec.ts")
	}
	if test.Location.StartLine != 3 {
		t.Errorf("Location.StartLine = %d, want %d", test.Location.StartLine, 3)
	}
}

func TestRegisterDefault(t *testing.T) {
	// Not parallel - modifies global registry state
	strategies.DefaultRegistry().Clear()
	defer strategies.DefaultRegistry().Clear()

	// When
	RegisterDefault()

	// Then
	all := strategies.GetStrategies()
	if len(all) != 1 {
		t.Fatalf("len(strategies) = %d, want 1", len(all))
	}
	if all[0].Name() != "playwright" {
		t.Errorf("Name = %q, want %q", all[0].Name(), "playwright")
	}
}

// mustParse is a test helper that parses source and fails if error occurs.
func mustParse(t *testing.T, s *Strategy, source, filename string) *domain.TestFile {
	t.Helper()
	file, err := s.Parse(context.Background(), []byte(source), filename)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	return file
}

func TestStrategy_Parse_EdgeCases(t *testing.T) {
	t.Parallel()

	s := &Strategy{}

	tests := []struct {
		name       string
		source     string
		wantSuites int
		wantTests  int
	}{
		{
			name:       "should handle empty file",
			source:     ``,
			wantSuites: 0,
			wantTests:  0,
		},
		{
			name:       "should handle file with only imports",
			source:     `import { test } from '@playwright/test';`,
			wantSuites: 0,
			wantTests:  0,
		},
		{
			name:       "should ignore test calls without name",
			source:     `test.describe('S', () => { test(); });`,
			wantSuites: 1,
			wantTests:  0,
		},
		{
			name:       "should handle describe without callback as empty suite",
			source:     `test.describe('Empty');`,
			wantSuites: 1,
			wantTests:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// When
			file := mustParse(t, s, tt.source, "test.spec.ts")

			// Then
			if len(file.Suites) != tt.wantSuites {
				t.Errorf("len(Suites) = %d, want %d", len(file.Suites), tt.wantSuites)
			}
			if len(file.Tests) != tt.wantTests {
				t.Errorf("len(Tests) = %d, want %d", len(file.Tests), tt.wantTests)
			}
		})
	}
}
