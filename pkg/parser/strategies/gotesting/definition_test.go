package gotesting

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

	assert.Equal(t, "go-testing", def.Name)
	assert.Equal(t, framework.PriorityGeneric, def.Priority)
	assert.ElementsMatch(t,
		[]domain.Language{domain.LanguageGo},
		def.Languages,
	)
	assert.Nil(t, def.ConfigParser, "Go testing should not have config parser")
	assert.NotNil(t, def.Parser)
	assert.Len(t, def.Matchers, 2) // ImportMatcher + GoTestFileMatcher
}

func TestGoTestFileMatcher_Match(t *testing.T) {
	matcher := &GoTestFileMatcher{}
	ctx := context.Background()

	tests := []struct {
		name        string
		fileName    string
		shouldMatch bool
	}{
		{
			name:        "standard test file",
			fileName:    "example_test.go",
			shouldMatch: true,
		},
		{
			name:        "test file with path",
			fileName:    "/project/pkg/parser/scanner_test.go",
			shouldMatch: true,
		},
		{
			name:        "test file with multiple underscores",
			fileName:    "my_module_test.go",
			shouldMatch: true,
		},
		{
			name:        "non-test go file",
			fileName:    "example.go",
			shouldMatch: false,
		},
		{
			name:        "test in name but wrong position",
			fileName:    "testing_utils.go",
			shouldMatch: false,
		},
		{
			name:        "test file with wrong extension",
			fileName:    "example_test.js",
			shouldMatch: false,
		},
		{
			name:        "empty string",
			fileName:    "",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signal := framework.Signal{
				Type:  framework.SignalFileName,
				Value: tt.fileName,
			}

			result := matcher.Match(ctx, signal)

			if tt.shouldMatch {
				assert.Equal(t, 100, result.Confidence, "Should be definite match for Go test file")
				assert.NotEmpty(t, result.Evidence, "Should have evidence")
			} else {
				assert.Equal(t, 0, result.Confidence, "Should not match")
			}
		})
	}
}

func TestGoTestFileMatcher_NonFileNameSignal(t *testing.T) {
	matcher := &GoTestFileMatcher{}
	ctx := context.Background()

	// Should not match non-filename signals
	signal := framework.Signal{
		Type:  framework.SignalImport,
		Value: "testing",
	}

	result := matcher.Match(ctx, signal)
	assert.Equal(t, 0, result.Confidence)
}

func TestGoTestingParser_Parse(t *testing.T) {
	testSource := `
package mypackage

import (
	"testing"
)

func TestSimple(t *testing.T) {
	if 1+1 != 2 {
		t.Error("math is broken")
	}
}

func TestWithSubtests(t *testing.T) {
	t.Run("subtest 1", func(t *testing.T) {
		// test code
	})

	t.Run("subtest 2", func(t *testing.T) {
		// test code
	})
}

func TestAnother(t *testing.T) {
	// simple test without subtests
}
`

	parser := &GoTestingParser{}
	ctx := context.Background()

	testFile, err := parser.Parse(ctx, []byte(testSource), "mypackage_test.go")

	require.NoError(t, err)
	assert.Equal(t, "mypackage_test.go", testFile.Path)
	assert.Equal(t, "go-testing", testFile.Framework)
	assert.Equal(t, domain.LanguageGo, testFile.Language)

	// Verify we have one suite (TestWithSubtests) and two standalone tests
	require.Len(t, testFile.Suites, 1, "Should have one test suite")
	require.Len(t, testFile.Tests, 2, "Should have two standalone tests")

	// Verify suite with subtests
	suite := testFile.Suites[0]
	assert.Equal(t, "TestWithSubtests", suite.Name)
	assert.Len(t, suite.Tests, 2, "Suite should have 2 subtests")
	assert.Equal(t, "subtest 1", suite.Tests[0].Name)
	assert.Equal(t, "subtest 2", suite.Tests[1].Name)

	// Verify standalone tests
	assert.Equal(t, "TestSimple", testFile.Tests[0].Name)
	assert.Equal(t, "TestAnother", testFile.Tests[1].Name)
}

