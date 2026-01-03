package testng

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
)

func TestNewDefinition(t *testing.T) {
	def := NewDefinition()

	if def.Name != framework.FrameworkTestNG {
		t.Errorf("expected Name='%s', got '%s'", framework.FrameworkTestNG, def.Name)
	}
	if def.Priority != framework.PrioritySpecialized {
		t.Errorf("expected Priority=%d, got %d", framework.PrioritySpecialized, def.Priority)
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

func TestTestNGFileMatcher_Match(t *testing.T) {
	matcher := &TestNGFileMatcher{}
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

func TestTestNGContentMatcher_Match(t *testing.T) {
	matcher := &TestNGContentMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		content            string
		expectedConfidence int
	}{
		{
			name: "TestNG import",
			content: `
import org.testng.annotations.Test;

class CalculatorTest {
    @Test
    public void testAdd() {}
}
`,
			expectedConfidence: 40,
		},
		{
			name: "@Test(enabled=false) annotation",
			content: `
class CalculatorTest {
    @Test(enabled = false)
    public void testDisabled() {}
}
`,
			expectedConfidence: 40,
		},
		{
			name: "@DataProvider annotation",
			content: `
class DataProviderTest {
    @DataProvider
    public Object[][] data() { return null; }
}
`,
			expectedConfidence: 40,
		},
		{
			name: "@BeforeClass annotation",
			content: `
class SetupTest {
    @BeforeClass
    public void setup() {}
}
`,
			expectedConfidence: 40,
		},
		{
			name: "@AfterClass annotation",
			content: `
class CleanupTest {
    @AfterClass
    public void cleanup() {}
}
`,
			expectedConfidence: 40,
		},
		{
			name: "@BeforeMethod annotation",
			content: `
class MethodSetupTest {
    @BeforeMethod
    public void beforeEach() {}
}
`,
			expectedConfidence: 40,
		},
		{
			name: "@AfterMethod annotation",
			content: `
class MethodCleanupTest {
    @AfterMethod
    public void afterEach() {}
}
`,
			expectedConfidence: 40,
		},
		{
			name: "no TestNG patterns",
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

func TestTestNGParser_Parse(t *testing.T) {
	p := &TestNGParser{}
	ctx := context.Background()

	t.Run("basic test methods", func(t *testing.T) {
		source := `
package com.example;

import org.testng.annotations.Test;

class CalculatorTest {
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
		if testFile.Framework != framework.FrameworkTestNG {
			t.Errorf("expected Framework='%s', got '%s'", framework.FrameworkTestNG, testFile.Framework)
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

	t.Run("@Test(enabled=false) annotation", func(t *testing.T) {
		source := `
package com.example;

import org.testng.annotations.Test;

class SkippedTest {
    @Test(enabled = false)
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

		if suite.Tests[1].Name != "testNormal" {
			t.Errorf("expected Tests[1].Name='testNormal', got '%s'", suite.Tests[1].Name)
		}
		if suite.Tests[1].Status != domain.TestStatusActive {
			t.Errorf("expected Tests[1].Status='active', got '%s'", suite.Tests[1].Status)
		}
	})

	t.Run("@Test(enabled=false) no spaces", func(t *testing.T) {
		source := `
package com.example;

import org.testng.annotations.Test;

class NoSpaceTest {
    @Test(enabled=false)
    public void testSkipped() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "NoSpaceTest.java")
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

		if suite.Tests[0].Status != domain.TestStatusSkipped {
			t.Errorf("expected Tests[0].Status='skipped', got '%s'", suite.Tests[0].Status)
		}
	})

	t.Run("@Test with description attribute", func(t *testing.T) {
		source := `
package com.example;

import org.testng.annotations.Test;

class DescriptionTest {
    @Test(description = "Addition should work correctly")
    public void testAdd() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "DescriptionTest.java")
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

	t.Run("@Test with multiple attributes", func(t *testing.T) {
		source := `
package com.example;

import org.testng.annotations.Test;

class MultiAttrTest {
    @Test(description = "Test with priority", priority = 1, enabled = false)
    public void testWithPriority() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "MultiAttrTest.java")
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

		if suite.Tests[0].Name != "Test with priority" {
			t.Errorf("expected Name='Test with priority', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[0].Status != domain.TestStatusSkipped {
			t.Errorf("expected Status='skipped', got '%s'", suite.Tests[0].Status)
		}
	})

	t.Run("class with @Test annotation", func(t *testing.T) {
		source := `
package com.example;

import org.testng.annotations.Test;

@Test
class AllTestsClass {
    public void testOne() {}
    public void testTwo() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "AllTestsClass.java")
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
		if suite.Tests[0].Name != "testOne" {
			t.Errorf("expected Tests[0].Name='testOne', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[1].Name != "testTwo" {
			t.Errorf("expected Tests[1].Name='testTwo', got '%s'", suite.Tests[1].Name)
		}
	})

	t.Run("class-level @Test excludes config methods", func(t *testing.T) {
		source := `
package com.example;

import org.testng.annotations.Test;
import org.testng.annotations.BeforeMethod;
import org.testng.annotations.AfterMethod;
import org.testng.annotations.DataProvider;

@Test
class ConfigMethodsTest {
    public void testMethod() {}

    @BeforeMethod
    public void setup() {}

    @AfterMethod
    public void teardown() {}

    @DataProvider
    public Object[][] data() { return null; }

    private void helper() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "ConfigMethodsTest.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if len(suite.Tests) != 1 {
			t.Fatalf("expected 1 Test (testMethod only), got %d", len(suite.Tests))
		}
		if suite.Tests[0].Name != "testMethod" {
			t.Errorf("expected Tests[0].Name='testMethod', got '%s'", suite.Tests[0].Name)
		}
	})

	t.Run("class-level @Test with method-level @Test mixed", func(t *testing.T) {
		source := `
package com.example;

import org.testng.annotations.Test;

@Test
class MixedTest {
    @Test(description = "explicit test")
    public void explicitTest() {}

    public void implicitTest() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "MixedTest.java")
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
		if suite.Tests[0].Name != "explicit test" {
			t.Errorf("expected Tests[0].Name='explicit test', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[1].Name != "implicitTest" {
			t.Errorf("expected Tests[1].Name='implicitTest', got '%s'", suite.Tests[1].Name)
		}
	})

	t.Run("class-level @Test(enabled=false) marks all methods skipped", func(t *testing.T) {
		source := `
package com.example;

import org.testng.annotations.Test;

@Test(enabled = false)
class SkippedClassTest {
    public void testOne() {}
    public void testTwo() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "SkippedClassTest.java")
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
		for i, test := range suite.Tests {
			if test.Status != domain.TestStatusSkipped {
				t.Errorf("expected Tests[%d].Status='skipped', got '%s'", i, test.Status)
			}
		}
	})

	t.Run("class-level @Test with groups attribute", func(t *testing.T) {
		source := `
package com.example;

import org.testng.annotations.Test;

@Test(groups = "issue2195")
public class TestClass {
    public void someMethod() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "TestClass.java")
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
		if suite.Tests[0].Name != "someMethod" {
			t.Errorf("expected Tests[0].Name='someMethod', got '%s'", suite.Tests[0].Name)
		}
	})

	t.Run("class-level @Test excludes non-public methods", func(t *testing.T) {
		source := `
package com.example;

import org.testng.annotations.Test;

@Test
class VisibilityTest {
    public void publicTest() {}
    protected void protectedMethod() {}
    void defaultMethod() {}
    private void privateMethod() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "VisibilityTest.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if len(suite.Tests) != 1 {
			t.Fatalf("expected 1 Test (publicTest only), got %d", len(suite.Tests))
		}
		if suite.Tests[0].Name != "publicTest" {
			t.Errorf("expected Tests[0].Name='publicTest', got '%s'", suite.Tests[0].Name)
		}
	})

	t.Run("no test methods", func(t *testing.T) {
		source := `
package com.example;

class NoTests {
    public void helperMethod() {}
}
`
		testFile, err := p.Parse(ctx, []byte(source), "NoTests.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 0 {
			t.Errorf("expected 0 Suites, got %d", len(testFile.Suites))
		}
	})

	t.Run("nested class with @Test method", func(t *testing.T) {
		source := `
package com.example;

import org.testng.annotations.Test;

public class OuterClass {
    public static class InnerTestClass {
        @Test
        public void testInner() {}
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "OuterClass.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite (outer), got %d", len(testFile.Suites))
		}

		outerSuite := testFile.Suites[0]
		if outerSuite.Name != "OuterClass" {
			t.Errorf("expected outer Suite.Name='OuterClass', got '%s'", outerSuite.Name)
		}
		if len(outerSuite.Tests) != 0 {
			t.Errorf("expected 0 Tests in outer suite, got %d", len(outerSuite.Tests))
		}
		if len(outerSuite.Suites) != 1 {
			t.Fatalf("expected 1 nested Suite, got %d", len(outerSuite.Suites))
		}

		nestedSuite := outerSuite.Suites[0]
		if nestedSuite.Name != "InnerTestClass" {
			t.Errorf("expected nested Suite.Name='InnerTestClass', got '%s'", nestedSuite.Name)
		}
		if len(nestedSuite.Tests) != 1 {
			t.Fatalf("expected 1 Test in nested suite, got %d", len(nestedSuite.Tests))
		}
		if nestedSuite.Tests[0].Name != "testInner" {
			t.Errorf("expected Test.Name='testInner', got '%s'", nestedSuite.Tests[0].Name)
		}
	})

	t.Run("multiple nested classes with @Test methods", func(t *testing.T) {
		source := `
package com.example;

import org.testng.annotations.Test;

public class TestClassContainer {
    public static class FirstTestClass {
        @Test
        public void testMethod() {}
    }

    public static class SecondTestClass {
        @Test
        public void testMethod() {}
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "TestClassContainer.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite (outer), got %d", len(testFile.Suites))
		}

		outerSuite := testFile.Suites[0]
		if len(outerSuite.Suites) != 2 {
			t.Fatalf("expected 2 nested Suites, got %d", len(outerSuite.Suites))
		}

		if outerSuite.Suites[0].Name != "FirstTestClass" {
			t.Errorf("expected Suites[0].Name='FirstTestClass', got '%s'", outerSuite.Suites[0].Name)
		}
		if outerSuite.Suites[1].Name != "SecondTestClass" {
			t.Errorf("expected Suites[1].Name='SecondTestClass', got '%s'", outerSuite.Suites[1].Name)
		}

		totalTests := len(outerSuite.Suites[0].Tests) + len(outerSuite.Suites[1].Tests)
		if totalTests != 2 {
			t.Errorf("expected 2 total Tests in nested suites, got %d", totalTests)
		}
	})

	t.Run("outer and nested class both have @Test methods", func(t *testing.T) {
		source := `
package com.example;

import org.testng.annotations.Test;

public class NestedStaticSampleTest {
    @Test
    public void outerTest() {}

    public static class Nested {
        @Test
        public void nestedTest() {}
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "NestedStaticSampleTest.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite (outer), got %d", len(testFile.Suites))
		}

		outerSuite := testFile.Suites[0]
		if outerSuite.Name != "NestedStaticSampleTest" {
			t.Errorf("expected outer Suite.Name='NestedStaticSampleTest', got '%s'", outerSuite.Name)
		}
		if len(outerSuite.Tests) != 1 {
			t.Fatalf("expected 1 Test in outer suite, got %d", len(outerSuite.Tests))
		}
		if outerSuite.Tests[0].Name != "outerTest" {
			t.Errorf("expected outer Test.Name='outerTest', got '%s'", outerSuite.Tests[0].Name)
		}

		if len(outerSuite.Suites) != 1 {
			t.Fatalf("expected 1 nested Suite, got %d", len(outerSuite.Suites))
		}

		nestedSuite := outerSuite.Suites[0]
		if nestedSuite.Name != "Nested" {
			t.Errorf("expected nested Suite.Name='Nested', got '%s'", nestedSuite.Name)
		}
		if len(nestedSuite.Tests) != 1 {
			t.Fatalf("expected 1 Test in nested suite, got %d", len(nestedSuite.Tests))
		}
		if nestedSuite.Tests[0].Name != "nestedTest" {
			t.Errorf("expected nested Test.Name='nestedTest', got '%s'", nestedSuite.Tests[0].Name)
		}
	})

	t.Run("nested class with class-level @Test", func(t *testing.T) {
		source := `
package com.example;

import org.testng.annotations.Test;

public class ClassContainer {
    @Test
    public static class NonGroupClass {
        public void step1() {}
        public void step2() {}
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "ClassContainer.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite (outer), got %d", len(testFile.Suites))
		}

		outerSuite := testFile.Suites[0]
		if len(outerSuite.Suites) != 1 {
			t.Fatalf("expected 1 nested Suite, got %d", len(outerSuite.Suites))
		}

		nestedSuite := outerSuite.Suites[0]
		if nestedSuite.Name != "NonGroupClass" {
			t.Errorf("expected nested Suite.Name='NonGroupClass', got '%s'", nestedSuite.Name)
		}
		// Class-level @Test makes all public methods tests
		if len(nestedSuite.Tests) != 2 {
			t.Fatalf("expected 2 Tests in nested suite (from class-level @Test), got %d", len(nestedSuite.Tests))
		}
	})
}
