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
			name:     "should detect TSX for .tsx",
			filename: "test.tsx",
			want:     domain.LanguageTSX,
		},
		{
			name:     "should default to TypeScript for unknown extension",
			filename: "test.mjs",
			want:     domain.LanguageTypeScript,
		},
		{
			name:     "should handle path with directory",
			filename: "src/components/Button.test.tsx",
			want:     domain.LanguageTSX,
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
			name:      "should parse describe.each as single dynamic suite (ADR-02)",
			source:    `describe.each([['a'], ['b']])('case %s', () => {});`,
			wantCount: 1,
			wantFirst: "case %s (dynamic cases)",
			isSuite:   true,
		},
		{
			name:      "should parse it.each as single dynamic test (ADR-02)",
			source:    `it.each([[1], [2], [3]])('test %d', () => {});`,
			wantCount: 1,
			wantFirst: "test %d (dynamic cases)",
			isSuite:   false,
		},
		{
			name:      "should parse test.each as single dynamic test (ADR-02)",
			source:    `test.each(['foo', 'bar'])('val %s', () => {});`,
			wantCount: 1,
			wantFirst: "val %s (dynamic cases)",
			isSuite:   false,
		},
		{
			name:      "should handle variable-based dynamic cases",
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

func TestParse_MochaTDDStyle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		wantSuites int
		wantTests  int
	}{
		{
			name: "should parse suite with test",
			source: `suite('Calculator', () => {
				test('adds numbers', () => {});
			});`,
			wantSuites: 1,
			wantTests:  0,
		},
		{
			name: "should parse context with specify",
			source: `context('User', () => {
				specify('validates input', () => {});
			});`,
			wantSuites: 1,
			wantTests:  0,
		},
		{
			name:       "should parse top-level specify",
			source:     `specify('validates', () => {});`,
			wantSuites: 0,
			wantTests:  1,
		},
		{
			name: "should parse nested TDD style",
			source: `suite('Outer', () => {
				suite('Inner', () => {
					test('case', () => {});
				});
			});`,
			wantSuites: 1,
			wantTests:  0,
		},
		{
			name: "should parse mixed BDD and TDD",
			source: `describe('BDD', () => {
				context('with TDD context', () => {
					specify('test case', () => {});
				});
			});`,
			wantSuites: 1,
			wantTests:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := Parse(context.Background(), []byte(tt.source), "test.ts", "mocha")

			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if len(file.Suites) != tt.wantSuites {
				t.Errorf("len(Suites) = %d, want %d", len(file.Suites), tt.wantSuites)
			}

			if len(file.Tests) != tt.wantTests {
				t.Errorf("len(Tests) = %d, want %d", len(file.Tests), tt.wantTests)
			}
		})
	}
}