func TestGoTestingParser_TestNamingConventions(t *testing.T) {
	tests := []struct {
		name               string
		source             string
		expectedTestCount  int
		expectedSuiteCount int
		expectedTestName   string
	}{
		{
			name: "valid test with capital letter after Test",
			source: `
package test
import "testing"
func TestValidName(t *testing.T) {}
`,
			expectedTestCount:  1,
			expectedSuiteCount: 0,
			expectedTestName:   "TestValidName",
		},
		{
			name: "invalid test with lowercase after Test",
			source: `
package test
import "testing"
func Testinvalid(t *testing.T) {}
`,
			expectedTestCount:  0,
			expectedSuiteCount: 0,
		},
		{
			name: "test with numbers in name",
			source: `
package test
import "testing"
func TestCase123(t *testing.T) {}
`,
			expectedTestCount:  1,
			expectedSuiteCount: 0,
			expectedTestName:   "TestCase123",
		},
		{
			name: "test with underscore after Test prefix",
			source: `
package test
import "testing"
func Test_With_Underscores(t *testing.T) {}
`,
			expectedTestCount:  1,
			expectedSuiteCount: 0,
			expectedTestName:   "Test_With_Underscores",
		},
		{
			name: "test with underscores (valid)",
			source: `
package test
import "testing"
func TestWith_Underscores(t *testing.T) {}
`,
			expectedTestCount:  1,
			expectedSuiteCount: 0,
			expectedTestName:   "TestWith_Underscores",
		},
		{
			name: "non-test function starting with Test",
			source: `
package test
import "testing"
func Test() {}
`,
			expectedTestCount:  0,
			expectedSuiteCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &GoTestingParser{}
			ctx := context.Background()

			testFile, err := parser.Parse(ctx, []byte(tt.source), "test.go")

			require.NoError(t, err)
			assert.Len(t, testFile.Tests, tt.expectedTestCount, "Test count mismatch")
			assert.Len(t, testFile.Suites, tt.expectedSuiteCount, "Suite count mismatch")

			if tt.expectedTestCount > 0 {
				assert.Equal(t, tt.expectedTestName, testFile.Tests[0].Name)
			}
		})
	}
}

func TestGoTestingParser_ParameterValidation(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		shouldDetect bool
		description  string
	}{
		{
			name: "valid test with *testing.T",
			source: `
package test
import "testing"
func TestValid(t *testing.T) {}
`,
			shouldDetect: true,
			description:  "Standard valid test function",
		},
		{
			name: "invalid - wrong parameter type",
			source: `
package test
func TestInvalid(t string) {}
`,
			shouldDetect: false,
			description:  "Parameter is not *testing.T",
		},
		{
			name: "invalid - no parameters",
			source: `
package test
func TestInvalid() {}
`,
			shouldDetect: false,
			description:  "No parameters",
		},
		{
			name: "invalid - multiple parameters",
			source: `
package test
import "testing"
func TestInvalid(t *testing.T, s string) {}
`,
			shouldDetect: false,
			description:  "Too many parameters",
		},
		{
			name: "invalid - non-pointer testing.T",
			source: `
package test
import "testing"
func TestInvalid(t testing.T) {}
`,
			shouldDetect: false,
			description:  "testing.T instead of *testing.T",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &GoTestingParser{}
			ctx := context.Background()

			testFile, err := parser.Parse(ctx, []byte(tt.source), "test.go")

			require.NoError(t, err)

			totalTests := len(testFile.Tests) + len(testFile.Suites)
			if tt.shouldDetect {
				assert.Greater(t, totalTests, 0, tt.description)
			} else {
				assert.Equal(t, 0, totalTests, tt.description)
			}
		})
	}
}

func TestGoTestingParser_NestedSubtests(t *testing.T) {
	testSource := `
package test

import "testing"

func TestWithNestedSubtests(t *testing.T) {
	t.Run("level 1", func(t *testing.T) {
		t.Run("level 2", func(t *testing.T) {
			// deeply nested
		})
	})
}
`

	parser := &GoTestingParser{}
	ctx := context.Background()

	testFile, err := parser.Parse(ctx, []byte(testSource), "nested_test.go")

	require.NoError(t, err)
	assert.Len(t, testFile.Suites, 1)

	suite := testFile.Suites[0]
	assert.Equal(t, "TestWithNestedSubtests", suite.Name)
	// The current implementation captures all nested t.Run calls in a flat structure
	assert.Len(t, suite.Tests, 2)
	assert.Equal(t, "level 1", suite.Tests[0].Name)
	assert.Equal(t, "level 2", suite.Tests[1].Name)
}

func TestGoTestingParser_BenchmarkFunctions(t *testing.T) {
	testSource := `
package test

import "testing"

func BenchmarkSort(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// benchmark code
	}
}

func BenchmarkSearch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// benchmark code
	}
}
`

	parser := &GoTestingParser{}
	ctx := context.Background()

	testFile, err := parser.Parse(ctx, []byte(testSource), "bench_test.go")

	require.NoError(t, err)
	require.Len(t, testFile.Tests, 2)
	assert.Equal(t, "BenchmarkSort", testFile.Tests[0].Name)
	assert.Equal(t, "BenchmarkSearch", testFile.Tests[1].Name)
}

