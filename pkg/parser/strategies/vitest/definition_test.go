package vitest

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

	assert.Equal(t, "vitest", def.Name)
	assert.Equal(t, framework.PrioritySpecialized, def.Priority)
	assert.ElementsMatch(t,
		[]domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		def.Languages,
	)
	assert.NotNil(t, def.ConfigParser)
	assert.NotNil(t, def.Parser)
	assert.Len(t, def.Matchers, 3) // ImportMatcher + ConfigMatcher + ContentMatcher
}

func TestVitestConfigParser_ParseRoot(t *testing.T) {
	tests := []struct {
		name            string
		configContent   string
		configPath      string
		expectedRoot    string
		expectedBaseDir string
	}{
		{
			name: "root with parent directory",
			configContent: `
export default defineConfig({
  root: "..",
  test: {
    globals: true
  }
})
`,
			configPath:      "/project/apps/web/vitest.config.ts",
			expectedRoot:    "..",
			expectedBaseDir: "/project/apps",
		},
		{
			name: "root with explicit path",
			configContent: `
export default defineConfig({
  root: "src",
  test: {
    globals: false
  }
})
`,
			configPath:      "/project/vitest.config.ts",
			expectedRoot:    "src",
			expectedBaseDir: "/project/src",
		},
		{
			name: "no root specified",
			configContent: `
export default defineConfig({
  test: {
    globals: true
  }
})
`,
			configPath:      "/project/vitest.config.ts",
			expectedRoot:    "",
			expectedBaseDir: "/project",
		},
		{
			name: "root with double quotes",
			configContent: `
export default defineConfig({
  root: "../..",
  test: {
    include: ["**/*.test.ts"]
  }
})
`,
			configPath:      "/project/apps/web/vitest.config.ts",
			expectedRoot:    "../..",
			expectedBaseDir: "/project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &VitestConfigParser{}
			ctx := context.Background()

			scope, err := parser.Parse(ctx, tt.configPath, []byte(tt.configContent))

			require.NoError(t, err)
			assert.Equal(t, "vitest", scope.Framework)
			assert.Equal(t, tt.configPath, scope.ConfigPath)

			// Normalize paths for comparison (handles OS path separators)
			expectedBaseDir := filepath.Clean(tt.expectedBaseDir)
			actualBaseDir := filepath.Clean(scope.BaseDir)
			assert.Equal(t, expectedBaseDir, actualBaseDir, "BaseDir mismatch")
		})
	}
}

func TestVitestConfigParser_ParseGlobals(t *testing.T) {
	tests := []struct {
		name                string
		configContent       string
		expectedGlobalsMode bool
	}{
		{
			name: "globals enabled",
			configContent: `
export default defineConfig({
  test: {
    globals: true
  }
})
`,
			expectedGlobalsMode: true,
		},
		{
			name: "globals disabled",
			configContent: `
export default defineConfig({
  test: {
    globals: false
  }
})
`,
			expectedGlobalsMode: false,
		},
		{
			name: "no globals setting",
			configContent: `
export default defineConfig({
  test: {
    include: ["**/*.test.ts"]
  }
})
`,
			expectedGlobalsMode: false,
		},
		{
			name: "globals in comment (should not match)",
			configContent: `
export default defineConfig({
  test: {
    // globals: true
    globals: false
  }
})
`,
			expectedGlobalsMode: false,
		},
		{
			name: "globals true in comment (should be ignored)",
			configContent: `
export default defineConfig({
  test: {
    /*
     * globals: true
     */
  }
})
`,
			expectedGlobalsMode: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &VitestConfigParser{}
			ctx := context.Background()

			scope, err := parser.Parse(ctx, "/project/vitest.config.ts", []byte(tt.configContent))

			require.NoError(t, err)
			assert.Equal(t, tt.expectedGlobalsMode, scope.GlobalsMode)
		})
	}
}

func TestVitestConfigParser_ComplexConfig(t *testing.T) {
	// Test the specific bug case from the issue description
	configContent := `
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

export default defineConfig({
  root: "..",
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    include: ['src/**/*.{test,spec}.{js,ts,jsx,tsx}'],
    exclude: ['**/node_modules/**', '**/dist/**'],
  },
});
`

	parser := &VitestConfigParser{}
	ctx := context.Background()
	configPath := "/project/apps/web/vitest.config.ts"

	scope, err := parser.Parse(ctx, configPath, []byte(configContent))

	require.NoError(t, err)
	assert.Equal(t, "vitest", scope.Framework)
	assert.Equal(t, configPath, scope.ConfigPath)
	assert.True(t, scope.GlobalsMode)

	// Verify root resolution: config at /project/apps/web, root: ".."
	// Should resolve to /project/apps
	expectedBaseDir := filepath.Clean("/project/apps")
	actualBaseDir := filepath.Clean(scope.BaseDir)
	assert.Equal(t, expectedBaseDir, actualBaseDir)
}

