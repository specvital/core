package playwright

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
)

func TestNewDefinition(t *testing.T) {
	def := NewDefinition()

	assert.Equal(t, "playwright", def.Name)
	assert.Equal(t, framework.PriorityE2E, def.Priority)
	assert.ElementsMatch(t,
		[]domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript},
		def.Languages,
	)
	assert.NotNil(t, def.ConfigParser)
	assert.NotNil(t, def.Parser)
	assert.Len(t, def.Matchers, 2) // ImportMatcher + ConfigMatcher
}

func TestPlaywrightConfigParser_Parse(t *testing.T) {
	tests := []struct {
		name                string
		configContent       string
		configPath          string
		expectedGlobalsMode bool
		expectedBaseDir     string
	}{
		{
			name: "basic playwright config",
			configContent: `
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
`,
			configPath:          "/project/playwright.config.ts",
			expectedGlobalsMode: false,
			expectedBaseDir:     "/project/tests",
		},
		{
			name: "playwright config with multiple projects",
			configContent: `
export default defineConfig({
  testDir: './e2e',
  projects: [
    { name: 'chrome' },
    { name: 'firefox' },
  ],
});
`,
			configPath:          "/project/apps/web/playwright.config.ts",
			expectedGlobalsMode: false,
			expectedBaseDir:     "/project/apps/web/e2e",
		},
		{
			name:                "minimal config",
			configContent:       `export default defineConfig({});`,
			configPath:          "/project/playwright.config.js",
			expectedGlobalsMode: false,
			expectedBaseDir:     "/project",
		},
		{
			name: "testDirRoot variable (grafana pattern)",
			configContent: `
export const testDirRoot = 'e2e-playwright';

export default defineConfig({
  projects: [
    { name: 'admin', testDir: path.join(testDirRoot, '/admin') },
  ],
});
`,
			configPath:          "/project/playwright.config.ts",
			expectedGlobalsMode: false,
			expectedBaseDir:     "/project/e2e-playwright",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &PlaywrightConfigParser{}
			ctx := context.Background()

			scope, err := parser.Parse(ctx, tt.configPath, []byte(tt.configContent))

			require.NoError(t, err)
			assert.Equal(t, "playwright", scope.Framework)
			assert.Equal(t, tt.configPath, scope.ConfigPath)
			assert.Equal(t, tt.expectedGlobalsMode, scope.GlobalsMode, "Playwright should never use globals mode")
			assert.Equal(t, tt.expectedBaseDir, scope.BaseDir)
		})
	}
}

func TestPlaywrightConfigParser_Projects(t *testing.T) {
	tests := []struct {
		name             string
		configContent    string
		configPath       string
		expectedProjects int
		expectedNames    []string
		expectedBaseDirs []string
	}{
		{
			name: "projects with testDir string literal",
			configContent: `
export default defineConfig({
  projects: [
    {
      name: 'admin',
      testDir: './e2e/admin',
    },
    {
      name: 'viewer',
      testDir: './e2e/viewer',
    },
  ],
});
`,
			configPath:       "/project/playwright.config.ts",
			expectedProjects: 2,
			expectedNames:    []string{"admin", "viewer"},
			expectedBaseDirs: []string{"/project/e2e/admin", "/project/e2e/viewer"},
		},
		{
			name: "projects with path.join testDir",
			configContent: `
const testDirRoot = 'e2e/plugin-e2e/';

export default defineConfig({
  projects: [
    {
      name: 'api-admin',
      testDir: path.join(testDirRoot, '/api-tests/as-admin-user'),
    },
    {
      name: 'api-viewer',
      testDir: path.join(testDirRoot, '/api-tests/as-viewer-user'),
    },
  ],
});
`,
			configPath:       "/project/playwright.config.ts",
			expectedProjects: 2,
			expectedNames:    []string{"api-admin", "api-viewer"},
			expectedBaseDirs: []string{"/project/api-tests/as-admin-user", "/project/api-tests/as-viewer-user"},
		},
		{
			name: "projects without testDir should be ignored",
			configContent: `
export default defineConfig({
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
    },
    {
      name: 'with-testdir',
      testDir: './tests',
    },
  ],
});
`,
			configPath:       "/project/playwright.config.ts",
			expectedProjects: 1,
			expectedNames:    []string{"with-testdir"},
			expectedBaseDirs: []string{"/project/tests"},
		},
		{
			name: "no projects array",
			configContent: `
export default defineConfig({
  testDir: './tests',
});
`,
			configPath:       "/project/playwright.config.ts",
			expectedProjects: 0,
			expectedNames:    nil,
			expectedBaseDirs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &PlaywrightConfigParser{}
			ctx := context.Background()

			scope, err := parser.Parse(ctx, tt.configPath, []byte(tt.configContent))

			require.NoError(t, err)
			assert.Len(t, scope.Projects, tt.expectedProjects)

			if tt.expectedProjects > 0 {
				for i, proj := range scope.Projects {
					if i < len(tt.expectedNames) {
						assert.Equal(t, tt.expectedNames[i], proj.Name)
					}
					if i < len(tt.expectedBaseDirs) {
						assert.Equal(t, tt.expectedBaseDirs[i], proj.BaseDir)
					}
				}
			}
		})
	}
}