func TestParse_TDDModifiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		wantStatus domain.TestStatus
		isSuite    bool
	}{
		{
			name:       "should parse suite.skip",
			source:     `suite.skip('Suite', () => {});`,
			wantStatus: domain.TestStatusSkipped,
			isSuite:    true,
		},
		{
			name:       "should parse suite.only",
			source:     `suite.only('Suite', () => {});`,
			wantStatus: domain.TestStatusFocused,
			isSuite:    true,
		},
		{
			name:       "should parse context.skip",
			source:     `context.skip('Context', () => {});`,
			wantStatus: domain.TestStatusSkipped,
			isSuite:    true,
		},
		{
			name:       "should parse context.only",
			source:     `context.only('Context', () => {});`,
			wantStatus: domain.TestStatusFocused,
			isSuite:    true,
		},
		{
			name:       "should parse specify.skip",
			source:     `specify.skip('test', () => {});`,
			wantStatus: domain.TestStatusSkipped,
			isSuite:    false,
		},
		{
			name:       "should parse specify.only",
			source:     `specify.only('test', () => {});`,
			wantStatus: domain.TestStatusFocused,
			isSuite:    false,
		},
		{
			name:       "should parse xcontext",
			source:     `xcontext('Context', () => {});`,
			wantStatus: domain.TestStatusSkipped,
			isSuite:    true,
		},
		{
			name:       "should parse xspecify",
			source:     `xspecify('test', () => {});`,
			wantStatus: domain.TestStatusSkipped,
			isSuite:    false,
		},
		{
			name:       "should parse fcontext",
			source:     `fcontext('Context', () => {});`,
			wantStatus: domain.TestStatusFocused,
			isSuite:    true,
		},
		{
			name:       "should parse fspecify",
			source:     `fspecify('test', () => {});`,
			wantStatus: domain.TestStatusFocused,
			isSuite:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := Parse(context.Background(), []byte(tt.source), "test.ts", "mocha")

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

func TestParse_Concurrent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		wantCount  int
		wantStatus domain.TestStatus
		isSuite    bool
	}{
		{
			name:       "should parse test.concurrent",
			source:     `test.concurrent('async test', async () => {});`,
			wantCount:  1,
			wantStatus: domain.TestStatusActive,
			isSuite:    false,
		},
		{
			name:       "should parse it.concurrent",
			source:     `it.concurrent('async test', async () => {});`,
			wantCount:  1,
			wantStatus: domain.TestStatusActive,
			isSuite:    false,
		},
		{
			name:       "should parse describe.concurrent",
			source:     `describe.concurrent('async suite', () => {});`,
			wantCount:  1,
			wantStatus: domain.TestStatusActive,
			isSuite:    true,
		},
		{
			name:       "should parse test.concurrent.skip",
			source:     `test.concurrent.skip('skipped async', async () => {});`,
			wantCount:  1,
			wantStatus: domain.TestStatusSkipped,
			isSuite:    false,
		},
		{
			name:       "should parse it.concurrent.only",
			source:     `it.concurrent.only('focused async', async () => {});`,
			wantCount:  1,
			wantStatus: domain.TestStatusFocused,
			isSuite:    false,
		},
		{
			name:       "should parse describe.concurrent.skip",
			source:     `describe.concurrent.skip('skipped async suite', () => {});`,
			wantCount:  1,
			wantStatus: domain.TestStatusSkipped,
			isSuite:    true,
		},
		{
			name: "should parse tests inside concurrent suite",
			source: `describe.concurrent('suite', () => {
				it('test1', async () => {});
				it('test2', async () => {});
			});`,
			wantCount:  1,
			wantStatus: domain.TestStatusActive,
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

			if tt.isSuite {
				if len(file.Suites) != tt.wantCount {
					t.Fatalf("len(Suites) = %d, want %d", len(file.Suites), tt.wantCount)
				}
				if file.Suites[0].Status != tt.wantStatus {
					t.Errorf("Status = %q, want %q", file.Suites[0].Status, tt.wantStatus)
				}
			} else {
				if len(file.Tests) != tt.wantCount {
					t.Fatalf("len(Tests) = %d, want %d", len(file.Tests), tt.wantCount)
				}
				if file.Tests[0].Status != tt.wantStatus {
					t.Errorf("Status = %q, want %q", file.Tests[0].Status, tt.wantStatus)
				}
			}
		})
	}
}

func TestParse_ConcurrentEach(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		source    string
		wantCount int
		wantFirst string
		isSuite   bool
	}{
		{
			name:      "should parse test.concurrent.each as single dynamic test (ADR-02)",
			source:    `test.concurrent.each([[1], [2], [3]])('test %d', async () => {});`,
			wantCount: 1,
			wantFirst: "test %d (dynamic cases)",
			isSuite:   false,
		},
		{
			name:      "should parse it.concurrent.each as single dynamic test (ADR-02)",
			source:    `it.concurrent.each([['a'], ['b']])('test %s', async () => {});`,
			wantCount: 1,
			wantFirst: "test %s (dynamic cases)",
			isSuite:   false,
		},
		{
			name:      "should parse describe.concurrent.each as single dynamic suite (ADR-02)",
			source:    `describe.concurrent.each([['x'], ['y']])('suite %s', () => {});`,
			wantCount: 1,
			wantFirst: "suite %s (dynamic cases)",
			isSuite:   true,
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

func TestParse_Bench(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		wantCount  int
		wantName   string
		wantStatus domain.TestStatus
	}{
		{
			name:       "should parse bench",
			source:     `bench('sort array', () => { array.sort(); });`,
			wantCount:  1,
			wantName:   "sort array",
			wantStatus: domain.TestStatusActive,
		},
		{
			name:       "should parse bench.skip",
			source:     `bench.skip('slow sort', () => { array.sort(); });`,
			wantCount:  1,
			wantName:   "slow sort",
			wantStatus: domain.TestStatusSkipped,
		},
		{
			name:       "should parse bench.only",
			source:     `bench.only('critical sort', () => { array.sort(); });`,
			wantCount:  1,
			wantName:   "critical sort",
			wantStatus: domain.TestStatusFocused,
		},
		{
			name: "should parse bench inside describe",
			source: `describe('Sorting', () => {
				bench('sort 1000 items', () => {});
				bench('sort 10000 items', () => {});
			});`,
			wantCount:  0,
			wantName:   "",
			wantStatus: domain.TestStatusActive,
		},
		{
			name:       "should parse multiple top-level bench",
			source:     `bench('bench1', () => {}); bench('bench2', () => {});`,
			wantCount:  2,
			wantName:   "bench1",
			wantStatus: domain.TestStatusActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := Parse(context.Background(), []byte(tt.source), "bench.test.ts", "vitest")

			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if len(file.Tests) != tt.wantCount {
				t.Errorf("len(Tests) = %d, want %d", len(file.Tests), tt.wantCount)
			}

			if tt.wantCount > 0 && len(file.Tests) > 0 {
				if file.Tests[0].Name != tt.wantName {
					t.Errorf("Tests[0].Name = %q, want %q", file.Tests[0].Name, tt.wantName)
				}
				if file.Tests[0].Status != tt.wantStatus {
					t.Errorf("Tests[0].Status = %q, want %q", file.Tests[0].Status, tt.wantStatus)
				}
			}
		})
	}
}

