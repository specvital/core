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
		},
		{
			name:                "minimal config",
			configContent:       `export default defineConfig({});`,
			configPath:          "/project/playwright.config.js",
			expectedGlobalsMode: false,
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
