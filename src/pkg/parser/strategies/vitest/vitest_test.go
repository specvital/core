package vitest

import (
	"context"
	"testing"

	"github.com/specvital/core/domain"
	"github.com/specvital/core/parser/strategies"
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
	if name != "vitest" {
		t.Errorf("Name() = %q, want %q", name, "vitest")
	}
}

func TestStrategy_Priority(t *testing.T) {
	t.Parallel()

	// Given
	s := NewStrategy()

	// When
	priority := s.Priority()

	// Then
	expectedPriority := strategies.DefaultPriority + 10
	if priority != expectedPriority {
		t.Errorf("Priority() = %d, want %d", priority, expectedPriority)
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
			name:     "should handle .test.ts with vitest import",
			filename: "user.test.ts",
			content:  `import { describe, it } from 'vitest';`,
			want:     true,
		},
		{
			name:     "should handle .spec.ts with vitest import",
			filename: "user.spec.ts",
			content:  `import { expect } from 'vitest';`,
			want:     true,
		},
		{
			name:     "should handle __tests__ directory with vitest import",
			filename: "__tests__/user.ts",
			content:  `import { vi } from 'vitest';`,
			want:     true,
		},
		{
			name:     "should reject test file without vitest import",
			filename: "user.test.ts",
			content:  `import { describe, it } from '@jest/globals';`,
			want:     false,
		},
		{
			name:     "should reject non-test file even with vitest import",
			filename: "user.ts",
			content:  `import { describe } from 'vitest';`,
			want:     false,
		},
		{
			name:     "should handle vitest in require statement",
			filename: "user.test.js",
			content:  `const { describe } = require('vitest');`,
			want:     true,
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

func TestStrategy_Parse(t *testing.T) {
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
			name: "should parse simple describe",
			source: `import { describe, it } from 'vitest';
describe('Suite', () => {
	it('test1', () => {});
	it('test2', () => {});
});`,
			filename:   "user.test.ts",
			wantSuites: 1,
			wantTests:  0,
			wantLang:   domain.LanguageTypeScript,
		},
		{
			name: "should parse nested describe",
			source: `import { describe, it } from 'vitest';
describe('Outer', () => {
	describe('Inner', () => {
		it('test', () => {});
	});
});`,
			filename:   "user.test.ts",
			wantSuites: 1,
			wantTests:  0,
			wantLang:   domain.LanguageTypeScript,
		},
		{
			name:       "should parse top-level tests",
			source:     `import { it, test } from 'vitest'; it('test1', () => {}); test('test2', () => {});`,
			filename:   "user.test.ts",
			wantSuites: 0,
			wantTests:  2,
			wantLang:   domain.LanguageTypeScript,
		},
		{
			name:       "should detect JavaScript",
			source:     `import { describe, it } from 'vitest'; describe('JS', () => { it('test', () => {}); });`,
			filename:   "user.test.js",
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
			if file.Framework != "vitest" {
				t.Errorf("Framework = %q, want %q", file.Framework, "vitest")
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

func TestStrategy_Parse_Modifiers(t *testing.T) {
	t.Parallel()

	s := &Strategy{}

	tests := []struct {
		name       string
		source     string
		wantName   string
		wantStatus domain.TestStatus
		isTest     bool
	}{
		{
			name:       "should parse it.skip",
			source:     `describe('S', () => { it.skip('test', () => {}); });`,
			wantName:   "test",
			wantStatus: domain.TestStatusSkipped,
			isTest:     true,
		},
		{
			name:       "should parse it.only",
			source:     `describe('S', () => { it.only('test', () => {}); });`,
			wantName:   "test",
			wantStatus: domain.TestStatusOnly,
			isTest:     true,
		},
		{
			name:       "should parse test.todo",
			source:     `describe('S', () => { test.todo('test'); });`,
			wantName:   "test",
			wantStatus: domain.TestStatusSkipped,
			isTest:     true,
		},
		{
			name:       "should parse describe.skip",
			source:     `describe.skip('Suite', () => {});`,
			wantName:   "Suite",
			wantStatus: domain.TestStatusSkipped,
			isTest:     false,
		},
		{
			name:       "should parse describe.only",
			source:     `describe.only('Suite', () => {});`,
			wantName:   "Suite",
			wantStatus: domain.TestStatusOnly,
			isTest:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// When
			file, err := s.Parse(context.Background(), []byte(tt.source), "test.ts")

			// Then
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if tt.isTest {
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
			} else {
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
			}
		})
	}
}

func TestStrategy_Parse_Each(t *testing.T) {
	t.Parallel()

	s := &Strategy{}

	tests := []struct {
		name      string
		source    string
		wantCount int
		wantFirst string
		isSuite   bool
	}{
		{
			name:      "should parse describe.each",
			source:    `describe.each([['a'], ['b']])('case %s', () => {});`,
			wantCount: 2,
			wantFirst: "case a",
			isSuite:   true,
		},
		{
			name:      "should parse it.each",
			source:    `describe('S', () => { it.each([[1], [2], [3]])('test %d', () => {}); });`,
			wantCount: 3,
			wantFirst: "test 1",
			isSuite:   false,
		},
		{
			name:      "should parse test.each",
			source:    `describe('S', () => { test.each([['x']])('val %s', () => {}); });`,
			wantCount: 1,
			wantFirst: "val x",
			isSuite:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// When
			file, err := s.Parse(context.Background(), []byte(tt.source), "test.ts")

			// Then
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if tt.isSuite {
				if len(file.Suites) != tt.wantCount {
					t.Fatalf("len(Suites) = %d, want %d", len(file.Suites), tt.wantCount)
				}
				if file.Suites[0].Name != tt.wantFirst {
					t.Errorf("Suites[0].Name = %q, want %q", file.Suites[0].Name, tt.wantFirst)
				}
			} else {
				if len(file.Suites) != 1 {
					t.Fatal("Expected 1 suite")
				}
				if len(file.Suites[0].Tests) != tt.wantCount {
					t.Fatalf("len(Tests) = %d, want %d", len(file.Suites[0].Tests), tt.wantCount)
				}
				if file.Suites[0].Tests[0].Name != tt.wantFirst {
					t.Errorf("Tests[0].Name = %q, want %q", file.Suites[0].Tests[0].Name, tt.wantFirst)
				}
			}
		})
	}
}

func TestRegisterDefault(t *testing.T) {
	strategies.DefaultRegistry().Clear()
	defer strategies.DefaultRegistry().Clear()

	// When
	RegisterDefault()

	// Then
	all := strategies.GetStrategies()
	if len(all) != 1 {
		t.Fatalf("len(strategies) = %d, want 1", len(all))
	}
	if all[0].Name() != "vitest" {
		t.Errorf("Name = %q, want %q", all[0].Name(), "vitest")
	}
}

func TestVitestHigherPriorityThanJest(t *testing.T) {
	t.Parallel()

	// Given
	vitestStrategy := NewStrategy()
	jestPriority := strategies.DefaultPriority

	// When
	vitestPriority := vitestStrategy.Priority()

	// Then
	if vitestPriority <= jestPriority {
		t.Errorf("Vitest priority (%d) should be higher than Jest priority (%d)", vitestPriority, jestPriority)
	}
}