func TestParse_BenchInSuite(t *testing.T) {
	t.Parallel()

	source := `describe('Sorting', () => {
		bench('sort 1000 items', () => {});
		bench.skip('sort 10000 items', () => {});
		bench.only('sort 100 items', () => {});
	});`

	file, err := Parse(context.Background(), []byte(source), "bench.test.ts", "vitest")

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(file.Suites) != 1 {
		t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
	}

	suite := file.Suites[0]
	if len(suite.Tests) != 3 {
		t.Fatalf("len(suite.Tests) = %d, want 3", len(suite.Tests))
	}

	expectedBenches := []struct {
		name   string
		status domain.TestStatus
	}{
		{"sort 1000 items", domain.TestStatusActive},
		{"sort 10000 items", domain.TestStatusSkipped},
		{"sort 100 items", domain.TestStatusFocused},
	}

	for i, expected := range expectedBenches {
		if suite.Tests[i].Name != expected.name {
			t.Errorf("Tests[%d].Name = %q, want %q", i, suite.Tests[i].Name, expected.name)
		}
		if suite.Tests[i].Status != expected.status {
			t.Errorf("Tests[%d].Status = %q, want %q", i, suite.Tests[i].Status, expected.status)
		}
	}
}

func TestParse_ForEachCallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		source    string
		wantCount int
		wantFirst string
	}{
		{
			name:      "should parse forEach with template literal",
			source:    "browsers.forEach((browser) => {\n  it(`supports ${browser}`, () => {});\n});",
			wantCount: 1,
			wantFirst: "supports ${browser} (dynamic cases)",
		},
		{
			name: "should parse forEach with multiple tests",
			source: `testCases.forEach(({ color, status }) => {
  it('renders full mode', () => {});
  it('renders compact mode', () => {});
});`,
			wantCount: 2,
			wantFirst: "renders full mode (dynamic cases)",
		},
		{
			name:      "should parse map with template literal",
			source:    "items.map((item) => {\n  it(`handles ${item}`, () => {});\n});",
			wantCount: 1,
			wantFirst: "handles ${item} (dynamic cases)",
		},
		{
			name: "should parse forEach inside describe",
			source: `describe('Suite', () => {
  testCases.forEach((tc) => {
    it('test case', () => {});
  });
});`,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := Parse(context.Background(), []byte(tt.source), "test.ts", "jest")

			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if len(file.Tests) != tt.wantCount {
				t.Fatalf("len(Tests) = %d, want %d", len(file.Tests), tt.wantCount)
			}

			if tt.wantCount > 0 && file.Tests[0].Name != tt.wantFirst {
				t.Errorf("Tests[0].Name = %q, want %q", file.Tests[0].Name, tt.wantFirst)
			}
		})
	}
}