func TestPlaywrightParser_Parse(t *testing.T) {
	testSource := `
import { test, expect } from '@playwright/test';

test.describe('Login page', () => {
  test('should allow user to login', async ({ page }) => {
    await page.goto('https://example.com/login');
    await page.fill('#username', 'user');
    await page.fill('#password', 'pass');
    await page.click('#submit');
    await expect(page).toHaveURL(/dashboard/);
  });

  test.skip('skipped test', async ({ page }) => {
    // This test is skipped
  });

  test.fixme('broken test', async ({ page }) => {
    // This test needs fixing
  });
});

test('top-level test', async ({ page }) => {
  await page.goto('https://example.com');
  await expect(page).toHaveTitle(/Example/);
});
`

	parser := &PlaywrightParser{}
	ctx := context.Background()

	testFile, err := parser.Parse(ctx, []byte(testSource), "login.spec.ts")

	require.NoError(t, err)
	assert.Equal(t, "login.spec.ts", testFile.Path)
	assert.Equal(t, "playwright", testFile.Framework)
	assert.Equal(t, domain.LanguageTypeScript, testFile.Language)

	// Verify suite structure
	require.Len(t, testFile.Suites, 1)
	suite := testFile.Suites[0]
	assert.Equal(t, "Login page", suite.Name)
	assert.Len(t, suite.Tests, 3)

	// Verify tests within suite
	assert.Equal(t, "should allow user to login", suite.Tests[0].Name)
	assert.Equal(t, domain.TestStatusActive, suite.Tests[0].Status)

	assert.Equal(t, "skipped test", suite.Tests[1].Name)
	assert.Equal(t, domain.TestStatusSkipped, suite.Tests[1].Status)

	assert.Equal(t, "broken test", suite.Tests[2].Name)
	assert.Equal(t, domain.TestStatusSkipped, suite.Tests[2].Status)

	// Verify top-level test
	require.Len(t, testFile.Tests, 1)
	assert.Equal(t, "top-level test", testFile.Tests[0].Name)
}

func TestPlaywrightParser_NestedDescribe(t *testing.T) {
	testSource := `
import { test } from '@playwright/test';

test.describe('Parent suite', () => {
  test.describe('Child suite', () => {
    test('nested test', async ({ page }) => {
      // test code
    });
  });
});
`

	parser := &PlaywrightParser{}
	ctx := context.Background()

	testFile, err := parser.Parse(ctx, []byte(testSource), "nested.spec.ts")

	require.NoError(t, err)
	assert.Equal(t, "playwright", testFile.Framework)

	// Verify parent suite
	require.Len(t, testFile.Suites, 1)
	parentSuite := testFile.Suites[0]
	assert.Equal(t, "Parent suite", parentSuite.Name)

	// Verify child suite
	require.Len(t, parentSuite.Suites, 1)
	childSuite := parentSuite.Suites[0]
	assert.Equal(t, "Child suite", childSuite.Name)

	// Verify nested test
	require.Len(t, childSuite.Tests, 1)
	assert.Equal(t, "nested test", childSuite.Tests[0].Name)
}

func TestPlaywrightParser_Modifiers(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		expectedStatus domain.TestStatus
	}{
		{
			name: "test.skip",
			source: `
import { test } from '@playwright/test';
test.skip('skipped', async ({ page }) => {});
`,
			expectedStatus: domain.TestStatusSkipped,
		},
		{
			name: "test.only",
			source: `
import { test } from '@playwright/test';
test.only('focused', async ({ page }) => {});
`,
			expectedStatus: domain.TestStatusFocused,
		},
		{
			name: "test.fixme",
			source: `
import { test } from '@playwright/test';
test.fixme('needs fix', async ({ page }) => {});
`,
			expectedStatus: domain.TestStatusSkipped,
		},
		{
			name: "test.describe.skip",
			source: `
import { test } from '@playwright/test';
test.describe.skip('skipped suite', () => {
  test('test', async ({ page }) => {});
});
`,
			expectedStatus: domain.TestStatusSkipped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &PlaywrightParser{}
			ctx := context.Background()

			testFile, err := parser.Parse(ctx, []byte(tt.source), "modifier.spec.ts")

			require.NoError(t, err)

			// Check status (either on test or suite)
			if len(testFile.Tests) > 0 {
				assert.Equal(t, tt.expectedStatus, testFile.Tests[0].Status)
			} else if len(testFile.Suites) > 0 {
				assert.Equal(t, tt.expectedStatus, testFile.Suites[0].Status)
			} else {
				t.Fatal("No tests or suites found")
			}
		})
	}
}

