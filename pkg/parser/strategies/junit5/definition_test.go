package junit5

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
)

func TestNewDefinition(t *testing.T) {
	def := NewDefinition()

	if def.Name != "junit5" {
		t.Errorf("expected Name='junit5', got '%s'", def.Name)
	}
	if def.Priority != framework.PriorityGeneric {
		t.Errorf("expected Priority=%d, got %d", framework.PriorityGeneric, def.Priority)
	}
	if len(def.Languages) != 1 || def.Languages[0] != domain.LanguageJava {
		t.Errorf("expected Languages=[java], got %v", def.Languages)
	}
	if def.Parser == nil {
		t.Error("expected Parser to be non-nil")
	}
	if len(def.Matchers) != 3 {
		t.Errorf("expected 3 Matchers, got %d", len(def.Matchers))
	}
}

func TestJUnit5FileMatcher_Match(t *testing.T) {
	matcher := &JUnit5FileMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		filename           string
		expectedConfidence int
	}{
		{"Test suffix", "CalculatorTest.java", 20},
		{"Tests suffix", "CalculatorTests.java", 20},
		{"Test prefix", "TestCalculator.java", 20},
		{"Test suffix with path", "src/test/java/com/example/UserServiceTest.java", 20},
		{"regular java file", "Calculator.java", 0},
		{"non-java file", "CalculatorTest.py", 0},
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

func TestJUnit5ContentMatcher_Match(t *testing.T) {
	matcher := &JUnit5ContentMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		content            string
		expectedConfidence int
	}{
		{
			name: "@Test annotation",
			content: `
import org.junit.jupiter.api.Test;

class CalculatorTest {
    @Test
    void testAdd() {
        assertEquals(3, 1 + 2);
    }
}
`,
			expectedConfidence: 40,
		},
		{
			name: "@ParameterizedTest annotation",
			content: `
import org.junit.jupiter.params.ParameterizedTest;

class CalculatorTest {
    @ParameterizedTest
    @ValueSource(ints = {1, 2, 3})
    void testNumbers(int n) {}
}
`,
			expectedConfidence: 40,
		},
		{
			name: "@Nested annotation",
			content: `
import org.junit.jupiter.api.Nested;

class CalculatorTest {
    @Nested
    class AdditionTests {}
}
`,
			expectedConfidence: 40,
		},
		{
			name: "@DisplayName annotation",
			content: `
import org.junit.jupiter.api.DisplayName;

@DisplayName("Calculator Tests")
class CalculatorTest {}
`,
			expectedConfidence: 40,
		},
		{
			name: "JUnit Jupiter import",
			content: `
import org.junit.jupiter.api.Assertions;

class SomeClass {}
`,
			expectedConfidence: 40,
		},
		{
			name: "no JUnit 5 patterns",
			content: `
public class Calculator {
    public int add(int a, int b) {
        return a + b;
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

func TestJUnit5Parser_Parse(t *testing.T) {
	p := &JUnit5Parser{}
	ctx := context.Background()

	t.Run("basic test methods", func(t *testing.T) {
		source := `
package com.example;

import org.junit.jupiter.api.Test;

class CalculatorTest {
    @Test
    void testAdd() {
        assertEquals(3, 1 + 2);
    }

    @Test
    void testSubtract() {
        assertEquals(1, 3 - 2);
    }

    void helperMethod() {
        // not a test
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "CalculatorTest.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if testFile.Path != "CalculatorTest.java" {
			t.Errorf("expected Path='CalculatorTest.java', got '%s'", testFile.Path)
		}
		if testFile.Framework != "junit5" {
			t.Errorf("expected Framework='junit5', got '%s'", testFile.Framework)
		}
		if testFile.Language != domain.LanguageJava {
			t.Errorf("expected Language=java, got '%s'", testFile.Language)
		}
		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if suite.Name != "CalculatorTest" {
			t.Errorf("expected Suite.Name='CalculatorTest', got '%s'", suite.Name)
		}
		if len(suite.Tests) != 2 {
			t.Fatalf("expected 2 Tests in suite, got %d", len(suite.Tests))
		}
		if suite.Tests[0].Name != "testAdd" {
			t.Errorf("expected Tests[0].Name='testAdd', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[1].Name != "testSubtract" {
			t.Errorf("expected Tests[1].Name='testSubtract', got '%s'", suite.Tests[1].Name)
		}
	})

	t.Run("@Disabled annotation", func(t *testing.T) {
		source := `
package com.example;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.Disabled;

class SkippedTest {
    @Test
    @Disabled("not implemented")
    void testSkipped() {}

    @Test
    void testNormal() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "SkippedTest.java")
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

		if suite.Tests[0].Name != "testSkipped" {
			t.Errorf("expected Tests[0].Name='testSkipped', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[0].Status != domain.TestStatusSkipped {
			t.Errorf("expected Tests[0].Status='skipped', got '%s'", suite.Tests[0].Status)
		}

		if suite.Tests[1].Name != "testNormal" {
			t.Errorf("expected Tests[1].Name='testNormal', got '%s'", suite.Tests[1].Name)
		}
		if suite.Tests[1].Status != domain.TestStatusActive {
			t.Errorf("expected Tests[1].Status='active', got '%s'", suite.Tests[1].Status)
		}
	})

	t.Run("@DisplayName annotation", func(t *testing.T) {
		source := `
package com.example;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

class DisplayNameTest {
    @Test
    @DisplayName("Addition should work correctly")
    void testAdd() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "DisplayNameTest.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if len(suite.Tests) != 1 {
			t.Fatalf("expected 1 Test, got %d", len(suite.Tests))
		}

		if suite.Tests[0].Name != "Addition should work correctly" {
			t.Errorf("expected Name='Addition should work correctly', got '%s'", suite.Tests[0].Name)
		}
	})

	t.Run("@ParameterizedTest annotation", func(t *testing.T) {
		source := `
package com.example;

import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.ValueSource;

class ParameterizedTests {
    @ParameterizedTest
    @ValueSource(ints = {1, 2, 3})
    void testWithValue(int value) {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "ParameterizedTests.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if len(suite.Tests) != 1 {
			t.Fatalf("expected 1 Test, got %d", len(suite.Tests))
		}

		if suite.Tests[0].Name != "testWithValue" {
			t.Errorf("expected Name='testWithValue', got '%s'", suite.Tests[0].Name)
		}
	})

	t.Run("@Nested classes", func(t *testing.T) {
		source := `
package com.example;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.Nested;

class OuterTest {
    @Test
    void outerTest() {}

    @Nested
    class InnerTest {
        @Test
        void innerTest() {}
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "OuterTest.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if suite.Name != "OuterTest" {
			t.Errorf("expected Suite.Name='OuterTest', got '%s'", suite.Name)
		}
		if len(suite.Tests) != 1 {
			t.Fatalf("expected 1 Test in outer suite, got %d", len(suite.Tests))
		}
		if suite.Tests[0].Name != "outerTest" {
			t.Errorf("expected outer test name='outerTest', got '%s'", suite.Tests[0].Name)
		}

		if len(suite.Suites) != 1 {
			t.Fatalf("expected 1 nested Suite, got %d", len(suite.Suites))
		}

		nested := suite.Suites[0]
		if nested.Name != "InnerTest" {
			t.Errorf("expected nested Suite.Name='InnerTest', got '%s'", nested.Name)
		}
		if len(nested.Tests) != 1 {
			t.Fatalf("expected 1 Test in nested suite, got %d", len(nested.Tests))
		}
		if nested.Tests[0].Name != "innerTest" {
			t.Errorf("expected nested test name='innerTest', got '%s'", nested.Tests[0].Name)
		}
	})

	t.Run("@Disabled on class", func(t *testing.T) {
		source := `
package com.example;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.Disabled;

@Disabled("class disabled")
class DisabledClassTest {
    @Test
    void testOne() {}

    @Test
    void testTwo() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "DisabledClassTest.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if suite.Status != domain.TestStatusSkipped {
			t.Errorf("expected Suite.Status='skipped', got '%s'", suite.Status)
		}

		// Methods inherit class status
		if len(suite.Tests) != 2 {
			t.Fatalf("expected 2 Tests, got %d", len(suite.Tests))
		}
		if suite.Tests[0].Status != domain.TestStatusSkipped {
			t.Errorf("expected Tests[0].Status='skipped', got '%s'", suite.Tests[0].Status)
		}
		if suite.Tests[1].Status != domain.TestStatusSkipped {
			t.Errorf("expected Tests[1].Status='skipped', got '%s'", suite.Tests[1].Status)
		}
	})

	t.Run("@RepeatedTest annotation", func(t *testing.T) {
		source := `
package com.example;

import org.junit.jupiter.api.RepeatedTest;

class RepeatedTests {
    @RepeatedTest(3)
    void testRepeated() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "RepeatedTests.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if len(suite.Tests) != 1 {
			t.Fatalf("expected 1 Test, got %d", len(suite.Tests))
		}

		if suite.Tests[0].Name != "testRepeated" {
			t.Errorf("expected Name='testRepeated', got '%s'", suite.Tests[0].Name)
		}
	})
}