func TestParse_ForEachInsideSuite(t *testing.T) {
	t.Parallel()

	source := `describe('Badge Renderer', () => {
  testCases.forEach(({ color, status }) => {
    it('renders full mode', () => {});
    it('renders compact mode', () => {});
  });
});`

	file, err := Parse(context.Background(), []byte(source), "test.ts", "vitest")

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(file.Suites) != 1 {
		t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
	}

	suite := file.Suites[0]
	if len(suite.Tests) != 2 {
		t.Fatalf("len(suite.Tests) = %d, want 2", len(suite.Tests))
	}

	expectedTests := []string{
		"renders full mode (dynamic cases)",
		"renders compact mode (dynamic cases)",
	}

	for i, expected := range expectedTests {
		if suite.Tests[i].Name != expected {
			t.Errorf("Tests[%d].Name = %q, want %q", i, suite.Tests[i].Name, expected)
		}
	}
}

func TestParse_EachWithObjectArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		source    string
		wantCount int
		wantFirst string
	}{
		{
			name: "should parse it.each with object array as single dynamic test (ADR-02)",
			source: `it.each([
  { input: 1, expected: 2 },
  { input: 2, expected: 4 },
])('test $input', ({ input, expected }) => {});`,
			wantCount: 1,
			wantFirst: "test $input (dynamic cases)",
		},
		{
			name: "should parse describe.each with object array as single dynamic suite (ADR-02)",
			source: `describe.each([
  { name: 'Chrome' },
  { name: 'Firefox' },
])('Browser $name', () => {
  it('works', () => {});
});`,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := Parse(context.Background(), []byte(tt.source), "test.ts", "jest")

			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if len(file.Tests) != tt.wantCount {
				t.Fatalf("len(Tests) = %d, want %d", len(file.Tests), tt.wantCount)
			}

			if tt.wantCount > 0 && file.Tests[0].Name != tt.wantFirst {
				t.Errorf("Tests[0].Name = %q, want %q", file.Tests[0].Name, tt.wantFirst)
			}
		})
	}
}

func TestParse_MixedStaticAndDynamic(t *testing.T) {
	t.Parallel()

	source := "describe('Suite', () => {\n  it('static test', () => {});\n\n  [1, 2].forEach((n) => {\n    it(`dynamic ${n}`, () => {});\n  });\n});"

	file, err := Parse(context.Background(), []byte(source), "test.ts", "vitest")

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(file.Suites) != 1 {
		t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
	}

	suite := file.Suites[0]
	if len(suite.Tests) != 2 {
		t.Fatalf("len(suite.Tests) = %d, want 2", len(suite.Tests))
	}

	if suite.Tests[0].Name != "static test" {
		t.Errorf("Tests[0].Name = %q, want %q", suite.Tests[0].Name, "static test")
	}

	if suite.Tests[1].Name != "dynamic ${n} (dynamic cases)" {
		t.Errorf("Tests[1].Name = %q, want %q", suite.Tests[1].Name, "dynamic ${n} (dynamic cases)")
	}
}

func TestParse_ForEachWithDescribe(t *testing.T) {
	t.Parallel()

	// Simpler pattern: forEach → describe
	source := `items.forEach((item) => {
  describe('Suite', () => {
    it('test', () => {});
  });
});`

	file, err := Parse(context.Background(), []byte(source), "test.ts", "jest")

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Should detect 1 dynamic suite with 1 test inside
	if len(file.Suites) != 1 {
		t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
	}

	suite := file.Suites[0]
	if len(suite.Tests) != 1 {
		t.Errorf("len(suite.Tests) = %d, want 1", len(suite.Tests))
	}
}

func TestParse_ForEachWithConstBeforeIt(t *testing.T) {
	t.Parallel()

	// Bug case: forEach callback with const declaration before it()
	source := `items.forEach(item => {
  const name = 'test' + item;
  it(name, () => {});
});`

	file, err := Parse(context.Background(), []byte(source), "test.ts", "jest")

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Should detect 1 dynamic test
	if len(file.Tests) != 1 {
		t.Fatalf("len(Tests) = %d, want 1", len(file.Tests))
	}
}

