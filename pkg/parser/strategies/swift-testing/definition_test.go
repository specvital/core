package swifttesting

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
)

func TestNewDefinition(t *testing.T) {
	def := NewDefinition()

	if def.Name != "swift-testing" {
		t.Errorf("expected Name='swift-testing', got '%s'", def.Name)
	}
	if def.Priority != framework.PrioritySpecialized {
		t.Errorf("expected Priority=%d, got %d", framework.PrioritySpecialized, def.Priority)
	}
	if len(def.Languages) != 1 || def.Languages[0] != domain.LanguageSwift {
		t.Errorf("expected Languages=[swift], got %v", def.Languages)
	}
	if def.Parser == nil {
		t.Error("expected Parser to be non-nil")
	}
	if len(def.Matchers) != 3 {
		t.Errorf("expected 3 Matchers, got %d", len(def.Matchers))
	}
}

func TestSwiftTestingFileMatcher_Match(t *testing.T) {
	matcher := &SwiftTestingFileMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		filename           string
		expectedConfidence int
	}{
		{"Test suffix", "CalculatorTest.swift", 20},
		{"Tests suffix", "CalculatorTests.swift", 20},
		{"Test suffix with path", "Tests/AppTests/CalculatorTests.swift", 20},
		{"regular swift file", "Calculator.swift", 0},
		{"non-swift file", "CalculatorTest.java", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signal := framework.Signal{
				Type:  framework.SignalFileName,
				Value: tt.filename,
			}

			result := matcher.Match(ctx, signal)

			if result.Confidence != tt.expectedConfidence {
				t.Errorf("expected Confidence=%d, got %d", tt.expectedConfidence, result.Confidence)
			}
		})
	}
}

func TestSwiftTestingContentMatcher_Match(t *testing.T) {
	matcher := &SwiftTestingContentMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		content            string
		expectedConfidence int
	}{
		{
			name: "@Test attribute",
			content: `
import Testing

@Suite
struct CalculatorTests {
    @Test
    func addition() {
        #expect(1 + 2 == 3)
    }
}
`,
			expectedConfidence: 40,
		},
		{
			name: "@Suite attribute only",
			content: `
import Testing

@Suite
struct SomeTests {
}
`,
			expectedConfidence: 40,
		},
		{
			name: "#expect macro",
			content: `
#expect(result == expected)
`,
			expectedConfidence: 40,
		},
		{
			name: "#require macro",
			content: `
let value = try #require(optionalValue)
`,
			expectedConfidence: 40,
		},
		{
			name: "Testing import",
			content: `
import Testing
import Foundation
`,
			expectedConfidence: 40,
		},
		{
			name:               "no Swift Testing patterns",
			content:            `class Calculator { func add() {} }`,
			expectedConfidence: 0,
		},
		{
			name: "XCTest patterns should not match",
			content: `
import XCTest

class CalculatorTests: XCTestCase {
    func testAdd() {
        XCTAssertEqual(3, 1 + 2)
    }
}
`,
			expectedConfidence: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signal := framework.Signal{
				Type:    framework.SignalFileContent,
				Value:   tt.content,
				Context: []byte(tt.content),
			}

			result := matcher.Match(ctx, signal)

			if result.Confidence != tt.expectedConfidence {
				t.Errorf("expected Confidence=%d, got %d", tt.expectedConfidence, result.Confidence)
			}
		})
	}
}

