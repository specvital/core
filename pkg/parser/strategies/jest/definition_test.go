package jest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
)

func TestNewDefinition(t *testing.T) {
	def := NewDefinition()

	assert.Equal(t, "jest", def.Name)
	assert.Equal(t, framework.PriorityGeneric, def.Priority)
	assert.ElementsMatch(t,
		[]domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		def.Languages,
	)
	assert.NotNil(t, def.ConfigParser)
	assert.NotNil(t, def.Parser)
	assert.Len(t, def.Matchers, 3) // ImportMatcher + ConfigMatcher + ContentMatcher
}

func TestJestContentMatcher_Match(t *testing.T) {
	matcher := &JestContentMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		content            string
		expectedConfidence int
		shouldMatch        bool
	}{
		{
			name: "jest.fn() pattern",
			content: `
import { describe, test } from '@jest/globals';

describe('MyClass', () => {
  test('should call method', () => {
    const mockFn = jest.fn();
    mockFn();
    expect(mockFn).toHaveBeenCalled();
  });
});
`,
			expectedConfidence: 40,
			shouldMatch:        true,
		},
		{
			name: "jest.mock() pattern",
			content: `
jest.mock('./myModule', () => ({
  myFunction: jest.fn()
}));

describe('tests', () => {
  test('with mock', () => {
    // test code
  });
});
`,
			expectedConfidence: 40,
			shouldMatch:        true,
		},
		{
			name: "jest.spyOn() pattern",
			content: `
describe('MyClass', () => {
  test('should spy on method', () => {
    const spy = jest.spyOn(object, 'method');
    expect(spy).toHaveBeenCalled();
  });
});
`,
			expectedConfidence: 40,
			shouldMatch:        true,
		},
		{
			name: "jest.useFakeTimers() pattern",
			content: `
beforeEach(() => {
  jest.useFakeTimers();
});

test('timer test', () => {
  jest.advanceTimersByTime(1000);
});
`,
			expectedConfidence: 40,
			shouldMatch:        true,
		},
		{
			name: "jest.clearAllMocks() pattern",
			content: `
afterEach(() => {
  jest.clearAllMocks();
});

test('mock test', () => {
  // test code
});
`,
			expectedConfidence: 40,
			shouldMatch:        true,
		},
		{
			name: "no jest patterns",
			content: `
import { describe, test, expect } from 'vitest';

describe('Calculator', () => {
  test('adds numbers', () => {
    expect(1 + 1).toBe(2);
  });
});
`,
			expectedConfidence: 0,
			shouldMatch:        false,
		},
		{
			name: "jest in comment (weak signal)",
			content: `
// This test uses jest.fn() for mocking
describe('tests', () => {
  test('test case', () => {
    expect(true).toBe(true);
  });
});
`,
			expectedConfidence: 40,
			shouldMatch:        true,
			// Note: ContentMatcher uses simple regex and doesn't parse comments
			// This is acceptable as it provides a weak signal (40 confidence)
			// Real detection will combine multiple signals
		},
		{
			name: "jest as part of variable name (should not match)",
			content: `
const jestConfig = { testMatch: ['**/*.test.js'] };

describe('tests', () => {
  test('test case', () => {
    expect(true).toBe(true);
  });
});
`,
			expectedConfidence: 0,
			shouldMatch:        false,
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

			if tt.shouldMatch {
				assert.Equal(t, tt.expectedConfidence, result.Confidence, "Expected confidence mismatch")
				assert.NotEmpty(t, result.Evidence, "Expected evidence for match")
			} else {
				assert.Equal(t, 0, result.Confidence, "Should not match")
			}
		})
	}
}

func TestJestContentMatcher_NonContentSignal(t *testing.T) {
	matcher := &JestContentMatcher{}
	ctx := context.Background()

	// Should not match non-content signals
	signal := framework.Signal{
		Type:  framework.SignalImport,
		Value: "@jest/globals",
	}

	result := matcher.Match(ctx, signal)
	assert.Equal(t, 0, result.Confidence)
}

func TestJestConfigParser_ParseInjectGlobals(t *testing.T) {
	tests := []struct {
		name                string
		configContent       string
		expectedGlobalsMode bool
	}{
		{
			name: "injectGlobals not set (default true)",
			configContent: `
module.exports = {
  testEnvironment: 'node',
  testMatch: ['**/*.test.js']
};
`,
			expectedGlobalsMode: true,
		},
		{
			name: "injectGlobals explicitly false",
			configContent: `
module.exports = {
  testEnvironment: 'node',
  injectGlobals: false,
  testMatch: ['**/*.test.js']
};
`,
			expectedGlobalsMode: false,
		},
		{
			name: "injectGlobals explicitly true",
			configContent: `
module.exports = {
  testEnvironment: 'node',
  injectGlobals: true,
  testMatch: ['**/*.test.js']
};
`,
			expectedGlobalsMode: true,
		},
		{
			name: "injectGlobals false with spaces",
			configContent: `
module.exports = {
  injectGlobals  :  false
};
`,
			expectedGlobalsMode: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &JestConfigParser{}
			ctx := context.Background()

			scope, err := parser.Parse(ctx, "/project/jest.config.js", []byte(tt.configContent))

			require.NoError(t, err)
			assert.Equal(t, "jest", scope.Framework)
			assert.Equal(t, tt.expectedGlobalsMode, scope.GlobalsMode)
		})
	}
}