func TestParse_NestedForEachWithDescribe(t *testing.T) {
	t.Parallel()

	// Pattern from react-testing-library/events.js:
	// forEach → describe → forEach → it
	// Note: inner forEach has a const declaration before it()
	source := `eventTypes.forEach(({type, events}) => {
  describe('Events', () => {
    events.forEach(eventName => {
      const propName = 'on' + eventName;
      it('triggers ' + propName, () => {});
    });
  });
});`

	file, err := Parse(context.Background(), []byte(source), "test.ts", "jest")

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Should detect 1 dynamic suite with 1 dynamic test inside
	if len(file.Suites) != 1 {
		t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
	}

	suite := file.Suites[0]
	if len(suite.Tests) != 1 {
		t.Errorf("len(suite.Tests) = %d, want 1", len(suite.Tests))
	}
}

func TestParse_ForLoop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		source    string
		wantCount int
		wantFirst string
	}{
		{
			name: "should parse for...of with test",
			source: `for (const item of items) {
  test('test item', () => {});
}`,
			wantCount: 1,
			wantFirst: "test item (dynamic cases)",
		},
		{
			name: "should parse for...in with it",
			source: `for (const key in obj) {
  it('test key', () => {});
}`,
			wantCount: 1,
			wantFirst: "test key (dynamic cases)",
		},
		{
			name: "should parse classic for loop",
			source: `for (let i = 0; i < 10; i++) {
  test('test ' + i, () => {});
}`,
			wantCount: 1,
			wantFirst: "(dynamic) (dynamic cases)",
		},
		{
			name: "should parse nested for loops",
			source: `for (const x of xs) {
  for (const y of ys) {
    test('test', () => {});
  }
}`,
			wantCount: 1,
			wantFirst: "test (dynamic cases)",
		},
		{
			name: "should parse while loop",
			source: `while (hasMore()) {
  test('dynamic test', () => {});
}`,
			wantCount: 1,
			wantFirst: "dynamic test (dynamic cases)",
		},
		{
			name: "should parse do-while loop",
			source: `do {
  it('iterative test', () => {});
} while (condition);`,
			wantCount: 1,
			wantFirst: "iterative test (dynamic cases)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := Parse(context.Background(), []byte(tt.source), "test.ts", "vitest")

			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if len(file.Tests) != tt.wantCount {
				t.Fatalf("len(Tests) = %d, want %d", len(file.Tests), tt.wantCount)
			}

			if tt.wantCount > 0 && file.Tests[0].Name != tt.wantFirst {
				t.Errorf("Tests[0].Name = %q, want %q", file.Tests[0].Name, tt.wantFirst)
			}
		})
	}
}

func TestParse_ForLoopInsideDescribe(t *testing.T) {
	t.Parallel()

	// Pattern from vite config.spec.ts
	source := `describe('loadConfigFromFile', () => {
  const cases = [
    { fileName: 'vite.config.js' },
    { fileName: 'vite.config.ts' },
  ];

  for (const { fileName } of cases) {
    for (const typeField of [undefined, 'module']) {
      test('load ' + fileName, async () => {});
    }
  }
});`

	file, err := Parse(context.Background(), []byte(source), "test.ts", "vitest")

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(file.Suites) != 1 {
		t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
	}

	suite := file.Suites[0]
	// Should detect 1 dynamic test (nested for loops count as 1)
	if len(suite.Tests) != 1 {
		t.Fatalf("len(suite.Tests) = %d, want 1", len(suite.Tests))
	}

	if suite.Tests[0].Name != "(dynamic) (dynamic cases)" {
		t.Errorf("Tests[0].Name = %q, want %q", suite.Tests[0].Name, "(dynamic) (dynamic cases)")
	}
}

func TestParse_ForLoopWithDescribe(t *testing.T) {
	t.Parallel()

	// Pattern: for loop containing describe
	source := `for (const version of versions) {
  describe('ES' + version, () => {
    test('should parse', () => {});
  });
}`

	file, err := Parse(context.Background(), []byte(source), "test.ts", "vitest")

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Should detect 1 dynamic suite
	if len(file.Suites) != 1 {
		t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
	}

	suite := file.Suites[0]
	if suite.Name != "(dynamic) (dynamic cases)" {
		t.Errorf("Suites[0].Name = %q, want %q", suite.Name, "(dynamic) (dynamic cases)")
	}

	// Test inside suite should be static (not dynamic)
	if len(suite.Tests) != 1 {
		t.Fatalf("len(suite.Tests) = %d, want 1", len(suite.Tests))
	}

	if suite.Tests[0].Name != "should parse" {
		t.Errorf("suite.Tests[0].Name = %q, want %q", suite.Tests[0].Name, "should parse")
	}
}

