package junit4

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
	"github.com/specvital/core/pkg/parser/strategies/shared/javaast"
)

func TestNewDefinition(t *testing.T) {
	def := NewDefinition()

	if def.Name != "junit4" {
		t.Errorf("expected Name='junit4', got '%s'", def.Name)
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

func TestJUnit4ContentMatcher_Match(t *testing.T) {
	matcher := &JUnit4ContentMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		content            string
		expectedConfidence int
	}{
		{
			name: "@Test annotation with JUnit 4 import",
			content: `
import org.junit.Test;
import static org.junit.Assert.assertEquals;

public class CalculatorTest {
    @Test
    public void testAdd() {
        assertEquals(3, 1 + 2);
    }
}
`,
			expectedConfidence: 40,
		},
		{
			name: "@Before annotation",
			content: `
import org.junit.Before;
import org.junit.Test;

public class SetupTest {
    @Before
    public void setUp() {}

    @Test
    public void testSomething() {}
}
`,
			expectedConfidence: 40,
		},
		{
			name: "@After annotation",
			content: `
import org.junit.After;
import org.junit.Test;

public class TeardownTest {
    @After
    public void tearDown() {}

    @Test
    public void testSomething() {}
}
`,
			expectedConfidence: 40,
		},
		{
			name: "@Ignore annotation",
			content: `
import org.junit.Test;
import org.junit.Ignore;

public class SkippedTest {
    @Test
    @Ignore
    public void testSkipped() {}
}
`,
			expectedConfidence: 40,
		},
		{
			name: "@RunWith annotation",
			content: `
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.junit.MockitoJUnitRunner;

@RunWith(MockitoJUnitRunner.class)
public class MockTest {
    @Test
    public void testMock() {}
}
`,
			expectedConfidence: 40,
		},
		{
			name: "wildcard import",
			content: `
import org.junit.*;

public class WildcardTest {
    @Test
    public void testWildcard() {}
}
`,
			expectedConfidence: 40,
		},
		{
			name: "static import",
			content: `
import static org.junit.Assert.assertEquals;

public class StaticImportTest {
    @Test
    public void testStatic() {
        assertEquals(1, 1);
    }
}
`,
			expectedConfidence: 40,
		},
		{
			name: "JUnit 5 file should not match",
			content: `
import org.junit.jupiter.api.Test;

class JUnit5Test {
    @Test
    void test() {}
}
`,
			expectedConfidence: 0,
		},
		{
			name: "mixed JUnit 4 and 5 imports should not match (migration scenario - prefer JUnit 5)",
			content: `
import org.junit.Test;
import org.junit.jupiter.api.Assertions;

public class MigrationTest {
    @Test
    public void test() {}
}
`,
			expectedConfidence: 0,
		},
		{
			name: "no JUnit patterns",
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
			name: "no JUnit 4 import (only annotation without import)",
			content: `
public class NoImportTest {
    @Test
    public void test() {}
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

func TestJUnit4Parser_Parse(t *testing.T) {
	p := &JUnit4Parser{}
	ctx := context.Background()

	t.Run("basic test methods", func(t *testing.T) {
		source := `
package com.example;

import org.junit.Test;

public class CalculatorTest {
    @Test
    public void testAdd() {
        assertEquals(3, 1 + 2);
    }

    @Test
    public void testSubtract() {
        assertEquals(1, 3 - 2);
    }

    public void helperMethod() {
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
		if testFile.Framework != "junit4" {
			t.Errorf("expected Framework='junit4', got '%s'", testFile.Framework)
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

	t.Run("@Ignore annotation on method", func(t *testing.T) {
		source := `
package com.example;

import org.junit.Test;
import org.junit.Ignore;

public class SkippedTest {
    @Test
    @Ignore
    public void testSkipped() {}

    @Test
    public void testNormal() {}
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
		if suite.Tests[0].Modifier != "@Ignore" {
			t.Errorf("expected Tests[0].Modifier='@Ignore', got '%s'", suite.Tests[0].Modifier)
		}

		if suite.Tests[1].Name != "testNormal" {
			t.Errorf("expected Tests[1].Name='testNormal', got '%s'", suite.Tests[1].Name)
		}
		if suite.Tests[1].Status != domain.TestStatusActive {
			t.Errorf("expected Tests[1].Status='active', got '%s'", suite.Tests[1].Status)
		}
	})

	t.Run("@Ignore annotation on class", func(t *testing.T) {
		source := `
package com.example;

import org.junit.Test;
import org.junit.Ignore;

@Ignore
public class IgnoredClassTest {
    @Test
    public void testOne() {}

    @Test
    public void testTwo() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "IgnoredClassTest.java")
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

	t.Run("@RunWith annotation", func(t *testing.T) {
		source := `
package com.example;

import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.junit.MockitoJUnitRunner;

@RunWith(MockitoJUnitRunner.class)
public class MockTest {
    @Test
    public void testMock() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "MockTest.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if suite.Name != "MockTest" {
			t.Errorf("expected Suite.Name='MockTest', got '%s'", suite.Name)
		}
		if len(suite.Tests) != 1 {
			t.Fatalf("expected 1 Test, got %d", len(suite.Tests))
		}
	})

	t.Run("multiple test classes in one file", func(t *testing.T) {
		source := `
package com.example;

import org.junit.Test;

public class FirstTest {
    @Test
    public void testFirst() {}
}

class SecondTest {
    @Test
    public void testSecond() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "MultipleTests.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 2 {
			t.Fatalf("expected 2 Suites, got %d", len(testFile.Suites))
		}

		if testFile.Suites[0].Name != "FirstTest" {
			t.Errorf("expected Suites[0].Name='FirstTest', got '%s'", testFile.Suites[0].Name)
		}
		if testFile.Suites[1].Name != "SecondTest" {
			t.Errorf("expected Suites[1].Name='SecondTest', got '%s'", testFile.Suites[1].Name)
		}
	})

	t.Run("class with no tests", func(t *testing.T) {
		source := `
package com.example;

public class NoTestsClass {
    public void helperMethod() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "NoTestsClass.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 0 {
			t.Errorf("expected 0 Suites, got %d", len(testFile.Suites))
		}
	})
}
