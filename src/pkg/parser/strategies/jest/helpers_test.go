package jest

import (
	"testing"

	"github.com/specvital/core/domain"
)

func TestParseFunctionName(t *testing.T) {
	t.Parallel()

	// parseFunctionName은 내부 함수이므로 parse를 통해 간접 테스트
	tests := []struct {
		name       string
		source     string
		wantStatus domain.TestStatus
	}{
		{
			name:       "should parse describe.skip as skipped",
			source:     `describe.skip('Suite', () => {});`,
			wantStatus: domain.TestStatusSkipped,
		},
		{
			name:       "should parse describe.only as only",
			source:     `describe.only('Suite', () => {});`,
			wantStatus: domain.TestStatusOnly,
		},
		{
			name:       "should parse xdescribe as skipped",
			source:     `xdescribe('Suite', () => {});`,
			wantStatus: domain.TestStatusSkipped,
		},
		{
			name:       "should parse fdescribe as only",
			source:     `fdescribe('Suite', () => {});`,
			wantStatus: domain.TestStatusOnly,
		},
		{
			name:       "should parse it.skip as skipped",
			source:     `it.skip('test', () => {});`,
			wantStatus: domain.TestStatusSkipped,
		},
		{
			name:       "should parse it.only as only",
			source:     `it.only('test', () => {});`,
			wantStatus: domain.TestStatusOnly,
		},
		{
			name:       "should parse xit as skipped",
			source:     `xit('test', () => {});`,
			wantStatus: domain.TestStatusSkipped,
		},
		{
			name:       "should parse fit as only",
			source:     `fit('test', () => {});`,
			wantStatus: domain.TestStatusOnly,
		},
		{
			name:       "should parse test.todo as skipped",
			source:     `test.todo('test');`,
			wantStatus: domain.TestStatusSkipped,
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

			var status domain.TestStatus
			if len(file.Suites) > 0 {
				status = file.Suites[0].Status
			} else if len(file.Tests) > 0 {
				status = file.Tests[0].Status
			} else {
				t.Fatal("no suites or tests found")
			}

			if status != tt.wantStatus {
				t.Errorf("status = %q, want %q", status, tt.wantStatus)
			}
		})
	}
}

func TestUnquoteString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "should unquote double quotes",
			input: `"hello"`,
			want:  "hello",
		},
		{
			name:  "should unquote single quotes",
			input: `'hello'`,
			want:  "hello",
		},
		{
			name:  "should unquote backticks",
			input: "`hello`",
			want:  "hello",
		},
		{
			name:  "should return short string as-is",
			input: "a",
			want:  "a",
		},
		{
			name:  "should return unquoted string as-is",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "should handle mismatched quotes",
			input: `"hello'`,
			want:  `"hello'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// When
			got := unquoteString(tt.input)

			// Then
			if got != tt.want {
				t.Errorf("unquoteString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractTestName(t *testing.T) {
	t.Parallel()

	// extractTestName은 내부 함수이므로 parse를 통해 간접 테스트
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "should extract single-quoted name",
			source: `it('single quoted', () => {});`,
			want:   "single quoted",
		},
		{
			name:   "should extract double-quoted name",
			source: `it("double quoted", () => {});`,
			want:   "double quoted",
		},
		{
			name:   "should extract template string name",
			source: "it(`template string`, () => {});",
			want:   "template string",
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
			if len(file.Tests) != 1 {
				t.Fatalf("len(Tests) = %d, want 1", len(file.Tests))
			}
			if file.Tests[0].Name != tt.want {
				t.Errorf("Name = %q, want %q", file.Tests[0].Name, tt.want)
			}
		})
	}
}

func TestFindCallback(t *testing.T) {
	t.Parallel()

	// findCallback은 내부 함수이므로 parse를 통해 간접 테스트
	tests := []struct {
		name       string
		source     string
		wantSuites int
		wantTests  int
	}{
		{
			name:       "should find arrow function callback",
			source:     `describe('Suite', () => { it('test', () => {}); });`,
			wantSuites: 1,
			wantTests:  0,
		},
		{
			name:       "should find function expression callback",
			source:     `describe('Suite', function() { it('test', function() {}); });`,
			wantSuites: 1,
			wantTests:  0,
		},
		{
			name:       "should ignore describe without callback",
			source:     `describe('NoCallback');`,
			wantSuites: 0,
			wantTests:  0,
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
		})
	}
}

func TestExtractEachTestCases(t *testing.T) {
	t.Parallel()

	// extractEachTestCases는 내부 함수이므로 parse를 통해 간접 테스트
	tests := []struct {
		name      string
		source    string
		wantCount int
		wantFirst string
	}{
		{
			name:      "should extract array of arrays",
			source:    `describe('S', () => { it.each([[1], [2], [3]])('test %d', () => {}); });`,
			wantCount: 3,
			wantFirst: "test 1",
		},
		{
			name:      "should extract string values",
			source:    `describe('S', () => { it.each(['foo', 'bar'])('test %s', () => {}); });`,
			wantCount: 2,
			wantFirst: "test 'foo'",
		},
		{
			name:      "should extract number values",
			source:    `describe('S', () => { it.each([1, 2])('test %d', () => {}); });`,
			wantCount: 2,
			wantFirst: "test 1",
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
			if len(file.Suites) != 1 {
				t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
			}
			if len(file.Suites[0].Tests) != tt.wantCount {
				t.Fatalf("len(Tests) = %d, want %d", len(file.Suites[0].Tests), tt.wantCount)
			}
			if file.Suites[0].Tests[0].Name != tt.wantFirst {
				t.Errorf("Tests[0].Name = %q, want %q", file.Suites[0].Tests[0].Name, tt.wantFirst)
			}
		})
	}
}

func TestFormatEachName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		template string
		data     string
		want     string
	}{
		{
			name:     "should replace %s placeholder",
			template: "test %s",
			data:     "value",
			want:     "test value",
		},
		{
			name:     "should replace %d placeholder",
			template: "test %d",
			data:     "123",
			want:     "test 123",
		},
		{
			name:     "should replace %p placeholder",
			template: "test %p",
			data:     "data",
			want:     "test data",
		},
		{
			name:     "should replace first placeholder only",
			template: "test %s %s",
			data:     "first",
			want:     "test first %s",
		},
		{
			name:     "should return template if no placeholder",
			template: "no placeholder",
			data:     "data",
			want:     "no placeholder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// When
			got := formatEachName(tt.template, tt.data)

			// Then
			if got != tt.want {
				t.Errorf("formatEachName(%q, %q) = %q, want %q", tt.template, tt.data, got, tt.want)
			}
		})
	}
}