func TestParse_IIFEConditionalDescribe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		wantSuites int
		wantTests  int
	}{
		{
			name: "should parse IIFE with ternary describe",
			source: `;(process.env.SKIP ? describe.skip : describe)(
  'test suite',
  () => {
    it('should work', () => {})
  }
)`,
			wantSuites: 1,
			wantTests:  0,
		},
		{
			name: "should parse nested IIFE ternary describes",
			source: `;(condition1 ? describe.skip : describe)(
  'outer suite',
  () => {
    ;(condition2 ? describe.skip : describe)(
      'inner suite',
      () => {
        it('test', () => {})
      }
    )
  }
)`,
			wantSuites: 1,
			wantTests:  0,
		},
		{
			name: "should parse IIFE with ternary it",
			source: `;(process.env.SKIP ? it.skip : it)(
  'test case',
  () => {}
)`,
			wantSuites: 0,
			wantTests:  1,
		},
		{
			name: "should parse parenthesized describe without ternary",
			source: `(describe)(
  'simple suite',
  () => {
    it('test', () => {})
  }
)`,
			wantSuites: 1,
			wantTests:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := Parse(context.Background(), []byte(tt.source), "test.ts", "jest")

			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if len(file.Suites) != tt.wantSuites {
				t.Errorf("len(Suites) = %d, want %d", len(file.Suites), tt.wantSuites)
			}

			if len(file.Tests) != tt.wantTests {
				t.Errorf("len(Tests) = %d, want %d", len(file.Tests), tt.wantTests)
			}
		})
	}
}

func TestParse_IIFENestedTests(t *testing.T) {
	t.Parallel()

	source := `;(process.env.IS_TURBOPACK_TEST ? describe.skip : describe)(
  'build trace with extra entries',
  () => {
    ;(process.env.TURBOPACK_DEV ? describe.skip : describe)(
      'production mode',
      () => {
        it('should build and trace correctly', async () => {})
      }
    )
  }
)`

	file, err := Parse(context.Background(), []byte(source), "test.ts", "jest")

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(file.Suites) != 1 {
		t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
	}

	outerSuite := file.Suites[0]
	if outerSuite.Name != "build trace with extra entries" {
		t.Errorf("outer suite name = %q, want %q", outerSuite.Name, "build trace with extra entries")
	}

	if len(outerSuite.Suites) != 1 {
		t.Fatalf("len(outerSuite.Suites) = %d, want 1", len(outerSuite.Suites))
	}

	innerSuite := outerSuite.Suites[0]
	if innerSuite.Name != "production mode" {
		t.Errorf("inner suite name = %q, want %q", innerSuite.Name, "production mode")
	}

	if len(innerSuite.Tests) != 1 {
		t.Fatalf("len(innerSuite.Tests) = %d, want 1", len(innerSuite.Tests))
	}

	if innerSuite.Tests[0].Name != "should build and trace correctly" {
		t.Errorf("test name = %q, want %q", innerSuite.Tests[0].Name, "should build and trace correctly")
	}
}

func TestParse_CustomWrapperFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		wantSuites int
		wantTests  int
	}{
		{
			name: "should detect tests inside describeMatrix wrapper",
			source: `describeMatrix({ providers: { d1: true } }, 'D1', () => {
  test('should succeed', async () => {});
  test('should fail gracefully', async () => {});
});`,
			wantSuites: 0,
			wantTests:  2,
		},
		{
			name: "should detect tests inside custom wrapper with describe inside",
			source: `describeMatrix({ providers: sqliteOnly }, 'SQLite', () => {
  describe('introspection', () => {
    it('basic introspection', async () => {});
    it('introspection --force', async () => {});
  });
});`,
			wantSuites: 1,
			wantTests:  0,
		},
		{
			name: "should detect tests inside nested custom wrappers",
			source: `customWrapper('outer', () => {
  anotherWrapper('inner', () => {
    test('nested test', () => {});
  });
});`,
			wantSuites: 0,
			wantTests:  1,
		},
		{
			name: "should detect tests with multiple arguments before callback",
			source: `myTestHelper(config, options, 'name', () => {
  it('should work', () => {});
});`,
			wantSuites: 0,
			wantTests:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := Parse(context.Background(), []byte(tt.source), "test.ts", "jest")

			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if len(file.Suites) != tt.wantSuites {
				t.Errorf("len(Suites) = %d, want %d", len(file.Suites), tt.wantSuites)
			}

			if len(file.Tests) != tt.wantTests {
				t.Errorf("len(Tests) = %d, want %d", len(file.Tests), tt.wantTests)
			}
		})
	}
}