func TestSwiftTestingParser_Parse(t *testing.T) {
	p := &SwiftTestingParser{}
	ctx := context.Background()

	t.Run("basic @Test functions in @Suite struct", func(t *testing.T) {
		source := `
import Testing

@Suite
struct CalculatorTests {
    @Test
    func addition() {
        #expect(1 + 2 == 3)
    }

    @Test
    func subtraction() {
        #expect(3 - 2 == 1)
    }

    func helperMethod() {
        // not a test
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "CalculatorTests.swift")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if testFile.Path != "CalculatorTests.swift" {
			t.Errorf("expected Path='CalculatorTests.swift', got '%s'", testFile.Path)
		}
		if testFile.Framework != "swift-testing" {
			t.Errorf("expected Framework='swift-testing', got '%s'", testFile.Framework)
		}
		if testFile.Language != domain.LanguageSwift {
			t.Errorf("expected Language=swift, got '%s'", testFile.Language)
		}
		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if suite.Name != "CalculatorTests" {
			t.Errorf("expected Suite.Name='CalculatorTests', got '%s'", suite.Name)
		}
		if len(suite.Tests) != 2 {
			t.Fatalf("expected 2 Tests in suite, got %d", len(suite.Tests))
		}
		if suite.Tests[0].Name != "addition" {
			t.Errorf("expected Tests[0].Name='addition', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[1].Name != "subtraction" {
			t.Errorf("expected Tests[1].Name='subtraction', got '%s'", suite.Tests[1].Name)
		}
	})

	t.Run("async throws test functions", func(t *testing.T) {
		source := `
import Testing

@Suite
struct AsyncTests {
    @Test
    func asyncTest() async throws {
        let result = await fetchData()
        #expect(result != nil)
    }

    @Test
    func syncTest() {
        #expect(true)
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "AsyncTests.swift")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if len(suite.Tests) != 2 {
			t.Fatalf("expected 2 Tests, got %d", len(suite.Tests))
		}

		if suite.Tests[0].Name != "asyncTest" {
			t.Errorf("expected Tests[0].Name='asyncTest', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[0].Modifier != "async" {
			t.Errorf("expected Tests[0].Modifier='async', got '%s'", suite.Tests[0].Modifier)
		}
	})

	t.Run("multiple @Suite structs", func(t *testing.T) {
		source := `
import Testing

@Suite
struct FirstTests {
    @Test
    func first() {
        #expect(true)
    }
}

@Suite
struct SecondTests {
    @Test
    func second() {
        #expect(true)
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "MultipleTests.swift")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 2 {
			t.Fatalf("expected 2 Suites, got %d", len(testFile.Suites))
		}

		if testFile.Suites[0].Name != "FirstTests" {
			t.Errorf("expected Suites[0].Name='FirstTests', got '%s'", testFile.Suites[0].Name)
		}
		if testFile.Suites[1].Name != "SecondTests" {
			t.Errorf("expected Suites[1].Name='SecondTests', got '%s'", testFile.Suites[1].Name)
		}
	})

	t.Run("struct without @Suite but with @Test functions", func(t *testing.T) {
		source := `
import Testing

struct ImplicitTests {
    @Test
    func implicitTest() {
        #expect(true)
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "ImplicitTests.swift")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if suite.Name != "ImplicitTests" {
			t.Errorf("expected Suite.Name='ImplicitTests', got '%s'", suite.Name)
		}
		if len(suite.Tests) != 1 {
			t.Fatalf("expected 1 Test, got %d", len(suite.Tests))
		}
	})

	t.Run("struct without @Test functions ignored", func(t *testing.T) {
		source := `
import Testing

struct NotATestStruct {
    func someFunction() {
        // no @Test attribute
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "NotATest.swift")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 0 {
			t.Errorf("expected 0 Suites, got %d", len(testFile.Suites))
		}
	})

	t.Run("@Test with arguments (parameterized)", func(t *testing.T) {
		source := `
import Testing

@Suite
struct ParameterizedTests {
    @Test(arguments: [1, 2, 3])
    func parameterized(value: Int) {
        #expect(value > 0)
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "ParameterizedTests.swift")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if len(suite.Tests) != 1 {
			t.Fatalf("expected 1 Test (parameterized counts as 1), got %d", len(suite.Tests))
		}
		if suite.Tests[0].Name != "parameterized" {
			t.Errorf("expected Tests[0].Name='parameterized', got '%s'", suite.Tests[0].Name)
		}
	})

	t.Run("@Test(.disabled) handling", func(t *testing.T) {
		source := `
import Testing

@Suite
struct DisabledTests {
    @Test(.disabled("not implemented"))
    func disabledTest() {
        #expect(false)
    }

    @Test
    func activeTest() {
        #expect(true)
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "DisabledTests.swift")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if len(suite.Tests) != 2 {
			t.Fatalf("expected 2 Tests, got %d", len(suite.Tests))
		}

		if suite.Tests[0].Status != domain.TestStatusSkipped {
			t.Errorf("expected Tests[0].Status='skipped', got '%s'", suite.Tests[0].Status)
		}
		if suite.Tests[1].Status != domain.TestStatusActive {
			t.Errorf("expected Tests[1].Status='active', got '%s'", suite.Tests[1].Status)
		}
	})
}
