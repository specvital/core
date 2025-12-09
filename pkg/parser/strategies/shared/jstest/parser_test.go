package jstest

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
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
		{
			name:     "should handle path with directory",
			filename: "src/components/Button.test.tsx",
			want:     domain.LanguageTypeScript,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := DetectLanguage(tt.filename)

			if got != tt.want {
				t.Errorf("DetectLanguage(%q) = %q, want %q", tt.filename, got, tt.want)
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
		framework  string
		wantSuites int
		wantTests  int
		wantLang   domain.Language
	}{
		{
			name: "should parse describe with tests",
			source: `describe('Suite', () => {
				it('test1', () => {});
				it('test2', () => {});
			});`,
			filename:   "test.ts",
			framework:  "jest",
			wantSuites: 1,
			wantTests:  0,
			wantLang:   domain.LanguageTypeScript,
		},
		{
			name:       "should parse top-level tests",
			source:     `it('test1', () => {}); test('test2', () => {});`,
			filename:   "test.ts",
			framework:  "vitest",
			wantSuites: 0,
			wantTests:  2,
			wantLang:   domain.LanguageTypeScript,
		},
		{
			name:       "should parse empty file",
			source:     "",
			filename:   "test.ts",
			framework:  "jest",
			wantSuites: 0,
			wantTests:  0,
			wantLang:   domain.LanguageTypeScript,
		},
		{
			name: "should parse nested describes",
			source: `describe('Outer', () => {
				describe('Inner', () => {
					it('test', () => {});
				});
			});`,
			filename:   "test.ts",
			framework:  "jest",
			wantSuites: 1,
			wantTests:  0,
			wantLang:   domain.LanguageTypeScript,
		},
		{
			name:       "should detect JavaScript language",
			source:     `describe('Suite', () => { it('test', () => {}); });`,
			filename:   "test.js",
			framework:  "jest",
			wantSuites: 1,
			wantTests:  0,
			wantLang:   domain.LanguageJavaScript,
		},
		{
			name:       "should use provided framework name",
			source:     `it('test', () => {});`,
			filename:   "test.ts",
			framework:  "custom-framework",
			wantSuites: 0,
			wantTests:  1,
			wantLang:   domain.LanguageTypeScript,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := Parse(context.Background(), []byte(tt.source), tt.filename, tt.framework)

			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if file.Framework != tt.framework {
				t.Errorf("Framework = %q, want %q", file.Framework, tt.framework)
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

			if file.Path != tt.filename {
				t.Errorf("Path = %q, want %q", file.Path, tt.filename)
			}
		})
	}
}

func TestParse_Modifiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		wantStatus domain.TestStatus
		isSuite    bool
	}{
		{
			name:       "should parse it.skip",
			source:     `it.skip('test', () => {});`,
			wantStatus: domain.TestStatusSkipped,
			isSuite:    false,
		},
		{
			name:       "should parse it.only",
			source:     `it.only('test', () => {});`,
			wantStatus: domain.TestStatusFocused,
			isSuite:    false,
		},
		{
			name:       "should parse test.todo",
			source:     `test.todo('test');`,
			wantStatus: domain.TestStatusTodo,
			isSuite:    false,
		},
		{
			name:       "should parse xit",
			source:     `xit('test', () => {});`,
			wantStatus: domain.TestStatusSkipped,
			isSuite:    false,
		},
		{
			name:       "should parse fit",
			source:     `fit('test', () => {});`,
			wantStatus: domain.TestStatusFocused,
			isSuite:    false,
		},
		{
			name:       "should parse describe.skip",
			source:     `describe.skip('Suite', () => {});`,
			wantStatus: domain.TestStatusSkipped,
			isSuite:    true,
		},
		{
			name:       "should parse describe.only",
			source:     `describe.only('Suite', () => {});`,
			wantStatus: domain.TestStatusFocused,
			isSuite:    true,
		},
		{
			name:       "should parse xdescribe",
			source:     `xdescribe('Suite', () => {});`,
			wantStatus: domain.TestStatusSkipped,
			isSuite:    true,
		},
		{
			name:       "should parse fdescribe",
			source:     `fdescribe('Suite', () => {});`,
			wantStatus: domain.TestStatusFocused,
			isSuite:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := Parse(context.Background(), []byte(tt.source), "test.ts", "jest")

			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			var status domain.TestStatus
			if tt.isSuite {
				if len(file.Suites) != 1 {
					t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
				}
				status = file.Suites[0].Status
			} else {
				if len(file.Tests) != 1 {
					t.Fatalf("len(Tests) = %d, want 1", len(file.Tests))
				}
				status = file.Tests[0].Status
			}

			if status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", status, tt.wantStatus)
			}
		})
	}
}

