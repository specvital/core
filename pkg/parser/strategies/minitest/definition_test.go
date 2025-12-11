package minitest

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
)

func TestMinitestParser_Parse(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		checkFunc func(t *testing.T, file *domain.TestFile)
	}{
		{
			name: "basic Minitest::Test class with test methods",
			source: `
require 'minitest/autorun'

class UserTest < Minitest::Test
  def test_creates_user
    user = User.new
    assert user.valid?
  end

  def test_validates_email
    user = User.new(email: "invalid")
    refute user.valid?
  end
end
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 suite, got %d", len(file.Suites))
					return
				}
				suite := file.Suites[0]
				if suite.Name != "UserTest" {
					t.Errorf("expected suite name 'UserTest', got %q", suite.Name)
				}
				if len(suite.Tests) != 2 {
					t.Errorf("expected 2 tests, got %d", len(suite.Tests))
					return
				}
				if suite.Tests[0].Name != "test_creates_user" {
					t.Errorf("expected test name 'test_creates_user', got %q", suite.Tests[0].Name)
				}
				if suite.Tests[1].Name != "test_validates_email" {
					t.Errorf("expected test name 'test_validates_email', got %q", suite.Tests[1].Name)
				}
			},
		},
		{
			name: "Minitest::Spec style with describe and it",
			source: `
require 'minitest/spec'
require 'minitest/autorun'

describe Calculator do
  describe "#add" do
    it "returns the sum" do
      Calculator.add(1, 2).must_equal 3
    end
  end
end
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 top-level suite, got %d", len(file.Suites))
					return
				}
				suite := file.Suites[0]
				if suite.Name != "Calculator" {
					t.Errorf("expected suite name 'Calculator', got %q", suite.Name)
				}
				if len(suite.Suites) != 1 {
					t.Errorf("expected 1 nested suite, got %d", len(suite.Suites))
					return
				}
				addSuite := suite.Suites[0]
				if addSuite.Name != "#add" {
					t.Errorf("expected nested suite name '#add', got %q", addSuite.Name)
				}
				if len(addSuite.Tests) != 1 {
					t.Errorf("expected 1 test, got %d", len(addSuite.Tests))
					return
				}
				if addSuite.Tests[0].Name != "returns the sum" {
					t.Errorf("expected test name 'returns the sum', got %q", addSuite.Tests[0].Name)
				}
			},
		},
		{
			name: "skipped test with skip method",
			source: `
class UserTest < Minitest::Test
  def test_skipped
    skip "not implemented yet"
    assert true
  end

  def test_active
    assert true
  end
end
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				suite := file.Suites[0]
				if len(suite.Tests) != 2 {
					t.Errorf("expected 2 tests, got %d", len(suite.Tests))
					return
				}

				skipped := suite.Tests[0]
				if skipped.Name != "test_skipped" {
					t.Errorf("expected test name 'test_skipped', got %q", skipped.Name)
				}
				if skipped.Status != domain.TestStatusSkipped {
					t.Errorf("expected status skipped, got %q", skipped.Status)
				}

				active := suite.Tests[1]
				if active.Status != domain.TestStatusActive {
					t.Errorf("expected active status, got %q", active.Status)
				}
			},
		},
		{
			name: "skip in spec style",
			source: `
describe "User" do
  it "is skipped" do
    skip
    true.must_equal true
  end
end
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				suite := file.Suites[0]
				if len(suite.Tests) != 1 {
					t.Errorf("expected 1 test, got %d", len(suite.Tests))
					return
				}
				if suite.Tests[0].Status != domain.TestStatusSkipped {
					t.Errorf("expected status skipped, got %q", suite.Tests[0].Status)
				}
			},
		},
		{
			name: "multiple tests in one class",
			source: `
class ArrayTest < Minitest::Test
  def test_empty
    assert [].empty?
  end

  def test_has_elements
    refute [1, 2].empty?
  end

  def test_length
    assert_equal 3, [1, 2, 3].length
  end
end
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				suite := file.Suites[0]
				if len(suite.Tests) != 3 {
					t.Errorf("expected 3 tests, got %d", len(suite.Tests))
				}
			},
		},
		{
			name: "nested describe blocks",
			source: `
describe Array do
  describe "when empty" do
    it "has zero length" do
      [].length.must_equal 0
    end
  end

  describe "when has elements" do
    it "returns correct length" do
      [1, 2, 3].length.must_equal 3
    end
  end
end
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 top-level suite, got %d", len(file.Suites))
					return
				}
				suite := file.Suites[0]
				if suite.Name != "Array" {
					t.Errorf("expected suite name 'Array', got %q", suite.Name)
				}
				if len(suite.Suites) != 2 {
					t.Errorf("expected 2 nested suites, got %d", len(suite.Suites))
					return
				}
				if suite.Suites[0].Name != "when empty" {
					t.Errorf("expected nested suite name 'when empty', got %q", suite.Suites[0].Name)
				}
				if suite.Suites[1].Name != "when has elements" {
					t.Errorf("expected nested suite name 'when has elements', got %q", suite.Suites[1].Name)
				}
			},
		},
		{
			name: "string description with double quotes",
			source: `
describe "String Utils" do
  it "handles strings" do
    "hello".must_equal "hello"
  end
end
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				suite := file.Suites[0]
				if suite.Name != "String Utils" {
					t.Errorf("expected suite name 'String Utils', got %q", suite.Name)
				}
			},
		},
		{
			name: "class inheriting from custom Test base",
			source: `
class MyTest < ActiveSupport::Test
  def test_something
    assert true
  end
end
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				// ActiveSupport::Test ends with Test, so should be recognized
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 suite, got %d", len(file.Suites))
					return
				}
				if len(file.Suites[0].Tests) != 1 {
					t.Errorf("expected 1 test, got %d", len(file.Suites[0].Tests))
				}
			},
		},
		{
			name: "anonymous it block without description",
			source: `
describe "Something" do
  it do
    assert true
  end
end
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 suite, got %d", len(file.Suites))
					return
				}
				suite := file.Suites[0]
				if len(suite.Tests) != 1 {
					t.Errorf("expected 1 test, got %d", len(suite.Tests))
					return
				}
				if suite.Tests[0].Name != "(anonymous)" {
					t.Errorf("expected test name '(anonymous)', got %q", suite.Tests[0].Name)
				}
			},
		},
		{
			name: "symbol description in describe",
			source: `
describe :user_authentication do
  it "works" do
    assert true
  end
end
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 suite, got %d", len(file.Suites))
					return
				}
				suite := file.Suites[0]
				if suite.Name != "user_authentication" {
					t.Errorf("expected suite name 'user_authentication', got %q", suite.Name)
				}
			},
		},
	}

	parser := &MinitestParser{}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := parser.Parse(ctx, []byte(tt.source), "test_example.rb")
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if file.Framework != frameworkName {
				t.Errorf("expected framework %q, got %q", frameworkName, file.Framework)
			}

			if file.Language != domain.LanguageRuby {
				t.Errorf("expected language Ruby, got %q", file.Language)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, file)
			}
		})
	}
}