func TestParse_CustomWrapperWithDescribeInside(t *testing.T) {
	t.Parallel()

	source := `describeMatrix({ providers: sqliteOnly }, 'SQLite', () => {
  describe('introspection', () => {
    it('basic introspection', async () => {});
    it('introspection --force', async () => {});
  });

  it('should succeed when schema and db do match', async () => {});
});`

	file, err := Parse(context.Background(), []byte(source), "test.ts", "jest")

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// describeMatrix is not recognized as a suite, so describe inside becomes top-level
	if len(file.Suites) != 1 {
		t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
	}

	suite := file.Suites[0]
	if suite.Name != "introspection" {
		t.Errorf("Suites[0].Name = %q, want %q", suite.Name, "introspection")
	}

	if len(suite.Tests) != 2 {
		t.Errorf("len(suite.Tests) = %d, want 2", len(suite.Tests))
	}

	// The it() outside describe should be at file level
	if len(file.Tests) != 1 {
		t.Errorf("len(file.Tests) = %d, want 1", len(file.Tests))
	}
}

func TestParse_VariableDeclaration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		wantCount  int
		wantFirst  string
		wantStatus domain.TestStatus
	}{
		{
			name:       "should parse it inside var declaration",
			source:     `var runningTest = it("test name", function() {});`,
			wantCount:  1,
			wantFirst:  "test name",
			wantStatus: domain.TestStatusActive,
		},
		{
			name:       "should parse xit inside var declaration",
			source:     `var skippedTest = xit("pending test", function() {});`,
			wantCount:  1,
			wantFirst:  "pending test",
			wantStatus: domain.TestStatusSkipped,
		},
		{
			name:       "should parse it with method chain inside var declaration",
			source:     `var test = it("test", function() {}).timeout(1000);`,
			wantCount:  1,
			wantFirst:  "test",
			wantStatus: domain.TestStatusActive,
		},
		{
			name:       "should parse const declaration",
			source:     `const myTest = it("const test", () => {});`,
			wantCount:  1,
			wantFirst:  "const test",
			wantStatus: domain.TestStatusActive,
		},
		{
			name:       "should parse let declaration",
			source:     `let myTest = it("let test", () => {});`,
			wantCount:  1,
			wantFirst:  "let test",
			wantStatus: domain.TestStatusActive,
		},
		{
			name:       "should parse it.skip inside variable declaration",
			source:     `const skipped = it.skip("skipped test", () => {});`,
			wantCount:  1,
			wantFirst:  "skipped test",
			wantStatus: domain.TestStatusSkipped,
		},
		{
			name:       "should parse multiple chained methods",
			source:     `var test = it("chained", () => {}).timeout(1000).retries(3);`,
			wantCount:  1,
			wantFirst:  "chained",
			wantStatus: domain.TestStatusActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := Parse(context.Background(), []byte(tt.source), "test.js", "mocha")

			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if len(file.Tests) != tt.wantCount {
				t.Fatalf("len(Tests) = %d, want %d", len(file.Tests), tt.wantCount)
			}

			if tt.wantCount > 0 {
				if file.Tests[0].Name != tt.wantFirst {
					t.Errorf("Tests[0].Name = %q, want %q", file.Tests[0].Name, tt.wantFirst)
				}
				if file.Tests[0].Status != tt.wantStatus {
					t.Errorf("Tests[0].Status = %q, want %q", file.Tests[0].Status, tt.wantStatus)
				}
			}
		})
	}
}

