package gtest

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
)

func TestGTestParser_Parse(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		checkFunc func(t *testing.T, file *domain.TestFile)
	}{
		{
			name: "basic TEST macro",
			source: `
#include <gtest/gtest.h>

TEST(SuiteName, TestName) {
    EXPECT_EQ(1, 1);
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 suite, got %d", len(file.Suites))
					return
				}
				suite := file.Suites[0]
				if suite.Name != "SuiteName" {
					t.Errorf("expected suite name 'SuiteName', got %q", suite.Name)
				}
				if len(suite.Tests) != 1 {
					t.Errorf("expected 1 test, got %d", len(suite.Tests))
					return
				}
				test := suite.Tests[0]
				if test.Name != "TestName" {
					t.Errorf("expected test name 'TestName', got %q", test.Name)
				}
				if test.Status != domain.TestStatusActive {
					t.Errorf("expected status active, got %q", test.Status)
				}
			},
		},
		{
			name: "TEST_F with fixture",
			source: `
#include <gtest/gtest.h>

class MyFixture : public ::testing::Test {
protected:
    void SetUp() override {}
};

TEST_F(MyFixture, TestWithFixture) {
    EXPECT_TRUE(true);
}

TEST_F(MyFixture, AnotherTest) {
    EXPECT_FALSE(false);
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 suite, got %d", len(file.Suites))
					return
				}
				suite := file.Suites[0]
				if suite.Name != "MyFixture" {
					t.Errorf("expected suite name 'MyFixture', got %q", suite.Name)
				}
				if len(suite.Tests) != 2 {
					t.Errorf("expected 2 tests, got %d", len(suite.Tests))
					return
				}
			},
		},
		{
			name: "TEST_P parameterized test",
			source: `
#include <gtest/gtest.h>

class ParamTest : public ::testing::TestWithParam<int> {};

TEST_P(ParamTest, ChecksValue) {
    EXPECT_GT(GetParam(), 0);
}

INSTANTIATE_TEST_SUITE_P(MyInstance, ParamTest, ::testing::Values(1, 2, 3));
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 suite, got %d", len(file.Suites))
					return
				}
				suite := file.Suites[0]
				if suite.Name != "ParamTest" {
					t.Errorf("expected suite name 'ParamTest', got %q", suite.Name)
				}
				if len(suite.Tests) != 1 {
					t.Errorf("expected 1 test, got %d", len(suite.Tests))
					return
				}
				test := suite.Tests[0]
				if test.Name != "ChecksValue" {
					t.Errorf("expected test name 'ChecksValue', got %q", test.Name)
				}
			},
		},
		{
			name: "DISABLED_ test prefix",
			source: `
#include <gtest/gtest.h>

TEST(Suite, DISABLED_SkippedTest) {
    FAIL() << "Should not run";
}

TEST(Suite, ActiveTest) {
    EXPECT_TRUE(true);
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 suite, got %d", len(file.Suites))
					return
				}
				suite := file.Suites[0]
				if len(suite.Tests) != 2 {
					t.Errorf("expected 2 tests, got %d", len(suite.Tests))
					return
				}

				// Find tests by name
				var skipped, active *domain.Test
				for i := range suite.Tests {
					if suite.Tests[i].Name == "DISABLED_SkippedTest" {
						skipped = &suite.Tests[i]
					} else if suite.Tests[i].Name == "ActiveTest" {
						active = &suite.Tests[i]
					}
				}

				if skipped == nil {
					t.Error("expected to find DISABLED_SkippedTest")
					return
				}
				if skipped.Status != domain.TestStatusSkipped {
					t.Errorf("expected skipped status, got %q", skipped.Status)
				}
				if skipped.Modifier != "DISABLED_" {
					t.Errorf("expected modifier 'DISABLED_', got %q", skipped.Modifier)
				}

				if active == nil {
					t.Error("expected to find ActiveTest")
					return
				}
				if active.Status != domain.TestStatusActive {
					t.Errorf("expected active status, got %q", active.Status)
				}
			},
		},
		{
			name: "DISABLED_ suite prefix",
			source: `
#include <gtest/gtest.h>

TEST(DISABLED_Suite, TestOne) {
    EXPECT_TRUE(true);
}

TEST(DISABLED_Suite, TestTwo) {
    EXPECT_TRUE(true);
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 suite, got %d", len(file.Suites))
					return
				}
				suite := file.Suites[0]
				if suite.Name != "DISABLED_Suite" {
					t.Errorf("expected suite name 'DISABLED_Suite', got %q", suite.Name)
				}
				if suite.Status != domain.TestStatusSkipped {
					t.Errorf("expected suite status skipped, got %q", suite.Status)
				}
				// All tests in disabled suite should be skipped
				for _, test := range suite.Tests {
					if test.Status != domain.TestStatusSkipped {
						t.Errorf("expected test %q status skipped, got %q", test.Name, test.Status)
					}
				}
			},
		},
		{
			name: "multiple suites",
			source: `
#include <gtest/gtest.h>

TEST(SuiteA, Test1) { EXPECT_TRUE(true); }
TEST(SuiteA, Test2) { EXPECT_TRUE(true); }
TEST(SuiteB, Test1) { EXPECT_TRUE(true); }
TEST(SuiteC, Test1) { EXPECT_TRUE(true); }
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 3 {
					t.Errorf("expected 3 suites, got %d", len(file.Suites))
					return
				}

				suiteMap := make(map[string]*domain.TestSuite)
				for i := range file.Suites {
					suiteMap[file.Suites[i].Name] = &file.Suites[i]
				}

				if suiteA, ok := suiteMap["SuiteA"]; ok {
					if len(suiteA.Tests) != 2 {
						t.Errorf("SuiteA expected 2 tests, got %d", len(suiteA.Tests))
					}
				} else {
					t.Error("expected SuiteA")
				}

				if suiteB, ok := suiteMap["SuiteB"]; ok {
					if len(suiteB.Tests) != 1 {
						t.Errorf("SuiteB expected 1 test, got %d", len(suiteB.Tests))
					}
				} else {
					t.Error("expected SuiteB")
				}

				if suiteC, ok := suiteMap["SuiteC"]; ok {
					if len(suiteC.Tests) != 1 {
						t.Errorf("SuiteC expected 1 test, got %d", len(suiteC.Tests))
					}
				} else {
					t.Error("expected SuiteC")
				}
			},
		},
		{
			name: "non-test functions ignored",
			source: `
#include <gtest/gtest.h>

void HelperFunction() {
    // Not a test
}

TEST(Suite, ActualTest) {
    EXPECT_TRUE(true);
}

int main(int argc, char** argv) {
    ::testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
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
			name: "test location accuracy",
			source: `#include <gtest/gtest.h>

TEST(Suite, TestName) {
    EXPECT_TRUE(true);
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 || len(file.Suites[0].Tests) != 1 {
					t.Error("expected 1 suite with 1 test")
					return
				}
				test := file.Suites[0].Tests[0]
				// TEST starts at line 3 (1-indexed)
				if test.Location.StartLine != 3 {
					t.Errorf("expected start line 3, got %d", test.Location.StartLine)
				}
			},
		},
		{
			name: "TYPED_TEST macro",
			source: `
#include <gtest/gtest.h>

template <typename T>
class MyTypedTest : public ::testing::Test {};

typedef ::testing::Types<int, float, double> MyTypes;
TYPED_TEST_SUITE(MyTypedTest, MyTypes);

TYPED_TEST(MyTypedTest, DoesWork) {
    TypeParam value = 0;
    EXPECT_EQ(value, TypeParam());
}

TYPED_TEST(MyTypedTest, AlsoWorks) {
    EXPECT_TRUE(true);
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 suite, got %d", len(file.Suites))
					return
				}
				suite := file.Suites[0]
				if suite.Name != "MyTypedTest" {
					t.Errorf("expected suite name 'MyTypedTest', got %q", suite.Name)
				}
				if len(suite.Tests) != 2 {
					t.Errorf("expected 2 tests, got %d", len(suite.Tests))
					return
				}
				// Verify test names
				testNames := make(map[string]bool)
				for _, test := range suite.Tests {
					testNames[test.Name] = true
				}
				if !testNames["DoesWork"] {
					t.Error("expected to find test 'DoesWork'")
				}
				if !testNames["AlsoWorks"] {
					t.Error("expected to find test 'AlsoWorks'")
				}
			},
		},
		{
			name: "TYPED_TEST_P macro",
			source: `
#include <gtest/gtest.h>

template <typename T>
class MyTypedTestP : public ::testing::Test {};

TYPED_TEST_SUITE_P(MyTypedTestP);

TYPED_TEST_P(MyTypedTestP, CanBeDefaultConstructed) {
    TypeParam container;
}

TYPED_TEST_P(MyTypedTestP, InitialSizeIsZero) {
    TypeParam container;
    EXPECT_EQ(0U, container.size());
}

TYPED_TEST_P(MyTypedTestP, CanGetNextPrime) {
    EXPECT_TRUE(true);
}

REGISTER_TYPED_TEST_SUITE_P(MyTypedTestP, CanBeDefaultConstructed, InitialSizeIsZero, CanGetNextPrime);
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 suite, got %d", len(file.Suites))
					return
				}
				suite := file.Suites[0]
				if suite.Name != "MyTypedTestP" {
					t.Errorf("expected suite name 'MyTypedTestP', got %q", suite.Name)
				}
				if len(suite.Tests) != 3 {
					t.Errorf("expected 3 tests, got %d", len(suite.Tests))
					return
				}
				// Verify test names
				testNames := make(map[string]bool)
				for _, test := range suite.Tests {
					testNames[test.Name] = true
				}
				if !testNames["CanBeDefaultConstructed"] {
					t.Error("expected to find test 'CanBeDefaultConstructed'")
				}
				if !testNames["InitialSizeIsZero"] {
					t.Error("expected to find test 'InitialSizeIsZero'")
				}
				if !testNames["CanGetNextPrime"] {
					t.Error("expected to find test 'CanGetNextPrime'")
				}
			},
		},
		{
			name: "TYPED_TEST with DISABLED_ prefix",
			source: `
#include <gtest/gtest.h>

template <typename T>
class DisabledTypedTest : public ::testing::Test {};

TYPED_TEST_SUITE(DisabledTypedTest, ::testing::Types<int>);

TYPED_TEST(DisabledTypedTest, DISABLED_SkippedTest) {
    FAIL() << "Should not run";
}

TYPED_TEST(DisabledTypedTest, ActiveTest) {
    EXPECT_TRUE(true);
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 suite, got %d", len(file.Suites))
					return
				}
				suite := file.Suites[0]
				if len(suite.Tests) != 2 {
					t.Errorf("expected 2 tests, got %d", len(suite.Tests))
					return
				}

				// Find tests by name
				var skipped, active *domain.Test
				for i := range suite.Tests {
					if suite.Tests[i].Name == "DISABLED_SkippedTest" {
						skipped = &suite.Tests[i]
					} else if suite.Tests[i].Name == "ActiveTest" {
						active = &suite.Tests[i]
					}
				}

				if skipped == nil {
					t.Error("expected to find DISABLED_SkippedTest")
					return
				}
				if skipped.Status != domain.TestStatusSkipped {
					t.Errorf("expected skipped status, got %q", skipped.Status)
				}

				if active == nil {
					t.Error("expected to find ActiveTest")
					return
				}
				if active.Status != domain.TestStatusActive {
					t.Errorf("expected active status, got %q", active.Status)
				}
			},
		},
	}

	parser := &GTestParser{}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := parser.Parse(ctx, []byte(tt.source), "test.cpp")
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if file.Framework != frameworkName {
				t.Errorf("expected framework %q, got %q", frameworkName, file.Framework)
			}

			if file.Language != domain.LanguageCpp {
				t.Errorf("expected language Cpp, got %q", file.Language)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, file)
			}
		})
	}
}

func TestGTestFileMatcher_Match(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantMatch bool
	}{
		{"_test.cc suffix", "module_test.cc", true},
		{"_test.cpp suffix", "module_test.cpp", true},
		{"_unittest.cc suffix", "module_unittest.cc", true},
		{"_unittest.cpp suffix", "module_unittest.cpp", true},
		{"Test.cpp suffix", "MyTest.cpp", true},
		{"Test.cc suffix", "MyTest.cc", true},
		{"test directory cc", "src/test/integration.cc", true},
		{"tests directory cpp", "src/tests/unit.cpp", true},
		{"regular cpp file", "main.cpp", false},
		{"header file", "test.h", false},
		{"source file", "source.cc", false},
	}

	matcher := &GTestFileMatcher{}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signal := framework.Signal{
				Type:  framework.SignalFileName,
				Value: tt.filename,
			}
			result := matcher.Match(ctx, signal)
			if (result.Confidence > 0) != tt.wantMatch {
				t.Errorf("Match() confidence = %v, want match = %v", result.Confidence, tt.wantMatch)
			}
		})
	}
}

func TestGTestContentMatcher_Match(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantMatch bool
	}{
		{"gtest include angle", "#include <gtest/gtest.h>", true},
		{"gtest include quote", `#include "gtest/gtest.h"`, true},
		{"TEST macro", "TEST(Suite, Test) {}", true},
		{"TEST_F macro", "TEST_F(Fixture, Test) {}", true},
		{"TEST_P macro", "TEST_P(ParamTest, Test) {}", true},
		{"TYPED_TEST macro", "TYPED_TEST(TypedSuite, Test) {}", true},
		{"TYPED_TEST_P macro", "TYPED_TEST_P(TypedSuiteP, Test) {}", true},
		{"INSTANTIATE macro", "INSTANTIATE_TEST_SUITE_P(Instance, Suite, Values)", true},
		{"testing::Test base", "class Foo : public ::testing::Test {}", true},
		{"plain cpp code", "int main() { return 0; }", false},
		{"other test framework", "#include <catch2/catch.hpp>", false},
	}

	matcher := &GTestContentMatcher{}
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
				t.Errorf("Match() confidence = %v, want match = %v", result.Confidence, tt.wantMatch)
			}
		})
	}
}