func TestPlaywrightParser_SetupAlias(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		expectedName string
	}{
		{
			name: "setup alias from test",
			source: `
import { test as setup } from '@playwright/test';

setup("authenticate", async ({ request }) => {
  await request.post('/api/login');
});
`,
			expectedName: "authenticate",
		},
		{
			name: "teardown alias from test",
			source: `
import { test as teardown } from '@playwright/test';

teardown("cleanup", async ({ page }) => {
  await page.close();
});
`,
			expectedName: "cleanup",
		},
		{
			name: "multiple aliases",
			source: `
import { test, test as setup, expect } from '@playwright/test';

setup("auth setup", async ({ request }) => {});
test("regular test", async ({ page }) => {});
`,
			expectedName: "auth setup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &PlaywrightParser{}
			ctx := context.Background()

			testFile, err := parser.Parse(ctx, []byte(tt.source), "setup.spec.ts")

			require.NoError(t, err)
			require.NotEmpty(t, testFile.Tests, "Expected at least one test to be detected")
			assert.Equal(t, tt.expectedName, testFile.Tests[0].Name)
		})
	}

	t.Run("multiple aliases detects all tests", func(t *testing.T) {
		source := `
import { test, test as setup, expect } from '@playwright/test';

setup("auth setup", async ({ request }) => {});
test("regular test", async ({ page }) => {});
`
		parser := &PlaywrightParser{}
		ctx := context.Background()

		testFile, err := parser.Parse(ctx, []byte(source), "setup.spec.ts")

		require.NoError(t, err)
		require.Len(t, testFile.Tests, 2, "Both tests should be detected")
		assert.Equal(t, "auth setup", testFile.Tests[0].Name)
		assert.Equal(t, "regular test", testFile.Tests[1].Name)
	})
}

func TestPlaywrightParser_NonPlaywrightAlias(t *testing.T) {
	t.Run("alias from non-playwright import should not be detected", func(t *testing.T) {
		source := `
import { test as customTest } from './custom-utils';
import { test } from '@playwright/test';

customTest("should not be detected", async () => {});
test("should be detected", async ({ page }) => {});
`
		parser := &PlaywrightParser{}
		ctx := context.Background()

		testFile, err := parser.Parse(ctx, []byte(source), "mixed.spec.ts")

		require.NoError(t, err)
		require.Len(t, testFile.Tests, 1, "Only Playwright test should be detected")
		assert.Equal(t, "should be detected", testFile.Tests[0].Name)
	})
}