func TestVitestParser_Parse(t *testing.T) {
	testSource := `
import { describe, test, expect } from 'vitest';

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

	parser := &VitestParser{}
	ctx := context.Background()

	testFile, err := parser.Parse(ctx, []byte(testSource), "calculator.test.ts")

	require.NoError(t, err)
	assert.Equal(t, "calculator.test.ts", testFile.Path)
	assert.Equal(t, "vitest", testFile.Framework)
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
			content:  `root: '..'`,
			expected: "..",
		},
		{
			name:     "double quotes",
			content:  `root: ".."`,
			expected: "..",
		},
		{
			name:     "src directory",
			content:  `root: "src"`,
			expected: "src",
		},
		{
			name:     "no root",
			content:  `test: { globals: true }`,
			expected: "",
		},
		{
			name:     "root with spaces",
			content:  `root  :  "src"`,
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

func TestParseGlobals(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "globals true",
			content:  `globals: true`,
			expected: true,
		},
		{
			name:     "globals false",
			content:  `globals: false`,
			expected: false,
		},
		{
			name:     "no globals",
			content:  `test: { include: ["**/*.test.ts"] }`,
			expected: false,
		},
		{
			name:     "globals with spaces",
			content:  `globals  :  true`,
			expected: true,
		},
		{
			name: "globals in comment",
			content: `
// globals: true
globals: false
`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result := parseGlobals(ctx, []byte(tt.content))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVitestContentMatcher_Match(t *testing.T) {
	matcher := &VitestContentMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		content            string
		expectedConfidence int
		shouldMatch        bool
	}{
		{
			name:               "vi.fn pattern",
			content:            `const mock = vi.fn()`,
			expectedConfidence: 40,
			shouldMatch:        true,
		},
		{
			name:               "vi.mock pattern",
			content:            `vi.mock('./module')`,
			expectedConfidence: 40,
			shouldMatch:        true,
		},
		{
			name:               "vi.spyOn pattern",
			content:            `vi.spyOn(object, 'method')`,
			expectedConfidence: 40,
			shouldMatch:        true,
		},
		{
			name:               "vi.useFakeTimers pattern",
			content:            `vi.useFakeTimers()`,
			expectedConfidence: 40,
			shouldMatch:        true,
		},
		{
			name:               "vi.clearAllMocks pattern",
			content:            `vi.clearAllMocks()`,
			expectedConfidence: 40,
			shouldMatch:        true,
		},
		{
			name:               "vi.resetAllMocks pattern",
			content:            `vi.resetAllMocks()`,
			expectedConfidence: 40,
			shouldMatch:        true,
		},
		{
			name:               "vi.restoreAllMocks pattern",
			content:            `vi.restoreAllMocks()`,
			expectedConfidence: 40,
			shouldMatch:        true,
		},
		{
			name:               "vi.stubGlobal pattern",
			content:            `vi.stubGlobal('fetch', mockFetch)`,
			expectedConfidence: 40,
			shouldMatch:        true,
		},
		{
			name:               "vi.stubEnv pattern",
			content:            `vi.stubEnv('NODE_ENV', 'test')`,
			expectedConfidence: 40,
			shouldMatch:        true,
		},
		{
			name: "complex test file with vi patterns",
			content: `
import { describe, it, expect } from 'vitest';

describe('UserService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should fetch user', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ id: 1 });
    vi.stubGlobal('fetch', mockFetch);

    const result = await fetchUser(1);
    expect(result.id).toBe(1);
  });
});
`,
			expectedConfidence: 40,
			shouldMatch:        true,
		},
		{
			name:               "no vitest patterns",
			content:            `const x = 1; function test() {}`,
			expectedConfidence: 0,
			shouldMatch:        false,
		},
		{
			name:               "jest patterns should not match",
			content:            `jest.fn(); jest.mock('./module')`,
			expectedConfidence: 0,
			shouldMatch:        false,
		},
		{
			name: "globals mode test without vi patterns",
			content: `
describe('Calculator', () => {
  it('adds numbers', () => {
    expect(1 + 1).toBe(2);
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
				Value:   "",
				Context: []byte(tt.content),
			}

			result := matcher.Match(ctx, signal)

			if tt.shouldMatch {
				assert.Equal(t, tt.expectedConfidence, result.Confidence)
				assert.False(t, result.Negative)
			} else {
				assert.Equal(t, 0, result.Confidence)
			}
		})
	}
}

func TestVitestContentMatcher_NonContentSignal(t *testing.T) {
	matcher := &VitestContentMatcher{}
	ctx := context.Background()

	signal := framework.Signal{
		Type:  framework.SignalImport,
		Value: "vitest",
	}

	result := matcher.Match(ctx, signal)
	assert.Equal(t, 0, result.Confidence)
}

func TestConfigScope_Contains(t *testing.T) {
	// Test that ConfigScope correctly determines file containment with root setting
	// Config at /project/apps/web/vitest.config.ts with root: ".."
	// Should resolve to BaseDir: /project/apps
	configPath := "/project/apps/web/vitest.config.ts"
	root := ".."
	scope := framework.NewConfigScope(configPath, root)

	// Verify base dir is correctly computed
	expectedBaseDir := filepath.Clean("/project/apps")
	actualBaseDir := filepath.Clean(scope.BaseDir)
	assert.Equal(t, expectedBaseDir, actualBaseDir, "BaseDir should be /project/apps")

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "file within base dir - web subdirectory",
			filePath: "/project/apps/web/src/utils.test.ts",
			expected: true,
		},
		{
			name:     "file in sibling directory (within parent)",
			filePath: "/project/apps/api/src/handler.test.ts",
			expected: true,
		},
		{
			name:     "file outside base dir",
			filePath: "/other-project/test.ts",
			expected: false,
		},
		{
			name:     "file at base dir level",
			filePath: "/project/apps/shared.test.ts",
			expected: true,
		},
		{
			name:     "file in grandparent (outside base dir)",
			filePath: "/project/shared.test.ts",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scope.Contains(tt.filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}