func TestParse_Each(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		source    string
		wantCount int
		wantFirst string
		isSuite   bool
	}{
		{
			name:      "should parse describe.each with arrays",
			source:    `describe.each([['a'], ['b']])('case %s', () => {});`,
			wantCount: 2,
			wantFirst: "case a",
			isSuite:   true,
		},
		{
			name:      "should parse it.each with arrays",
			source:    `it.each([[1], [2], [3]])('test %d', () => {});`,
			wantCount: 3,
			wantFirst: "test 1",
			isSuite:   false,
		},
		{
			name:      "should parse test.each with strings",
			source:    `test.each(['foo', 'bar'])('val %s', () => {});`,
			wantCount: 2,
			wantFirst: "val foo",
			isSuite:   false,
		},
		{
			name:      "should handle dynamic cases",
			source:    `it.each(testData)('test %s', () => {});`,
			wantCount: 1,
			wantFirst: "test %s (dynamic cases)",
			isSuite:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := Parse(context.Background(), []byte(tt.source), "test.ts", "jest")

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
				if len(file.Tests) != tt.wantCount {
					t.Fatalf("len(Tests) = %d, want %d", len(file.Tests), tt.wantCount)
				}
				if file.Tests[0].Name != tt.wantFirst {
					t.Errorf("Tests[0].Name = %q, want %q", file.Tests[0].Name, tt.wantFirst)
				}
			}
		})
	}
}

func TestParse_Location(t *testing.T) {
	t.Parallel()

	source := `describe('Suite', () => {
  it('test', () => {});
});`

	file, err := Parse(context.Background(), []byte(source), "user.test.ts", "jest")

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(file.Suites) != 1 {
		t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
	}

	suite := file.Suites[0]
	if suite.Location.File != "user.test.ts" {
		t.Errorf("Suite.Location.File = %q, want %q", suite.Location.File, "user.test.ts")
	}
	if suite.Location.StartLine != 1 {
		t.Errorf("Suite.Location.StartLine = %d, want 1", suite.Location.StartLine)
	}

	if len(suite.Tests) != 1 {
		t.Fatalf("len(suite.Tests) = %d, want 1", len(suite.Tests))
	}

	test := suite.Tests[0]
	if test.Location.StartLine != 2 {
		t.Errorf("Test.Location.StartLine = %d, want 2", test.Location.StartLine)
	}
}

func TestResolveEachNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		template  string
		testCases []string
		want      []string
	}{
		{
			name:      "should resolve with test cases",
			template:  "test %s",
			testCases: []string{"a", "b"},
			want:      []string{"test a", "test b"},
		},
		{
			name:      "should add dynamic suffix when empty",
			template:  "test %s",
			testCases: []string{},
			want:      []string{"test %s (dynamic cases)"},
		},
		{
			name:      "should handle nil",
			template:  "test %s",
			testCases: nil,
			want:      []string{"test %s (dynamic cases)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ResolveEachNames(tt.template, tt.testCases)

			if len(got) != len(tt.want) {
				t.Fatalf("len(result) = %d, want %d", len(got), len(tt.want))
			}

			for i, want := range tt.want {
				if got[i] != want {
					t.Errorf("result[%d] = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}