func TestPlaywrightParser_ConditionalSkip(t *testing.T) {
	t.Run("conditional skip in describe should NOT be counted as test", func(t *testing.T) {
		source := `
import { test } from '@playwright/test';

test.describe('Mobile', () => {
  test.skip(templateName?.includes('ssv6') || false, 'Skip mobile UI tests for SSV6');
  test.use({ viewport: { width: 390, height: 844 } });

  test('Navigate to story', async ({ page }) => {
    await page.goto('https://example.com');
  });
});
`
		parser := &PlaywrightParser{}
		ctx := context.Background()

		testFile, err := parser.Parse(ctx, []byte(source), "manager.spec.ts")

		require.NoError(t, err)
		require.Len(t, testFile.Suites, 1, "Should have one suite")
		assert.Equal(t, "Mobile", testFile.Suites[0].Name)
		require.Len(t, testFile.Suites[0].Tests, 1, "Conditional skip should NOT be counted")
		assert.Equal(t, "Navigate to story", testFile.Suites[0].Tests[0].Name)
	})

	t.Run("conditional skip inside test should NOT create extra test", func(t *testing.T) {
		source := `
import { test } from '@playwright/test';

test('Story context menu actions', async ({ page }) => {
  test.skip(type !== 'dev', 'These actions are only applicable in dev mode');
  await page.goto('https://example.com');
});
`
		parser := &PlaywrightParser{}
		ctx := context.Background()

		testFile, err := parser.Parse(ctx, []byte(source), "test.spec.ts")

		require.NoError(t, err)
		require.Len(t, testFile.Tests, 1, "Only one test should be detected")
		assert.Equal(t, "Story context menu actions", testFile.Tests[0].Name)
	})

	t.Run("conditional fixme should NOT be counted as test", func(t *testing.T) {
		source := `
import { test } from '@playwright/test';

test.describe('Suite', () => {
  test.fixme(condition, 'This needs fixing later');

  test('actual test', async ({ page }) => {});
});
`
		parser := &PlaywrightParser{}
		ctx := context.Background()

		testFile, err := parser.Parse(ctx, []byte(source), "fixme.spec.ts")

		require.NoError(t, err)
		require.Len(t, testFile.Suites, 1)
		require.Len(t, testFile.Suites[0].Tests, 1, "Conditional fixme should NOT be counted")
		assert.Equal(t, "actual test", testFile.Suites[0].Tests[0].Name)
	})

	t.Run("test.skip with string name should still be detected", func(t *testing.T) {
		source := `
import { test } from '@playwright/test';

test.skip('skipped test', async ({ page }) => {
  await page.goto('https://example.com');
});
`
		parser := &PlaywrightParser{}
		ctx := context.Background()

		testFile, err := parser.Parse(ctx, []byte(source), "skip.spec.ts")

		require.NoError(t, err)
		require.Len(t, testFile.Tests, 1, "test.skip with string name should be detected")
		assert.Equal(t, "skipped test", testFile.Tests[0].Name)
		assert.Equal(t, domain.TestStatusSkipped, testFile.Tests[0].Status)
	})
}

func TestPlaywrightParser_IndirectImport(t *testing.T) {
	t.Run("tests with indirect import from pageTest", func(t *testing.T) {
		source := `
import { test as it, expect } from './pageTest';

it('should work with indirect import', async ({ page }) => {
  await page.goto('https://example.com');
});

it.skip('should skip this test', async ({ page }) => {
  // This test is skipped
});
`
		parser := &PlaywrightParser{}
		ctx := context.Background()

		testFile, err := parser.Parse(ctx, []byte(source), "page-test.spec.ts")

		require.NoError(t, err)
		assert.Equal(t, "playwright", testFile.Framework)
		require.Len(t, testFile.Tests, 2, "Both tests should be detected")
		assert.Equal(t, "should work with indirect import", testFile.Tests[0].Name)
		assert.Equal(t, domain.TestStatusActive, testFile.Tests[0].Status)
		assert.Equal(t, "should skip this test", testFile.Tests[1].Name)
		assert.Equal(t, domain.TestStatusSkipped, testFile.Tests[1].Status)
	})

	t.Run("tests with browserTest import", func(t *testing.T) {
		source := `
import { browserTest as it, expect } from '../config/browserTest';

it('should work with browserTest', async ({ browser }) => {
  const context = await browser.newContext();
});
`
		parser := &PlaywrightParser{}
		ctx := context.Background()

		testFile, err := parser.Parse(ctx, []byte(source), "browser.spec.ts")

		require.NoError(t, err)
		assert.Equal(t, "playwright", testFile.Framework)
		require.Len(t, testFile.Tests, 1, "Test should be detected")
		assert.Equal(t, "should work with browserTest", testFile.Tests[0].Name)
	})

	t.Run("tests with contextTest import", func(t *testing.T) {
		source := `
import { contextTest as it, expect } from '../config/browserTest';

it('should work with contextTest', async () => {
  // test code
});
`
		parser := &PlaywrightParser{}
		ctx := context.Background()

		testFile, err := parser.Parse(ctx, []byte(source), "context.spec.ts")

		require.NoError(t, err)
		assert.Equal(t, "playwright", testFile.Framework)
		require.Len(t, testFile.Tests, 1, "Test should be detected")
		assert.Equal(t, "should work with contextTest", testFile.Tests[0].Name)
	})

	t.Run("tests without any import should still detect test and it", func(t *testing.T) {
		source := `
test('test without import', async ({ page }) => {
  await page.goto('https://example.com');
});

it('it without import', async ({ page }) => {
  await page.click('button');
});
`
		parser := &PlaywrightParser{}
		ctx := context.Background()

		testFile, err := parser.Parse(ctx, []byte(source), "no-import.spec.ts")

		require.NoError(t, err)
		assert.Equal(t, "playwright", testFile.Framework)
		require.Len(t, testFile.Tests, 2, "Both tests should be detected")
		assert.Equal(t, "test without import", testFile.Tests[0].Name)
		assert.Equal(t, "it without import", testFile.Tests[1].Name)
	})
}