func TestJestConfigParser_ParseRoot(t *testing.T) {
	tests := []struct {
		name            string
		configContent   string
		configPath      string
		expectedRoot    string
		expectedBaseDir string
	}{
		{
			name: "rootDir with parent directory",
			configContent: `
module.exports = {
  rootDir: "..",
  testEnvironment: 'node'
};
`,
			configPath:      "/project/apps/web/jest.config.js",
			expectedRoot:    "..",
			expectedBaseDir: "/project/apps",
		},
		{
			name: "rootDir with explicit path",
			configContent: `
module.exports = {
  rootDir: "src",
  testEnvironment: 'node'
};
`,
			configPath:      "/project/jest.config.js",
			expectedRoot:    "src",
			expectedBaseDir: "/project/src",
		},
		{
			name: "no rootDir specified",
			configContent: `
module.exports = {
  testEnvironment: 'node'
};
`,
			configPath:      "/project/jest.config.js",
			expectedRoot:    "",
			expectedBaseDir: "/project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &JestConfigParser{}
			ctx := context.Background()

			scope, err := parser.Parse(ctx, tt.configPath, []byte(tt.configContent))

			require.NoError(t, err)
			assert.Equal(t, "jest", scope.Framework)
			assert.Equal(t, tt.configPath, scope.ConfigPath)

			// Normalize paths for comparison (handles OS path separators)
			expectedBaseDir := filepath.Clean(tt.expectedBaseDir)
			actualBaseDir := filepath.Clean(scope.BaseDir)
			assert.Equal(t, expectedBaseDir, actualBaseDir, "BaseDir mismatch")
		})
	}
}

func TestJestParser_Parse(t *testing.T) {
	testSource := `
import { describe, test, expect } from '@jest/globals';

describe('Calculator', () => {
  test('adds two numbers', () => {
    expect(1 + 1).toBe(2);
  });

  test.skip('skipped test', () => {
    expect(true).toBe(false);
  });
});

test('top-level test', () => {
  expect(true).toBe(true);
});
`

	parser := &JestParser{}
	ctx := context.Background()

	testFile, err := parser.Parse(ctx, []byte(testSource), "calculator.test.ts")

	require.NoError(t, err)
	assert.Equal(t, "calculator.test.ts", testFile.Path)
	assert.Equal(t, "jest", testFile.Framework)
	assert.Equal(t, domain.LanguageTypeScript, testFile.Language)

	// Verify suite structure
	require.Len(t, testFile.Suites, 1)
	suite := testFile.Suites[0]
	assert.Equal(t, "Calculator", suite.Name)
	assert.Len(t, suite.Tests, 2)

	// Verify tests within suite
	assert.Equal(t, "adds two numbers", suite.Tests[0].Name)
	assert.Equal(t, domain.TestStatusActive, suite.Tests[0].Status)

	assert.Equal(t, "skipped test", suite.Tests[1].Name)
	assert.Equal(t, domain.TestStatusSkipped, suite.Tests[1].Status)

	// Verify top-level test
	require.Len(t, testFile.Tests, 1)
	assert.Equal(t, "top-level test", testFile.Tests[0].Name)
}

func TestParseRoot(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "single quotes",
			content:  `rootDir: '..'`,
			expected: "..",
		},
		{
			name:     "double quotes",
			content:  `rootDir: ".."`,
			expected: "..",
		},
		{
			name:     "src directory",
			content:  `rootDir: "src"`,
			expected: "src",
		},
		{
			name:     "no rootDir",
			content:  `testEnvironment: 'node'`,
			expected: "",
		},
		{
			name:     "rootDir with spaces",
			content:  `rootDir  :  "src"`,
			expected: "src",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRoot([]byte(tt.content))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseInjectGlobalsFalse(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "injectGlobals false",
			content:  `injectGlobals: false`,
			expected: true,
		},
		{
			name:     "injectGlobals true (should return false)",
			content:  `injectGlobals: true`,
			expected: false,
		},
		{
			name:     "no injectGlobals (should return false)",
			content:  `testEnvironment: 'node'`,
			expected: false,
		},
		{
			name:     "injectGlobals false with spaces",
			content:  `injectGlobals  :  false`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInjectGlobalsFalse([]byte(tt.content))
			assert.Equal(t, tt.expected, result)
		})
	}
}
