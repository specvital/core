package jest

import (
	"testing"

	"github.com/specvital/core/domain"
)

func TestAddSuiteToTarget(t *testing.T) {
	t.Parallel()

	t.Run("should add suite to file when no parent suite", func(t *testing.T) {
		t.Parallel()

		// Given
		file := &domain.TestFile{}
		suite := domain.TestSuite{Name: "test"}

		// When
		addSuiteToTarget(suite, nil, file)

		// Then
		if len(file.Suites) != 1 {
			t.Fatalf("len(file.Suites) = %d, want 1", len(file.Suites))
		}
		if file.Suites[0].Name != "test" {
			t.Errorf("Suites[0].Name = %q, want %q", file.Suites[0].Name, "test")
		}
	})

	t.Run("should add suite to parent when parent exists", func(t *testing.T) {
		t.Parallel()

		// Given
		file := &domain.TestFile{}
		parent := &domain.TestSuite{Name: "parent"}
		suite := domain.TestSuite{Name: "child"}

		// When
		addSuiteToTarget(suite, parent, file)

		// Then
		if len(file.Suites) != 0 {
			t.Error("file.Suites should be empty")
		}
		if len(parent.Suites) != 1 {
			t.Fatalf("len(parent.Suites) = %d, want 1", len(parent.Suites))
		}
		if parent.Suites[0].Name != "child" {
			t.Errorf("parent.Suites[0].Name = %q, want %q", parent.Suites[0].Name, "child")
		}
	})
}

func TestAddTestToTarget(t *testing.T) {
	t.Parallel()

	t.Run("should add test to file when no parent suite", func(t *testing.T) {
		t.Parallel()

		// Given
		file := &domain.TestFile{}
		test := domain.Test{Name: "test"}

		// When
		addTestToTarget(test, nil, file)

		// Then
		if len(file.Tests) != 1 {
			t.Fatalf("len(file.Tests) = %d, want 1", len(file.Tests))
		}
		if file.Tests[0].Name != "test" {
			t.Errorf("Tests[0].Name = %q, want %q", file.Tests[0].Name, "test")
		}
	})

	t.Run("should add test to parent when parent exists", func(t *testing.T) {
		t.Parallel()

		// Given
		file := &domain.TestFile{}
		parent := &domain.TestSuite{Name: "parent"}
		test := domain.Test{Name: "test"}

		// When
		addTestToTarget(test, parent, file)

		// Then
		if len(file.Tests) != 0 {
			t.Error("file.Tests should be empty")
		}
		if len(parent.Tests) != 1 {
			t.Fatalf("len(parent.Tests) = %d, want 1", len(parent.Tests))
		}
		if parent.Tests[0].Name != "test" {
			t.Errorf("parent.Tests[0].Name = %q, want %q", parent.Tests[0].Name, "test")
		}
	})
}

func TestProcessSuite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		wantSuites int
		wantStatus domain.TestStatus
	}{
		{
			name:       "should process describe with pending status",
			source:     `describe('Suite', () => {});`,
			wantSuites: 1,
			wantStatus: domain.TestStatusPending,
		},
		{
			name:       "should process describe.skip with skipped status",
			source:     `describe.skip('Suite', () => {});`,
			wantSuites: 1,
			wantStatus: domain.TestStatusSkipped,
		},
		{
			name:       "should process describe.only with only status",
			source:     `describe.only('Suite', () => {});`,
			wantSuites: 1,
			wantStatus: domain.TestStatusOnly,
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
				t.Fatalf("len(Suites) = %d, want %d", len(file.Suites), tt.wantSuites)
			}
			if file.Suites[0].Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", file.Suites[0].Status, tt.wantStatus)
			}
		})
	}
}

func TestProcessTest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		wantTests  int
		wantStatus domain.TestStatus
	}{
		{
			name:       "should process it with pending status",
			source:     `it('test', () => {});`,
			wantTests:  1,
			wantStatus: domain.TestStatusPending,
		},
		{
			name:       "should process it.skip with skipped status",
			source:     `it.skip('test', () => {});`,
			wantTests:  1,
			wantStatus: domain.TestStatusSkipped,
		},
		{
			name:       "should process it.only with only status",
			source:     `it.only('test', () => {});`,
			wantTests:  1,
			wantStatus: domain.TestStatusOnly,
		},
		{
			name:       "should process test with pending status",
			source:     `test('test', () => {});`,
			wantTests:  1,
			wantStatus: domain.TestStatusPending,
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
			if len(file.Tests) != tt.wantTests {
				t.Fatalf("len(Tests) = %d, want %d", len(file.Tests), tt.wantTests)
			}
			if file.Tests[0].Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", file.Tests[0].Status, tt.wantStatus)
			}
		})
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
			name:      "should resolve names with placeholders",
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
			name:      "should handle single case",
			template:  "case %d",
			testCases: []string{"1"},
			want:      []string{"case 1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// When
			got := resolveEachNames(tt.template, tt.testCases)

			// Then
			if len(got) != len(tt.want) {
				t.Fatalf("len(resolveEachNames()) = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("resolveEachNames()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestProcessEachSuites(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		wantSuites int
		wantName   string
	}{
		{
			name:       "should create suites for each case",
			source:     `describe.each([['a'], ['b']])('case %s', () => {});`,
			wantSuites: 2,
			wantName:   "case a",
		},
		{
			name:       "should handle number cases",
			source:     `describe.each([[1], [2], [3]])('num %d', () => {});`,
			wantSuites: 3,
			wantName:   "num 1",
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
				t.Fatalf("len(Suites) = %d, want %d", len(file.Suites), tt.wantSuites)
			}
			if file.Suites[0].Name != tt.wantName {
				t.Errorf("Suites[0].Name = %q, want %q", file.Suites[0].Name, tt.wantName)
			}
		})
	}
}

func TestProcessEachTests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		source    string
		wantTests int
		wantName  string
	}{
		{
			name:      "should create tests for each case",
			source:    `describe('S', () => { it.each([[1], [2]])('val %d', () => {}); });`,
			wantTests: 2,
			wantName:  "val 1",
		},
		{
			name:      "should handle test.each",
			source:    `describe('S', () => { test.each([['a']])('str %s', () => {}); });`,
			wantTests: 1,
			wantName:  "str a",
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
			if len(file.Suites[0].Tests) != tt.wantTests {
				t.Fatalf("len(Tests) = %d, want %d", len(file.Suites[0].Tests), tt.wantTests)
			}
			if file.Suites[0].Tests[0].Name != tt.wantName {
				t.Errorf("Tests[0].Name = %q, want %q", file.Suites[0].Tests[0].Name, tt.wantName)
			}
		})
	}
}
