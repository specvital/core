package junit5

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
	"github.com/specvital/core/pkg/parser/strategies/shared/javaast"
)

func TestNewDefinition(t *testing.T) {
	def := NewDefinition()

	if def.Name != "junit5" {
		t.Errorf("expected Name='junit5', got '%s'", def.Name)
	}
	if def.Priority != framework.PriorityGeneric {
		t.Errorf("expected Priority=%d, got %d", framework.PriorityGeneric, def.Priority)
	}
	if len(def.Languages) != 2 {
		t.Errorf("expected Languages=[java, kotlin], got %v", def.Languages)
	}
	if def.Languages[0] != domain.LanguageJava || def.Languages[1] != domain.LanguageKotlin {
		t.Errorf("expected Languages=[java, kotlin], got %v", def.Languages)
	}
	if def.Parser == nil {
		t.Error("expected Parser to be non-nil")
	}
	if len(def.Matchers) != 4 {
		t.Errorf("expected 4 Matchers, got %d", len(def.Matchers))
	}
}

func TestJavaTestFileMatcher_Match(t *testing.T) {
	matcher := &javaast.JavaTestFileMatcher{}
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
		{"src/main should be excluded", "src/main/java/com/example/CartesianProductTest.java", 0},
		{"src/main/kotlin should be excluded", "src/main/kotlin/com/example/TestFactory.java", 0},
		{"nested src/main should be excluded", "project/src/main/java/SomeTest.java", 0},
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
			name: "@TestFactory annotation",
			content: `
import org.junit.jupiter.api.TestFactory;

class DynamicTests {
    @TestFactory
    Stream<DynamicTest> dynamicTests() {}
}
`,
			expectedConfidence: 40,
		},
		{
			name: "@TestTemplate annotation",
			content: `
import org.junit.jupiter.api.TestTemplate;

class TemplateTests {
    @TestTemplate
    void templateTest() {}
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
		{
			name: "JUnit 4 file should not match (org.junit.Test import)",
			content: `
import org.junit.Test;
import static org.junit.Assert.assertEquals;

public class JUnit4Test {
    @Test
    public void test() {
        assertEquals(3, 1 + 2);
    }
}
`,
			expectedConfidence: 0,
		},
		{
			name: "JUnit 4 file with Assert import should not match",
			content: `
import org.junit.Assert;

public class JUnit4AssertTest {
    @Test
    public void test() {
        Assert.assertEquals(3, 1 + 2);
    }
}
`,
			expectedConfidence: 0,
		},
		{
			name: "JUnit 4 wildcard import should not match",
			content: `
import org.junit.*;

public class JUnit4WildcardTest {
    @Test
    public void test() {
        Assert.assertEquals(3, 1 + 2);
    }
}
`,
			expectedConfidence: 0,
		},
		{
			name: "JUnit 4 static import should not match",
			content: `
import static org.junit.Assert.assertEquals;

public class JUnit4StaticTest {
    @Test
    public void test() {
        assertEquals(3, 1 + 2);
    }
}
`,
			expectedConfidence: 0,
		},
		{
			name: "mixed JUnit 4 and 5 imports should match (migration scenario)",
			content: `
import org.junit.Test;
import org.junit.jupiter.api.Assertions;

public class MigrationTest {
    @Test
    public void test() {}
}
`,
			expectedConfidence: 40,
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

	t.Run("@TestFactory annotation", func(t *testing.T) {
		source := `
package com.example;

import org.junit.jupiter.api.TestFactory;
import java.util.stream.Stream;

class DynamicTests {
    @TestFactory
    Stream<DynamicTest> dynamicTests() {
        return Stream.of();
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "DynamicTests.java")
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

		if suite.Tests[0].Name != "dynamicTests" {
			t.Errorf("expected Name='dynamicTests', got '%s'", suite.Tests[0].Name)
		}
	})

	t.Run("@TestTemplate annotation", func(t *testing.T) {
		source := `
package com.example;

import org.junit.jupiter.api.TestTemplate;

class TemplateTests {
    @TestTemplate
    void templateTest() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "TemplateTests.java")
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

		if suite.Tests[0].Name != "templateTest" {
			t.Errorf("expected Name='templateTest', got '%s'", suite.Tests[0].Name)
		}
	})

	t.Run("custom @TestTemplate-based annotation", func(t *testing.T) {
		source := `
package com.example;

class CartesianProductTests {
    @CartesianProductTest({"0", "1"})
    void threeBits(String a, String b, String c) {}

    @CartesianProductTest
    void nFold(String string, Class<?> type) {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "CartesianProductTests.java")
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

		if suite.Tests[0].Name != "threeBits" {
			t.Errorf("expected Tests[0].Name='threeBits', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[1].Name != "nFold" {
			t.Errorf("expected Tests[1].Name='nFold', got '%s'", suite.Tests[1].Name)
		}
	})

	t.Run("custom annotations ending with Test", func(t *testing.T) {
		source := `
package com.example;

class CustomTests {
    @CustomTest
    void customTestMethod() {}

    @MyFancyTest
    void fancyTestMethod() {}

    @NotATestAnnotation
    void shouldNotBeDetected() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "CustomTests.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if len(suite.Tests) != 2 {
			t.Fatalf("expected 2 Tests (only *Test annotations), got %d", len(suite.Tests))
		}

		if suite.Tests[0].Name != "customTestMethod" {
			t.Errorf("expected Tests[0].Name='customTestMethod', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[1].Name != "fancyTestMethod" {
			t.Errorf("expected Tests[1].Name='fancyTestMethod', got '%s'", suite.Tests[1].Name)
		}
	})
}

func TestKotlinTestFileMatcher_Match(t *testing.T) {
	matcher := &KotlinTestFileMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		filename           string
		expectedConfidence int
	}{
		{"Test suffix", "CalculatorTest.kt", 20},
		{"Tests suffix", "CalculatorTests.kt", 20},
		{"Test prefix", "TestCalculator.kt", 20},
		{"Test suffix with path", "src/test/kotlin/com/example/UserServiceTest.kt", 20},
		{"regular kotlin file", "Calculator.kt", 0},
		{"non-kotlin file", "CalculatorTest.java", 0},
		{"src/main should be excluded", "src/main/kotlin/com/example/TestFactory.kt", 0},
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

func TestJUnit5KotlinParser_Parse(t *testing.T) {
	p := &JUnit5Parser{}
	ctx := context.Background()

	t.Run("basic Kotlin test methods", func(t *testing.T) {
		source := `
package com.example.project

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test

class CalculatorTests {
    @Test
    fun addsTwoNumbers() {
        val calculator = Calculator()
        assertEquals(2, calculator.add(1, 1), "1 + 1 should equal 2")
    }

    @Test
    fun subtractsTwoNumbers() {
        val calculator = Calculator()
        assertEquals(1, calculator.subtract(3, 2))
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "CalculatorTests.kt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if testFile.Path != "CalculatorTests.kt" {
			t.Errorf("expected Path='CalculatorTests.kt', got '%s'", testFile.Path)
		}
		if testFile.Framework != "junit5" {
			t.Errorf("expected Framework='junit5', got '%s'", testFile.Framework)
		}
		if testFile.Language != domain.LanguageKotlin {
			t.Errorf("expected Language=kotlin, got '%s'", testFile.Language)
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
		if suite.Tests[0].Name != "addsTwoNumbers" {
			t.Errorf("expected Tests[0].Name='addsTwoNumbers', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[1].Name != "subtractsTwoNumbers" {
			t.Errorf("expected Tests[1].Name='subtractsTwoNumbers', got '%s'", suite.Tests[1].Name)
		}
	})

	t.Run("Kotlin @Disabled annotation", func(t *testing.T) {
		source := `
package com.example.project

import org.junit.jupiter.api.Test
import org.junit.jupiter.api.Disabled

class SkippedTest {
    @Test
    @Disabled("not implemented")
    fun testSkipped() {}

    @Test
    fun testNormal() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "SkippedTest.kt")
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

	t.Run("Kotlin @ParameterizedTest annotation", func(t *testing.T) {
		source := `
package com.example.project

import org.junit.jupiter.params.ParameterizedTest
import org.junit.jupiter.params.provider.CsvSource

class ParameterizedTests {
    @ParameterizedTest(name = "{0} + {1} = {2}")
    @CsvSource(
        "0, 1, 1",
        "1, 2, 3"
    )
    fun add(first: Int, second: Int, expectedResult: Int) {
        val calculator = Calculator()
        assertEquals(expectedResult, calculator.add(first, second))
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "ParameterizedTests.kt")
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

		if suite.Tests[0].Name != "add" {
			t.Errorf("expected Name='add', got '%s'", suite.Tests[0].Name)
		}
	})

	t.Run("Kotlin backtick function names", func(t *testing.T) {
		source := `
package com.example.project

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test

class CalculatorTests {
    @Test
    fun ` + "`" + `1 + 1 = 2` + "`" + `() {
        assertEquals(2, Calculator().add(1, 1))
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "CalculatorTests.kt")
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

		if suite.Tests[0].Name != "1 + 1 = 2" {
			t.Errorf("expected Name='1 + 1 = 2', got '%s'", suite.Tests[0].Name)
		}
	})

	t.Run("Kotlin Kotest file should be skipped", func(t *testing.T) {
		source := `
package kotest

import io.kotest.core.spec.style.StringSpec
import io.kotest.matchers.shouldBe

class KotestSpec : StringSpec({
  "1 + 2 should be 3" {
    1 + 2 shouldBe 3
  }
})
`
		testFile, err := p.Parse(ctx, []byte(source), "KotestSpec.kt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have no suites because Kotest specs are skipped
		if len(testFile.Suites) != 0 {
			t.Errorf("expected 0 Suites for Kotest file, got %d", len(testFile.Suites))
		}
	})
}
