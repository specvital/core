package jest

import (
	"testing"

	"github.com/specvital/core/domain"
)

func TestDetectLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		want     domain.Language
	}{
		{
			name:     "should detect JavaScript for .js",
			filename: "test.js",
			want:     domain.LanguageJavaScript,
		},
		{
			name:     "should detect JavaScript for .jsx",
			filename: "test.jsx",
			want:     domain.LanguageJavaScript,
		},
		{
			name:     "should detect TypeScript for .ts",
			filename: "test.ts",
			want:     domain.LanguageTypeScript,
		},
		{
			name:     "should detect TypeScript for .tsx",
			filename: "test.tsx",
			want:     domain.LanguageTypeScript,
		},
		{
			name:     "should default to TypeScript for unknown extension",
			filename: "test.mjs",
			want:     domain.LanguageTypeScript,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// When
			got := detectLanguage(tt.filename)

			// Then
			if got != tt.want {
				t.Errorf("detectLanguage(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		filename   string
		wantSuites int
		wantTests  int
		wantErr    bool
	}{
		{
			name:       "should parse describe with tests",
			source:     `describe('Suite', () => { it('test', () => {}); });`,
			filename:   "test.ts",
			wantSuites: 1,
			wantTests:  0,
		},
		{
			name:       "should parse top-level tests",
			source:     `it('test1', () => {}); test('test2', () => {});`,
			filename:   "test.ts",
			wantSuites: 0,
			wantTests:  2,
		},
		{
			name:       "should parse empty file",
			source:     "",
			filename:   "test.ts",
			wantSuites: 0,
			wantTests:  0,
		},
		{
			name:       "should parse nested describes",
			source:     `describe('Outer', () => { describe('Inner', () => { it('test', () => {}); }); });`,
			filename:   "test.ts",
			wantSuites: 1,
			wantTests:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// When
			file, err := parse([]byte(tt.source), tt.filename)

			// Then
			if tt.wantErr {
				if err == nil {
					t.Error("parse() expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("parse() error = %v", err)
			}

			if len(file.Suites) != tt.wantSuites {
				t.Errorf("len(Suites) = %d, want %d", len(file.Suites), tt.wantSuites)
			}

			if len(file.Tests) != tt.wantTests {
				t.Errorf("len(Tests) = %d, want %d", len(file.Tests), tt.wantTests)
			}

			if file.Framework != "jest" {
				t.Errorf("Framework = %q, want %q", file.Framework, "jest")
			}

			if file.Path != tt.filename {
				t.Errorf("Path = %q, want %q", file.Path, tt.filename)
			}
		})
	}
}

func TestParseJestNode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		source          string
		wantSuites      int
		wantTests       int
		wantNestedTests int
	}{
		{
			name:            "should parse describe.each",
			source:          `describe.each([['a'], ['b']])('case %s', () => { it('test', () => {}); });`,
			wantSuites:      2,
			wantTests:       0,
			wantNestedTests: 1,
		},
		{
			name:            "should parse it.each inside describe",
			source:          `describe('Suite', () => { it.each([[1], [2]])('test %s', () => {}); });`,
			wantSuites:      1,
			wantTests:       0,
			wantNestedTests: 2,
		},
		{
			name:       "should handle mixed content",
			source:     `describe('A', () => {}); it('top', () => {}); describe('B', () => {});`,
			wantSuites: 2,
			wantTests:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// When
			file, err := parse([]byte(tt.source), "test.ts")

			// Then
			if err != nil {
				t.Fatalf("parse() error = %v", err)
			}

			if len(file.Suites) != tt.wantSuites {
				t.Errorf("len(Suites) = %d, want %d", len(file.Suites), tt.wantSuites)
			}

			if len(file.Tests) != tt.wantTests {
				t.Errorf("len(Tests) = %d, want %d", len(file.Tests), tt.wantTests)
			}

			if tt.wantNestedTests > 0 && len(file.Suites) > 0 {
				nestedCount := len(file.Suites[0].Tests)
				if nestedCount != tt.wantNestedTests {
					t.Errorf("len(Suites[0].Tests) = %d, want %d", nestedCount, tt.wantNestedTests)
				}
			}
		})
	}
}
