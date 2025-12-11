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

		// Note: TestNG class-level @Test makes all public methods tests,
		// but we only detect method-level @Test annotations
		if len(testFile.Suites) != 0 {
			// This is expected since we don't detect class-level @Test for test methods
			t.Logf("Note: Class-level @Test not supported yet")
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
}