func TestParse_VariableDeclarationInSuite(t *testing.T) {
	t.Parallel()

	source := `describe("setting timeout", function () {
  var runningTest =
    it("enables users to call timeout on active tests", function () {
      expect(1 + 1, "to be", 2);
    }).timeout(1003);

  var skippedTest =
    xit("enables users to call timeout on pending tests", function () {
      expect(1 + 1, "to be", 3);
    }).timeout(1002);

  it("sets timeout on pending tests", function () {
    expect(skippedTest._timeout, "to be", 1002);
  });
});`

	file, err := Parse(context.Background(), []byte(source), "test.js", "mocha")

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(file.Suites) != 1 {
		t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
	}

	suite := file.Suites[0]
	if len(suite.Tests) != 3 {
		t.Fatalf("len(suite.Tests) = %d, want 3", len(suite.Tests))
	}

	expectedTests := []struct {
		name   string
		status domain.TestStatus
	}{
		{"enables users to call timeout on active tests", domain.TestStatusActive},
		{"enables users to call timeout on pending tests", domain.TestStatusSkipped},
		{"sets timeout on pending tests", domain.TestStatusActive},
	}

	for i, expected := range expectedTests {
		if suite.Tests[i].Name != expected.name {
			t.Errorf("Tests[%d].Name = %q, want %q", i, suite.Tests[i].Name, expected.name)
		}
		if suite.Tests[i].Status != expected.status {
			t.Errorf("Tests[%d].Status = %q, want %q", i, suite.Tests[i].Status, expected.status)
		}
	}
}

func TestParse_RuleTester(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		source    string
		wantCount int
		wantFirst string
	}{
		{
			name: "should parse ruleTester.run as single dynamic test (ADR-02)",
			source: `const ruleTester = new RuleTester();
ruleTester.run('no-unused-vars', rule, {
  valid: [{ code: 'var x = 1; console.log(x);' }],
  invalid: [{ code: 'var x = 1;', errors: 1 }],
});`,
			wantCount: 1,
			wantFirst: "no-unused-vars (dynamic cases)",
		},
		{
			name: "should parse tester.run as single dynamic test",
			source: `const tester = new RuleTester({ parser: '@typescript-eslint/parser' });
tester.run('rule-name', rule, {
  valid: [],
  invalid: [],
});`,
			wantCount: 1,
			wantFirst: "rule-name (dynamic cases)",
		},
		{
			name: "should parse stylelintTester.run",
			source: `const stylelintTester = getTestRule();
stylelintTester.run('block-no-empty', rule, {
  accept: [{ code: 'a { color: red; }' }],
  reject: [{ code: 'a {}' }],
});`,
			wantCount: 1,
			wantFirst: "block-no-empty (dynamic cases)",
		},
		{
			name: "should parse multiple ruleTester.run calls",
			source: `const ruleTester = new RuleTester();
ruleTester.run('rule-a', ruleA, { valid: [], invalid: [] });
ruleTester.run('rule-b', ruleB, { valid: [], invalid: [] });`,
			wantCount: 2,
			wantFirst: "rule-a (dynamic cases)",
		},
		{
			name:      "should not match non-tester .run() calls",
			source:    `server.run('start', config, {});`,
			wantCount: 0,
		},
		{
			name:      "should not match tester.run without string first arg",
			source:    `tester.run(ruleName, rule, {});`,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := Parse(context.Background(), []byte(tt.source), "test.ts", "jest")

			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if len(file.Tests) != tt.wantCount {
				t.Fatalf("len(Tests) = %d, want %d", len(file.Tests), tt.wantCount)
			}

			if tt.wantCount > 0 && file.Tests[0].Name != tt.wantFirst {
				t.Errorf("Tests[0].Name = %q, want %q", file.Tests[0].Name, tt.wantFirst)
			}
		})
	}
}

func TestParse_RuleTesterInsideDescribe(t *testing.T) {
	t.Parallel()

	source := `describe('ESLint Rules', () => {
  const ruleTester = new RuleTester();

  ruleTester.run('no-console', rule, {
    valid: [{ code: 'var x = 1;' }],
    invalid: [{ code: 'console.log(1);', errors: 1 }],
  });
});`

	file, err := Parse(context.Background(), []byte(source), "test.ts", "jest")

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(file.Suites) != 1 {
		t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
	}

	suite := file.Suites[0]
	if len(suite.Tests) != 1 {
		t.Fatalf("len(suite.Tests) = %d, want 1", len(suite.Tests))
	}

	if suite.Tests[0].Name != "no-console (dynamic cases)" {
		t.Errorf("Tests[0].Name = %q, want %q", suite.Tests[0].Name, "no-console (dynamic cases)")
	}
}
