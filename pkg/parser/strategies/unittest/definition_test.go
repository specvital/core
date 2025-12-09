package unittest

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
)

func TestNewDefinition(t *testing.T) {
	def := NewDefinition()

	if def.Name != "unittest" {
		t.Errorf("expected Name='unittest', got '%s'", def.Name)
	}
	if def.Priority != framework.PriorityGeneric {
		t.Errorf("expected Priority=%d, got %d", framework.PriorityGeneric, def.Priority)
	}
	if len(def.Languages) != 1 || def.Languages[0] != domain.LanguagePython {
		t.Errorf("expected Languages=[python], got %v", def.Languages)
	}
	if def.Parser == nil {
		t.Error("expected Parser to be non-nil")
	}
	if len(def.Matchers) != 3 {
		t.Errorf("expected 3 Matchers, got %d", len(def.Matchers))
	}
}

func TestUnittestFileMatcher_Match(t *testing.T) {
	matcher := &UnittestFileMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		filename           string
		expectedConfidence int
	}{
		{"test_ prefix", "test_calculator.py", 20},
		{"_test suffix", "calculator_test.py", 20},
		{"test_ prefix with path", "tests/unit/test_user.py", 20},
		{"regular python file", "calculator.py", 0},
		{"non-test file", "utils.py", 0},
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

func TestUnittestContentMatcher_Match(t *testing.T) {
	matcher := &UnittestContentMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		content            string
		expectedConfidence int
	}{
		{
			name: "unittest.TestCase inheritance",
			content: `
import unittest

class MyTest(unittest.TestCase):
    def test_something(self):
        self.assertEqual(1, 1)
`,
			expectedConfidence: 40,
		},
		{
			name: "unittest.skip decorator",
			content: `
import unittest

@unittest.skip("not implemented")
def test_skipped():
    pass
`,
			expectedConfidence: 40,
		},
		{
			name: "unittest.skipIf decorator",
			content: `
import unittest

@unittest.skipIf(True, "condition met")
def test_conditional():
    pass
`,
			expectedConfidence: 40,
		},
		{
			name: "unittest.expectedFailure decorator",
			content: `
import unittest

@unittest.expectedFailure
def test_xfail():
    pass
`,
			expectedConfidence: 40,
		},
		{
			name: "self.assert* assertion",
			content: `
class MyTest:
    def test_assert(self):
        self.assertEqual(1, 1)
        self.assertTrue(True)
`,
			expectedConfidence: 40,
		},
		{
			name: "unittest.main()",
			content: `
import unittest

if __name__ == '__main__':
    unittest.main()
`,
			expectedConfidence: 40,
		},
		{
			name: "no unittest patterns",
			content: `
def test_simple():
    assert 1 + 1 == 2
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
			if tt.expectedConfidence > 0 && len(result.Evidence) == 0 {
				t.Error("expected Evidence to be non-empty")
			}
		})
	}
}

func TestUnittestParser_Parse(t *testing.T) {
	p := &UnittestParser{}
	ctx := context.Background()

	t.Run("basic TestCase class", func(t *testing.T) {
		source := `
import unittest

class TestCalculator(unittest.TestCase):
    def test_add(self):
        self.assertEqual(1 + 1, 2)

    def test_subtract(self):
        self.assertEqual(5 - 3, 2)

    def helper_method(self):
        return 42
`
		testFile, err := p.Parse(ctx, []byte(source), "test_calculator.py")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if testFile.Path != "test_calculator.py" {
			t.Errorf("expected Path='test_calculator.py', got '%s'", testFile.Path)
		}
		if testFile.Framework != "unittest" {
			t.Errorf("expected Framework='unittest', got '%s'", testFile.Framework)
		}
		if testFile.Language != domain.LanguagePython {
			t.Errorf("expected Language=python, got '%s'", testFile.Language)
		}
		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if suite.Name != "TestCalculator" {
			t.Errorf("expected Suite.Name='TestCalculator', got '%s'", suite.Name)
		}
		if len(suite.Tests) != 2 {
			t.Fatalf("expected 2 Tests in suite, got %d", len(suite.Tests))
		}
		if suite.Tests[0].Name != "test_add" {
			t.Errorf("expected Tests[0].Name='test_add', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[1].Name != "test_subtract" {
			t.Errorf("expected Tests[1].Name='test_subtract', got '%s'", suite.Tests[1].Name)
		}
	})

	t.Run("unittest.skip decorator", func(t *testing.T) {
		source := `
import unittest

class TestWithSkip(unittest.TestCase):
    @unittest.skip("not implemented")
    def test_skipped(self):
        pass

    def test_normal(self):
        pass
`
		testFile, err := p.Parse(ctx, []byte(source), "test_skip.py")
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

		if suite.Tests[0].Name != "test_skipped" {
			t.Errorf("expected Tests[0].Name='test_skipped', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[0].Status != domain.TestStatusSkipped {
			t.Errorf("expected Tests[0].Status='skipped', got '%s'", suite.Tests[0].Status)
		}

		if suite.Tests[1].Name != "test_normal" {
			t.Errorf("expected Tests[1].Name='test_normal', got '%s'", suite.Tests[1].Name)
		}
		if suite.Tests[1].Status != domain.TestStatusActive {
			t.Errorf("expected Tests[1].Status='active', got '%s'", suite.Tests[1].Status)
		}
	})

	t.Run("unittest.skipIf decorator", func(t *testing.T) {
		source := `
import unittest

class TestConditionalSkip(unittest.TestCase):
    @unittest.skipIf(True, "condition met")
    def test_conditional_skip(self):
        pass
`
		testFile, err := p.Parse(ctx, []byte(source), "test_skipif.py")
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
			t.Errorf("expected Status='skipped', got '%s'", suite.Tests[0].Status)
		}
	})

	t.Run("unittest.expectedFailure decorator", func(t *testing.T) {
		source := `
import unittest

class TestExpectedFailure(unittest.TestCase):
    @unittest.expectedFailure
    def test_xfail(self):
        self.assertEqual(1, 2)
`
		testFile, err := p.Parse(ctx, []byte(source), "test_expected_failure.py")
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

		if suite.Tests[0].Status != domain.TestStatusXfail {
			t.Errorf("expected Status='xfail', got '%s'", suite.Tests[0].Status)
		}
	})

	t.Run("class with skip decorator", func(t *testing.T) {
		source := `
import unittest

@unittest.skip("entire class skipped")
class TestSkippedClass(unittest.TestCase):
    def test_one(self):
        pass

    def test_two(self):
        pass
`
		testFile, err := p.Parse(ctx, []byte(source), "test_class_skip.py")
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
			t.Fatalf("expected 2 Tests in suite, got %d", len(suite.Tests))
		}
		if suite.Tests[0].Status != domain.TestStatusSkipped {
			t.Errorf("expected Tests[0].Status='skipped', got '%s'", suite.Tests[0].Status)
		}
		if suite.Tests[1].Status != domain.TestStatusSkipped {
			t.Errorf("expected Tests[1].Status='skipped', got '%s'", suite.Tests[1].Status)
		}
	})

	t.Run("multiple TestCase classes", func(t *testing.T) {
		source := `
import unittest

class TestAddition(unittest.TestCase):
    def test_add(self):
        self.assertEqual(1 + 1, 2)

class TestMultiplication(unittest.TestCase):
    def test_multiply(self):
        self.assertEqual(2 * 3, 6)
`
		testFile, err := p.Parse(ctx, []byte(source), "test_multiple.py")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 2 {
			t.Fatalf("expected 2 Suites, got %d", len(testFile.Suites))
		}

		if testFile.Suites[0].Name != "TestAddition" {
			t.Errorf("expected Suites[0].Name='TestAddition', got '%s'", testFile.Suites[0].Name)
		}
		if testFile.Suites[1].Name != "TestMultiplication" {
			t.Errorf("expected Suites[1].Name='TestMultiplication', got '%s'", testFile.Suites[1].Name)
		}
	})

	t.Run("TestCase with Test suffix", func(t *testing.T) {
		source := `
import unittest

class CalculatorTest(unittest.TestCase):
    def test_add(self):
        pass
`
		testFile, err := p.Parse(ctx, []byte(source), "test_suffix.py")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		if testFile.Suites[0].Name != "CalculatorTest" {
			t.Errorf("expected Suite.Name='CalculatorTest', got '%s'", testFile.Suites[0].Name)
		}
	})

	t.Run("method decorator overrides class status", func(t *testing.T) {
		source := `
import unittest

@unittest.skip("class skipped")
class TestMixed(unittest.TestCase):
    @unittest.expectedFailure
    def test_method_override(self):
        self.assertEqual(1, 2)

    def test_inherited(self):
        pass
`
		testFile, err := p.Parse(ctx, []byte(source), "test_mixed.py")
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

		// Method decorator takes precedence
		if suite.Tests[0].Name != "test_method_override" {
			t.Errorf("expected Tests[0].Name='test_method_override', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[0].Status != domain.TestStatusXfail {
			t.Errorf("expected Tests[0].Status='xfail' (from expectedFailure), got '%s'", suite.Tests[0].Status)
		}

		// Method without decorator inherits class status
		if suite.Tests[1].Name != "test_inherited" {
			t.Errorf("expected Tests[1].Name='test_inherited', got '%s'", suite.Tests[1].Name)
		}
		if suite.Tests[1].Status != domain.TestStatusSkipped {
			t.Errorf("expected Tests[1].Status='skipped' (inherited), got '%s'", suite.Tests[1].Status)
		}
	})

	t.Run("non-test class should be ignored", func(t *testing.T) {
		source := `
import unittest

class Helper:
    def do_something(self):
        pass

class TestReal(unittest.TestCase):
    def test_real(self):
        pass
`
		testFile, err := p.Parse(ctx, []byte(source), "test_helper.py")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite (Helper should be ignored), got %d", len(testFile.Suites))
		}

		if testFile.Suites[0].Name != "TestReal" {
			t.Errorf("expected Suite.Name='TestReal', got '%s'", testFile.Suites[0].Name)
		}
	})
}
