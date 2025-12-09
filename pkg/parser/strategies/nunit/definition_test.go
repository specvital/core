package nunit

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
)

func TestNewDefinition(t *testing.T) {
	def := NewDefinition()

	if def.Name != "nunit" {
		t.Errorf("expected Name='nunit', got '%s'", def.Name)
	}
	if def.Priority != framework.PriorityGeneric {
		t.Errorf("expected Priority=%d, got %d", framework.PriorityGeneric, def.Priority)
	}
	if len(def.Languages) != 1 || def.Languages[0] != domain.LanguageCSharp {
		t.Errorf("expected Languages=[csharp], got %v", def.Languages)
	}
	if def.Parser == nil {
		t.Error("expected Parser to be non-nil")
	}
	if len(def.Matchers) != 3 {
		t.Errorf("expected 3 Matchers, got %d", len(def.Matchers))
	}
}

func TestNUnitFileMatcher_Match(t *testing.T) {
	matcher := &NUnitFileMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		filename           string
		expectedConfidence int
	}{
		{"Test suffix", "CalculatorTest.cs", 20},
		{"Tests suffix", "CalculatorTests.cs", 20},
		{"Test prefix", "TestCalculator.cs", 20},
		{"Test suffix with path", "src/Tests/UserServiceTests.cs", 20},
		{"regular cs file", "Calculator.cs", 0},
		{"non-cs file", "CalculatorTest.java", 0},
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

func TestNUnitContentMatcher_Match(t *testing.T) {
	matcher := &NUnitContentMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		content            string
		expectedConfidence int
	}{
		{
			name: "[Test] attribute",
			content: `
using NUnit.Framework;

[TestFixture]
public class CalculatorTests
{
    [Test]
    public void Add_ReturnsSum()
    {
        Assert.AreEqual(3, 1 + 2);
    }
}
`,
			expectedConfidence: 40,
		},
		{
			name: "[TestCase] attribute",
			content: `
using NUnit.Framework;

public class CalculatorTests
{
    [TestCase(1, 2, 3)]
    [TestCase(2, 3, 5)]
    public void Add_ReturnsSum(int a, int b, int expected)
    {
        Assert.AreEqual(expected, a + b);
    }
}
`,
			expectedConfidence: 40,
		},
		{
			name: "[TestFixture] attribute",
			content: `
using NUnit.Framework;

[TestFixture]
public class SomeTests
{
}
`,
			expectedConfidence: 40,
		},
		{
			name: "[TestCaseSource] attribute",
			content: `
using NUnit.Framework;

public class CalculatorTests
{
    [TestCaseSource(nameof(TestData))]
    public void TestMethod(int value) {}
}
`,
			expectedConfidence: 40,
		},
		{
			name: "using NUnit.Framework",
			content: `
using NUnit.Framework;

public class SomeClass {}
`,
			expectedConfidence: 40,
		},
		{
			name: "[Ignore] attribute",
			content: `
using NUnit.Framework;

public class CalculatorTests
{
    [Test]
    [Ignore("Not implemented")]
    public void IgnoredTest() {}
}
`,
			expectedConfidence: 40,
		},
		{
			name: "no NUnit patterns",
			content: `
public class Calculator
{
    public int Add(int a, int b)
    {
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

func TestNUnitParser_Parse(t *testing.T) {
	p := &NUnitParser{}
	ctx := context.Background()

	t.Run("basic [Test] methods", func(t *testing.T) {
		source := `
using NUnit.Framework;

[TestFixture]
public class CalculatorTests
{
    [Test]
    public void Add_ReturnsSum()
    {
        Assert.AreEqual(3, 1 + 2);
    }

    [Test]
    public void Subtract_ReturnsDifference()
    {
        Assert.AreEqual(1, 3 - 2);
    }

    public void HelperMethod()
    {
        // not a test
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "CalculatorTests.cs")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if testFile.Path != "CalculatorTests.cs" {
			t.Errorf("expected Path='CalculatorTests.cs', got '%s'", testFile.Path)
		}
		if testFile.Framework != "nunit" {
			t.Errorf("expected Framework='nunit', got '%s'", testFile.Framework)
		}
		if testFile.Language != domain.LanguageCSharp {
			t.Errorf("expected Language=csharp, got '%s'", testFile.Language)
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
		if suite.Tests[0].Name != "Add_ReturnsSum" {
			t.Errorf("expected Tests[0].Name='Add_ReturnsSum', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[1].Name != "Subtract_ReturnsDifference" {
			t.Errorf("expected Tests[1].Name='Subtract_ReturnsDifference', got '%s'", suite.Tests[1].Name)
		}
	})

	t.Run("[TestCase] parameterized tests", func(t *testing.T) {
		source := `
using NUnit.Framework;

public class MathTests
{
    [TestCase(1, 2, 3)]
    [TestCase(2, 3, 5)]
    public void Add_WithValues_ReturnsSum(int a, int b, int expected)
    {
        Assert.AreEqual(expected, a + b);
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "MathTests.cs")
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

		if suite.Tests[0].Name != "Add_WithValues_ReturnsSum" {
			t.Errorf("expected Name='Add_WithValues_ReturnsSum', got '%s'", suite.Tests[0].Name)
		}
	})

	t.Run("[Ignore] marks test as skipped", func(t *testing.T) {
		source := `
using NUnit.Framework;

public class SkippedTests
{
    [Test]
    [Ignore("Not implemented yet")]
    public void SkippedTest()
    {
    }

    [Test]
    public void NormalTest()
    {
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "SkippedTests.cs")
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

		if suite.Tests[0].Name != "SkippedTest" {
			t.Errorf("expected Tests[0].Name='SkippedTest', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[0].Status != domain.TestStatusSkipped {
			t.Errorf("expected Tests[0].Status='skipped', got '%s'", suite.Tests[0].Status)
		}

		if suite.Tests[1].Name != "NormalTest" {
			t.Errorf("expected Tests[1].Name='NormalTest', got '%s'", suite.Tests[1].Name)
		}
		if suite.Tests[1].Status != domain.TestStatusActive {
			t.Errorf("expected Tests[1].Status='active', got '%s'", suite.Tests[1].Status)
		}
	})

	t.Run("[Test(Description = ...)] uses description", func(t *testing.T) {
		source := `
using NUnit.Framework;

public class DescriptionTests
{
    [Test(Description = "Addition should work correctly")]
    public void TestAdd()
    {
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "DescriptionTests.cs")
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

	t.Run("[TestCaseSource] parameterized tests", func(t *testing.T) {
		source := `
using NUnit.Framework;

public class TestCaseSourceTests
{
    public static IEnumerable<TestCaseData> TestData
    {
        get
        {
            yield return new TestCaseData(1, 2, 3);
            yield return new TestCaseData(2, 3, 5);
        }
    }

    [TestCaseSource(nameof(TestData))]
    public void Add_WithTestCaseSource(int a, int b, int expected)
    {
        Assert.AreEqual(expected, a + b);
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "TestCaseSourceTests.cs")
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

		if suite.Tests[0].Name != "Add_WithTestCaseSource" {
			t.Errorf("expected Name='Add_WithTestCaseSource', got '%s'", suite.Tests[0].Name)
		}
	})

	t.Run("nested classes", func(t *testing.T) {
		source := `
using NUnit.Framework;

[TestFixture]
public class OuterTests
{
    [Test]
    public void OuterTest()
    {
    }

    [TestFixture]
    public class InnerTests
    {
        [Test]
        public void InnerTest()
        {
        }
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "OuterTests.cs")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if suite.Name != "OuterTests" {
			t.Errorf("expected Suite.Name='OuterTests', got '%s'", suite.Name)
		}
		if len(suite.Tests) != 1 {
			t.Fatalf("expected 1 Test in outer suite, got %d", len(suite.Tests))
		}
		if suite.Tests[0].Name != "OuterTest" {
			t.Errorf("expected outer test name='OuterTest', got '%s'", suite.Tests[0].Name)
		}

		if len(suite.Suites) != 1 {
			t.Fatalf("expected 1 nested Suite, got %d", len(suite.Suites))
		}

		nested := suite.Suites[0]
		if nested.Name != "InnerTests" {
			t.Errorf("expected nested Suite.Name='InnerTests', got '%s'", nested.Name)
		}
		if len(nested.Tests) != 1 {
			t.Fatalf("expected 1 Test in nested suite, got %d", len(nested.Tests))
		}
		if nested.Tests[0].Name != "InnerTest" {
			t.Errorf("expected nested test name='InnerTest', got '%s'", nested.Tests[0].Name)
		}
	})

	t.Run("[Ignore] on class marks all tests as skipped", func(t *testing.T) {
		source := `
using NUnit.Framework;

[TestFixture]
[Ignore("Entire class ignored")]
public class IgnoredClassTests
{
    [Test]
    public void Test1()
    {
    }

    [Test]
    public void Test2()
    {
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "IgnoredClassTests.cs")
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

		for i, test := range suite.Tests {
			if test.Status != domain.TestStatusSkipped {
				t.Errorf("expected Tests[%d].Status='skipped', got '%s'", i, test.Status)
			}
		}
	})

	t.Run("multiple classes in file", func(t *testing.T) {
		source := `
using NUnit.Framework;

[TestFixture]
public class FirstTests
{
    [Test]
    public void FirstTest()
    {
    }
}

[TestFixture]
public class SecondTests
{
    [Test]
    public void SecondTest()
    {
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "MultipleClasses.cs")
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
}