func TestGoTestingParser_ExampleFunctions(t *testing.T) {
	testSource := `
package test

func Example() {
	// Output: hello
}

func ExampleHello() {
	// Output: hello
}

func Example_suffix() {
	// Output: hello
}
`

	parser := &GoTestingParser{}
	ctx := context.Background()

	testFile, err := parser.Parse(ctx, []byte(testSource), "example_test.go")

	require.NoError(t, err)
	require.Len(t, testFile.Tests, 3)
	assert.Equal(t, "Example", testFile.Tests[0].Name)
	assert.Equal(t, "ExampleHello", testFile.Tests[1].Name)
	assert.Equal(t, "Example_suffix", testFile.Tests[2].Name)
}

func TestGoTestingParser_FuzzFunctions(t *testing.T) {
	testSource := `
package test

import "testing"

func FuzzReverse(f *testing.F) {
	f.Add("hello")
	f.Fuzz(func(t *testing.T, s string) {
		// fuzz test
	})
}
`

	parser := &GoTestingParser{}
	ctx := context.Background()

	testFile, err := parser.Parse(ctx, []byte(testSource), "fuzz_test.go")

	require.NoError(t, err)
	require.Len(t, testFile.Tests, 1)
	assert.Equal(t, "FuzzReverse", testFile.Tests[0].Name)
}

func TestGoTestingParser_MixedFunctions(t *testing.T) {
	testSource := `
package test

import "testing"

func TestUnit(t *testing.T) {}
func BenchmarkPerf(b *testing.B) {}
func ExampleUsage() {}
func FuzzInput(f *testing.F) {}
`

	parser := &GoTestingParser{}
	ctx := context.Background()

	testFile, err := parser.Parse(ctx, []byte(testSource), "mixed_test.go")

	require.NoError(t, err)
	require.Len(t, testFile.Tests, 4)
	assert.Equal(t, "TestUnit", testFile.Tests[0].Name)
	assert.Equal(t, "BenchmarkPerf", testFile.Tests[1].Name)
	assert.Equal(t, "ExampleUsage", testFile.Tests[2].Name)
	assert.Equal(t, "FuzzInput", testFile.Tests[3].Name)
}

func TestClassifyTestFunction(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		want     goTestFuncType
	}{
		{"valid Test", "TestFoo", funcTypeTest},
		{"invalid Test lowercase", "Testfoo", funcTypeNone},
		{"Test only", "Test", funcTypeNone},
		{"Test with underscore", "Test_foo", funcTypeTest},
		{"Test with number", "Test123", funcTypeTest},
		{"valid Benchmark", "BenchmarkFoo", funcTypeBenchmark},
		{"invalid Benchmark lowercase", "Benchmarkfoo", funcTypeNone},
		{"Benchmark only", "Benchmark", funcTypeNone},
		{"Benchmark with underscore", "Benchmark_foo", funcTypeBenchmark},
		{"valid Example with name", "ExampleFoo", funcTypeExample},
		{"Example only", "Example", funcTypeExample},
		{"Example with underscore", "Example_foo", funcTypeExample},
		{"invalid Example lowercase", "Examplefoo", funcTypeNone},
		{"valid Fuzz", "FuzzFoo", funcTypeFuzz},
		{"invalid Fuzz lowercase", "Fuzzfoo", funcTypeNone},
		{"Fuzz only", "Fuzz", funcTypeNone},
		{"Fuzz with underscore", "Fuzz_foo", funcTypeFuzz},
		{"random function", "DoSomething", funcTypeNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyTestFunction(tt.funcName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGoTestingParser_InvalidParams(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name: "benchmark with wrong param",
			source: `
package test
import "testing"
func BenchmarkInvalid(t *testing.T) {}
`,
		},
		{
			name: "fuzz with wrong param",
			source: `
package test
import "testing"
func FuzzInvalid(t *testing.T) {}
`,
		},
		{
			name: "example with param",
			source: `
package test
import "testing"
func ExampleInvalid(t *testing.T) {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &GoTestingParser{}
			ctx := context.Background()

			testFile, err := parser.Parse(ctx, []byte(tt.source), "test.go")

			require.NoError(t, err)
			assert.Empty(t, testFile.Tests, "Should not detect function with wrong params")
		})
	}
}
