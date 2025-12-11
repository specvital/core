package mocha

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
)

func TestNewDefinition(t *testing.T) {
	def := NewDefinition()

	if def.Name != "mocha" {
		t.Errorf("expected Name to be 'mocha', got %q", def.Name)
	}

	if def.Priority != framework.PriorityGeneric {
		t.Errorf("expected Priority to be %d, got %d", framework.PriorityGeneric, def.Priority)
	}

	expectedLangs := []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript}
	if len(def.Languages) != len(expectedLangs) {
		t.Errorf("expected %d languages, got %d", len(expectedLangs), len(def.Languages))
	}
	for i, lang := range def.Languages {
		if lang != expectedLangs[i] {
			t.Errorf("expected language %d to be %v, got %v", i, expectedLangs[i], lang)
		}
	}

	if def.ConfigParser == nil {
		t.Error("expected ConfigParser to be non-nil")
	}

	if def.Parser == nil {
		t.Error("expected Parser to be non-nil")
	}

	// ImportMatcher + ConfigMatcher + ContentMatcher
	if len(def.Matchers) != 3 {
		t.Errorf("expected 3 matchers, got %d", len(def.Matchers))
	}
}

func TestMochaContentMatcher_Match(t *testing.T) {
	matcher := &MochaContentMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		content            string
		expectedConfidence int
	}{
		{
			name: "this.timeout() pattern",
			content: `
describe('Slow tests', function() {
  this.timeout(5000);
  it('takes time', function() {
    // slow operation
  });
});
`,
			expectedConfidence: 40,
		},
		{
			name: "this.slow() pattern",
			content: `
describe('Performance', function() {
  this.slow(1000);
  it('should be fast', function() {
    // test
  });
});
`,
			expectedConfidence: 40,
		},
		{
			name: "this.retries() pattern",
			content: `
describe('Flaky tests', function() {
  this.retries(3);
  it('might fail', function() {
    // flaky test
  });
});
`,
			expectedConfidence: 40,
		},
		{
			name: "this.skip() pattern",
			content: `
describe('Conditional', function() {
  it('skips on condition', function() {
    if (process.env.SKIP) {
      this.skip();
    }
  });
});
`,
			expectedConfidence: 40,
		},
		{
			name: "this.currentTest pattern",
			content: `
describe('Test info', function() {
  beforeEach(function() {
    console.log(this.currentTest.title);
  });
  it('has test info', function() {});
});
`,
			expectedConfidence: 40,
		},
		{
			name: "mocha.setup() pattern",
			content: `
mocha.setup('bdd');
describe('Browser tests', function() {
  it('works', function() {});
});
`,
			expectedConfidence: 40,
		},
		{
			name: "mocha.run() pattern",
			content: `
describe('Tests', function() {
  it('works', function() {});
});
mocha.run();
`,
			expectedConfidence: 40,
		},
		{
			name: "no Mocha patterns (plain Jest)",
			content: `
import { describe, test, expect } from '@jest/globals';

describe('Calculator', () => {
  test('adds numbers', () => {
    expect(1 + 1).toBe(2);
  });
});
`,
			expectedConfidence: 0,
		},
		{
			name: "generic describe/it without Mocha patterns",
			content: `
describe('Basic tests', () => {
  it('should work', () => {
    expect(true).toBe(true);
  });
});
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
				t.Errorf("expected confidence %d, got %d", tt.expectedConfidence, result.Confidence)
			}

			if tt.expectedConfidence > 0 && len(result.Evidence) == 0 {
				t.Error("expected evidence for match")
			}
		})
	}
}

func TestMochaContentMatcher_NonContentSignal(t *testing.T) {
	matcher := &MochaContentMatcher{}
	ctx := context.Background()

	signal := framework.Signal{
		Type:  framework.SignalImport,
		Value: "mocha",
	}

	result := matcher.Match(ctx, signal)
	if result.Confidence != 0 {
		t.Errorf("expected confidence 0 for non-content signal, got %d", result.Confidence)
	}
}

func TestMochaConfigParser_Parse(t *testing.T) {
	tests := []struct {
		name                 string
		configContent        string
		configPath           string
		expectedGlobalsMode  bool
		expectedTestPatterns []string
	}{
		{
			name: "default config without spec",
			configContent: `
{
  "reporter": "spec",
  "timeout": 5000
}
`,
			configPath:           "/project/.mocharc.json",
			expectedGlobalsMode:  true,
			expectedTestPatterns: nil,
		},
		{
			name: "spec single string",
			configContent: `
{
  "spec": "test/**/*.spec.js"
}
`,
			configPath:           "/project/.mocharc.json",
			expectedGlobalsMode:  true,
			expectedTestPatterns: []string{"test/**/*.spec.js"},
		},
		{
			name: "spec array",
			configContent: `
{
  "spec": ["test/**/*.spec.js", "test/**/*.test.js"]
}
`,
			configPath:           "/project/.mocharc.json",
			expectedGlobalsMode:  true,
			expectedTestPatterns: []string{"test/**/*.spec.js", "test/**/*.test.js"},
		},
		{
			name: "yaml format single",
			configContent: `
spec: 'test/**/*.spec.js'
reporter: spec
`,
			configPath:           "/project/.mocharc.yaml",
			expectedGlobalsMode:  true,
			expectedTestPatterns: []string{"test/**/*.spec.js"},
		},
		{
			name: "js format",
			configContent: `
module.exports = {
  spec: 'test/**/*.test.js',
  timeout: 10000
};
`,
			configPath:           "/project/.mocharc.js",
			expectedGlobalsMode:  true,
			expectedTestPatterns: []string{"test/**/*.test.js"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &MochaConfigParser{}
			ctx := context.Background()

			scope, err := parser.Parse(ctx, tt.configPath, []byte(tt.configContent))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if scope.Framework != "mocha" {
				t.Errorf("expected framework 'mocha', got %q", scope.Framework)
			}

			if scope.ConfigPath != tt.configPath {
				t.Errorf("expected configPath %q, got %q", tt.configPath, scope.ConfigPath)
			}

			if scope.GlobalsMode != tt.expectedGlobalsMode {
				t.Errorf("expected globalsMode %v, got %v", tt.expectedGlobalsMode, scope.GlobalsMode)
			}

			if len(scope.TestPatterns) != len(tt.expectedTestPatterns) {
				t.Errorf("expected %d test patterns, got %d", len(tt.expectedTestPatterns), len(scope.TestPatterns))
			}
			for i, pattern := range tt.expectedTestPatterns {
				if i < len(scope.TestPatterns) && scope.TestPatterns[i] != pattern {
					t.Errorf("expected pattern %d to be %q, got %q", i, pattern, scope.TestPatterns[i])
				}
			}
		})
	}
}

func TestParseSpec(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "single quoted",
			content:  `"spec": 'test/**/*.js'`,
			expected: []string{"test/**/*.js"},
		},
		{
			name:     "double quoted",
			content:  `"spec": "test/**/*.js"`,
			expected: []string{"test/**/*.js"},
		},
		{
			name:     "array",
			content:  `"spec": ['a.js', 'b.js']`,
			expected: []string{"a.js", "b.js"},
		},
		{
			name:     "yaml format",
			content:  `spec: 'test/*.js'`,
			expected: []string{"test/*.js"},
		},
		{
			name:     "no spec",
			content:  `"reporter": "spec"`,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSpec([]byte(tt.content))

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d results, got %d", len(tt.expected), len(result))
				return
			}
			for i, v := range tt.expected {
				if result[i] != v {
					t.Errorf("expected result[%d] to be %q, got %q", i, v, result[i])
				}
			}
		})
	}
}

func TestMochaParser_Parse(t *testing.T) {
	testSource := `
describe('User Authentication', function() {
  this.timeout(10000);

  it('should login successfully', function() {
    // test code
  });

  it.skip('skipped test', function() {
    // skipped
  });

  describe('Password Reset', function() {
    it('should send reset email', function() {
      // test code
    });
  });
});

it('top-level test', function() {
  // test
});
`

	parser := &MochaParser{}
	ctx := context.Background()

	testFile, err := parser.Parse(ctx, []byte(testSource), "auth.test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if testFile.Path != "auth.test.js" {
		t.Errorf("expected Path 'auth.test.js', got %q", testFile.Path)
	}

	if testFile.Framework != "mocha" {
		t.Errorf("expected Framework 'mocha', got %q", testFile.Framework)
	}

	if testFile.Language != domain.LanguageJavaScript {
		t.Errorf("expected Language JavaScript, got %v", testFile.Language)
	}

	// Verify suite structure
	if len(testFile.Suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(testFile.Suites))
	}

	suite := testFile.Suites[0]
	if suite.Name != "User Authentication" {
		t.Errorf("expected suite name 'User Authentication', got %q", suite.Name)
	}

	if len(suite.Tests) != 2 {
		t.Errorf("expected 2 tests in suite, got %d", len(suite.Tests))
	}

	// Verify tests within suite
	if suite.Tests[0].Name != "should login successfully" {
		t.Errorf("expected first test name 'should login successfully', got %q", suite.Tests[0].Name)
	}
	if suite.Tests[0].Status != domain.TestStatusActive {
		t.Errorf("expected first test status Active, got %v", suite.Tests[0].Status)
	}

	if suite.Tests[1].Name != "skipped test" {
		t.Errorf("expected second test name 'skipped test', got %q", suite.Tests[1].Name)
	}
	if suite.Tests[1].Status != domain.TestStatusSkipped {
		t.Errorf("expected second test status Skipped, got %v", suite.Tests[1].Status)
	}

	// Verify nested suite
	if len(suite.Suites) != 1 {
		t.Fatalf("expected 1 nested suite, got %d", len(suite.Suites))
	}
	nestedSuite := suite.Suites[0]
	if nestedSuite.Name != "Password Reset" {
		t.Errorf("expected nested suite name 'Password Reset', got %q", nestedSuite.Name)
	}

	// Verify top-level test
	if len(testFile.Tests) != 1 {
		t.Fatalf("expected 1 top-level test, got %d", len(testFile.Tests))
	}
	if testFile.Tests[0].Name != "top-level test" {
		t.Errorf("expected top-level test name 'top-level test', got %q", testFile.Tests[0].Name)
	}
}

func TestMochaParser_ParseTypeScript(t *testing.T) {
	testSource := `
describe('API Client', () => {
  it('should fetch data', async () => {
    const data = await client.fetch();
    expect(data).toBeDefined();
  });
});
`

	parser := &MochaParser{}
	ctx := context.Background()

	testFile, err := parser.Parse(ctx, []byte(testSource), "api.test.ts")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if testFile.Path != "api.test.ts" {
		t.Errorf("expected Path 'api.test.ts', got %q", testFile.Path)
	}

	if testFile.Framework != "mocha" {
		t.Errorf("expected Framework 'mocha', got %q", testFile.Framework)
	}

	if testFile.Language != domain.LanguageTypeScript {
		t.Errorf("expected Language TypeScript, got %v", testFile.Language)
	}
}

func BenchmarkMochaContentMatcher_Match(b *testing.B) {
	matcher := &MochaContentMatcher{}
	ctx := context.Background()

	testContent := `
describe('Performance test', function() {
  this.timeout(5000);
  this.slow(1000);
  it('should be fast', function() {
    expect(true).toBe(true);
  });
});
`
	signal := framework.Signal{
		Type:    framework.SignalFileContent,
		Value:   testContent,
		Context: []byte(testContent),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.Match(ctx, signal)
	}
}