func TestMinitestFileMatcher_Match(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		wantMatch  bool
		wantReason string
	}{
		{"test file", "user_test.rb", true, "*_test.rb"},
		{"test in directory", "test/models/user_test.rb", true, "*_test.rb"},
		{"file in test directory", "test/user.rb", true, "test/"},
		{"regular ruby file", "user.rb", false, ""},
		{"spec file", "user_spec.rb", false, ""},
	}

	matcher := &MinitestFileMatcher{}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signal := framework.Signal{
				Type:  framework.SignalFileName,
				Value: tt.filename,
			}
			result := matcher.Match(ctx, signal)
			if (result.Confidence > 0) != tt.wantMatch {
				t.Errorf("Match() = %v, want match = %v", result.Confidence > 0, tt.wantMatch)
			}
		})
	}
}

func TestMinitestContentMatcher_Match(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantMatch bool
	}{
		{"Minitest::Test class", "class UserTest < Minitest::Test", true},
		{"Minitest::Spec class", "class UserSpec < Minitest::Spec", true},
		{"def test_ method", "def test_creates_user", true},
		{"describe block", `describe User do`, true},
		{"it block", `it "works" do`, true},
		{"assert assertion", "assert user.valid?", true},
		{"assert_equal assertion", "assert_equal 1, result", true},
		{"refute assertion", "refute user.nil?", true},
		{"must_equal expectation", "result.must_equal 5", true},
		{"wont_be expectation", "result.wont_be :nil?", true},
		{"plain ruby code", "class User; end", false},
		// Note: RSpec.describe also matches because describe block is shared with Minitest::Spec
		// Framework detection relies on import/config matchers to distinguish them
	}

	matcher := &MinitestContentMatcher{}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signal := framework.Signal{
				Type:    framework.SignalFileContent,
				Value:   tt.content,
				Context: []byte(tt.content),
			}
			result := matcher.Match(ctx, signal)
			if (result.Confidence > 0) != tt.wantMatch {
				t.Errorf("Match() = %v, want match = %v", result.Confidence > 0, tt.wantMatch)
			}
		})
	}
}
