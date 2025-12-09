package pytest

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
)

func TestNewDefinition(t *testing.T) {
	def := NewDefinition()

	if def.Name != "pytest" {
		t.Errorf("expected Name='pytest', got '%s'", def.Name)
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
	if len(def.Matchers) != 5 {
		t.Errorf("expected 5 Matchers, got %d", len(def.Matchers))
	}
}

func TestPytestFileMatcher_Match(t *testing.T) {
	matcher := &PytestFileMatcher{}
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
		{"conftest.py", "conftest.py", 0},
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

func TestPytestContentMatcher_Match(t *testing.T) {
	matcher := &PytestContentMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		content            string
		expectedConfidence int
	}{
		{
			name: "pytest.mark.skip decorator",
			content: `
import pytest

@pytest.mark.skip(reason="not implemented")
def test_something():
    pass
`,
			expectedConfidence: 40,
		},
		{
			name: "pytest.fixture decorator",
			content: `
import pytest

@pytest.fixture
def my_fixture():
    return {"key": "value"}
`,
			expectedConfidence: 40,
		},
		{
			name: "pytest.raises usage",
			content: `
import pytest

def test_raises():
    with pytest.raises(ValueError):
        raise ValueError("expected")
`,
			expectedConfidence: 40,
		},
		{
			name: "pytest.mark.parametrize",
			content: `
import pytest

@pytest.mark.parametrize("x,y,expected", [(1, 2, 3), (2, 3, 5)])
def test_add(x, y, expected):
    assert x + y == expected
`,
			expectedConfidence: 40,
		},
		{
			name: "no pytest patterns",
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

func TestPytestConfigContentMatcher_Match(t *testing.T) {
	matcher := &PytestConfigContentMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		filename           string
		content            string
		expectedConfidence int
	}{
		{
			name:     "pyproject.toml with tool.pytest",
			filename: "pyproject.toml",
			content: `
[tool.pytest.ini_options]
minversion = "6.0"
addopts = "-ra -q"
testpaths = ["tests"]
`,
			expectedConfidence: 100,
		},
		{
			name:     "pyproject.toml without pytest",
			filename: "pyproject.toml",
			content: `
[tool.poetry]
name = "my-project"
version = "0.1.0"
`,
			expectedConfidence: 0,
		},
		{
			name:               "different config file",
			filename:           "setup.py",
			content:            "[tool.pytest]",
			expectedConfidence: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signal := framework.Signal{
				Type:    framework.SignalConfigFile,
				Value:   tt.filename,
				Context: []byte(tt.content),
			}

			result := matcher.Match(ctx, signal)

			if result.Confidence != tt.expectedConfidence {
				t.Errorf("expected Confidence=%d, got %d", tt.expectedConfidence, result.Confidence)
			}
		})
	}
}

func TestPytestParser_Parse(t *testing.T) {
	p := &PytestParser{}
	ctx := context.Background()

	t.Run("basic test functions", func(t *testing.T) {
		source := `
def test_add():
    assert 1 + 1 == 2

def test_subtract():
    assert 5 - 3 == 2

def helper_function():
    return 42
`
		testFile, err := p.Parse(ctx, []byte(source), "test_math.py")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if testFile.Path != "test_math.py" {
			t.Errorf("expected Path='test_math.py', got '%s'", testFile.Path)
		}
		if testFile.Framework != "pytest" {
			t.Errorf("expected Framework='pytest', got '%s'", testFile.Framework)
		}
		if testFile.Language != domain.LanguagePython {
			t.Errorf("expected Language=python, got '%s'", testFile.Language)
		}
		if len(testFile.Suites) != 0 {
			t.Errorf("expected 0 Suites, got %d", len(testFile.Suites))
		}
		if len(testFile.Tests) != 2 {
			t.Fatalf("expected 2 Tests, got %d", len(testFile.Tests))
		}
		if testFile.Tests[0].Name != "test_add" {
			t.Errorf("expected Tests[0].Name='test_add', got '%s'", testFile.Tests[0].Name)
		}
		if testFile.Tests[1].Name != "test_subtract" {
			t.Errorf("expected Tests[1].Name='test_subtract', got '%s'", testFile.Tests[1].Name)
		}
	})

	t.Run("test class with methods", func(t *testing.T) {
		source := `
class TestCalculator:
    def test_add(self):
        assert 1 + 1 == 2

    def test_multiply(self):
        assert 2 * 3 == 6

    def helper_method(self):
        pass
`
		testFile, err := p.Parse(ctx, []byte(source), "test_calculator.py")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
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
		if suite.Tests[1].Name != "test_multiply" {
			t.Errorf("expected Tests[1].Name='test_multiply', got '%s'", suite.Tests[1].Name)
		}
	})

	t.Run("skip decorator", func(t *testing.T) {
		source := `
import pytest

@pytest.mark.skip(reason="not implemented")
def test_skipped():
    pass

def test_normal():
    pass
`
		testFile, err := p.Parse(ctx, []byte(source), "test_skip.py")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Tests) != 2 {
			t.Fatalf("expected 2 Tests, got %d", len(testFile.Tests))
		}

		if testFile.Tests[0].Name != "test_skipped" {
			t.Errorf("expected Tests[0].Name='test_skipped', got '%s'", testFile.Tests[0].Name)
		}
		if testFile.Tests[0].Status != domain.TestStatusSkipped {
			t.Errorf("expected Tests[0].Status='skipped', got '%s'", testFile.Tests[0].Status)
		}

		if testFile.Tests[1].Name != "test_normal" {
			t.Errorf("expected Tests[1].Name='test_normal', got '%s'", testFile.Tests[1].Name)
		}
		if testFile.Tests[1].Status != domain.TestStatusActive {
			t.Errorf("expected Tests[1].Status='active', got '%s'", testFile.Tests[1].Status)
		}
	})

	t.Run("xfail decorator", func(t *testing.T) {
		source := `
import pytest

@pytest.mark.xfail(reason="known bug")
def test_xfail():
    assert False
`
		testFile, err := p.Parse(ctx, []byte(source), "test_xfail.py")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Tests) != 1 {
			t.Fatalf("expected 1 Test, got %d", len(testFile.Tests))
		}
		if testFile.Tests[0].Status != domain.TestStatusXfail {
			t.Errorf("expected Status='xfail', got '%s'", testFile.Tests[0].Status)
		}
	})

	t.Run("parametrize decorator", func(t *testing.T) {
		source := `
import pytest

@pytest.mark.parametrize("x,y,expected", [
    (1, 2, 3),
    (2, 3, 5),
])
def test_add(x, y, expected):
    assert x + y == expected
`
		testFile, err := p.Parse(ctx, []byte(source), "test_parametrize.py")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Tests) != 1 {
			t.Fatalf("expected 1 Test, got %d", len(testFile.Tests))
		}
		if testFile.Tests[0].Name != "test_add" {
			t.Errorf("expected Name='test_add', got '%s'", testFile.Tests[0].Name)
		}
	})

	t.Run("class with skip decorator", func(t *testing.T) {
		source := `
import pytest

@pytest.mark.skip(reason="class skipped")
class TestSkipped:
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
		// Methods inherit class status
		if suite.Tests[0].Status != domain.TestStatusSkipped {
			t.Errorf("expected Tests[0].Status='skipped', got '%s'", suite.Tests[0].Status)
		}
		if suite.Tests[1].Status != domain.TestStatusSkipped {
			t.Errorf("expected Tests[1].Status='skipped', got '%s'", suite.Tests[1].Status)
		}
	})

	t.Run("mixed functions and classes", func(t *testing.T) {
		source := `
def test_standalone():
    pass

class TestGroup:
    def test_in_class(self):
        pass

def test_another():
    pass
`
		testFile, err := p.Parse(ctx, []byte(source), "test_mixed.py")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Tests) != 2 {
			t.Fatalf("expected 2 Tests, got %d", len(testFile.Tests))
		}
		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		if testFile.Tests[0].Name != "test_standalone" {
			t.Errorf("expected Tests[0].Name='test_standalone', got '%s'", testFile.Tests[0].Name)
		}
		if testFile.Tests[1].Name != "test_another" {
			t.Errorf("expected Tests[1].Name='test_another', got '%s'", testFile.Tests[1].Name)
		}
		if testFile.Suites[0].Name != "TestGroup" {
			t.Errorf("expected Suites[0].Name='TestGroup', got '%s'", testFile.Suites[0].Name)
		}
	})
}
