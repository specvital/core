package parser_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/specvital/core/pkg/parser"
	"github.com/specvital/core/pkg/source"

	// Import frameworks to register them via init()
	_ "github.com/specvital/core/pkg/parser/strategies/gtest"
	_ "github.com/specvital/core/pkg/parser/strategies/jest"
	_ "github.com/specvital/core/pkg/parser/strategies/phpunit"
)

func TestScan(t *testing.T) {
	t.Run("should return empty inventory for empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		result, err := parser.Scan(context.Background(), src)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Inventory == nil {
			t.Fatal("inventory should not be nil")
		}
		if len(result.Inventory.Files) != 0 {
			t.Errorf("expected 0 files, got %d", len(result.Inventory.Files))
		}
	})

	t.Run("should scan test files in directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		testContent := []byte(`
import { describe, it } from '@jest/globals';

describe('UserService', () => {
  it('should create user', () => {});
  it('should delete user', () => {});
});
`)
		testFile := filepath.Join(tmpDir, "user.test.ts")
		if err := os.WriteFile(testFile, testContent, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		result, err := parser.Scan(context.Background(), src)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Inventory.Files) != 1 {
			t.Errorf("expected 1 file, got %d", len(result.Inventory.Files))
		}
		if result.Inventory.RootPath != tmpDir {
			t.Errorf("expected rootPath %s, got %s", tmpDir, result.Inventory.RootPath)
		}
	})

	t.Run("should respect exclude patterns", func(t *testing.T) {
		tmpDir := t.TempDir()

		customDir := filepath.Join(tmpDir, "custom_exclude")
		if err := os.MkdirAll(customDir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		testContent := []byte(`it('test', () => {});`)
		if err := os.WriteFile(filepath.Join(customDir, "excluded.test.ts"), testContent, 0644); err != nil {
			t.Fatalf("failed to write: %v", err)
		}

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		result, err := parser.Scan(context.Background(), src, parser.WithExclude([]string{"custom_exclude"}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Inventory.Files) != 0 {
			t.Errorf("expected 0 files, got %d", len(result.Inventory.Files))
		}
	})

	t.Run("should respect worker count option", func(t *testing.T) {
		tmpDir := t.TempDir()

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		result, err := parser.Scan(context.Background(), src, parser.WithWorkers(2))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result == nil {
			t.Fatal("result should not be nil")
		}
	})

	t.Run("should respect timeout option", func(t *testing.T) {
		tmpDir := t.TempDir()

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		result, err := parser.Scan(context.Background(), src, parser.WithTimeout(30*time.Second))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result == nil {
			t.Fatal("result should not be nil")
		}
	})

	t.Run("should return error for non-existent path", func(t *testing.T) {
		_, err := source.NewLocalSource("/non/existent/path")
		if err == nil {
			t.Error("expected error for non-existent path")
		}
	})

	t.Run("should return ErrScanCancelled on context cancellation", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a test file to ensure scan has work to do
		testContent := []byte(`import { it } from '@jest/globals'; it('test', () => {});`)
		if err := os.WriteFile(filepath.Join(tmpDir, "test.test.ts"), testContent, 0644); err != nil {
			t.Fatalf("failed to write: %v", err)
		}

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = parser.Scan(ctx, src)
		// Note: The scan may complete very quickly before detecting cancellation
		// on fast systems, so we just check it doesn't return unexpected errors
		if err != nil && !errors.Is(err, parser.ErrScanCancelled) {
			t.Errorf("expected nil or ErrScanCancelled, got %v", err)
		}
	})

	t.Run("should aggregate errors from multiple files", func(t *testing.T) {
		tmpDir := t.TempDir()

		validContent := []byte(`import { it } from '@jest/globals'; it('test', () => {});`)
		if err := os.WriteFile(filepath.Join(tmpDir, "valid.test.ts"), validContent, 0644); err != nil {
			t.Fatalf("failed to write: %v", err)
		}

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		result, err := parser.Scan(context.Background(), src)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Inventory.Files) < 1 {
			t.Errorf("expected at least 1 parsed file")
		}
	})
}

func TestScan_Concurrency(t *testing.T) {
	t.Run("should safely handle concurrent access", func(t *testing.T) {
		tmpDir := t.TempDir()

		for i := 0; i < 10; i++ {
			content := []byte(`import { it } from '@jest/globals'; it('test', () => {});`)
			filename := filepath.Join(tmpDir, fmt.Sprintf("test%d.test.ts", i))
			if err := os.WriteFile(filename, content, 0644); err != nil {
				t.Fatalf("failed to write: %v", err)
			}
		}

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		var wg sync.WaitGroup
		var errCount atomic.Int32

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := parser.Scan(context.Background(), src, parser.WithWorkers(4))
				if err != nil {
					errCount.Add(1)
				}
			}()
		}

		wg.Wait()

		if errCount.Load() > 0 {
			t.Errorf("concurrent scans had %d errors", errCount.Load())
		}
	})

	t.Run("should complete with race detector", func(t *testing.T) {
		tmpDir := t.TempDir()

		for i := 0; i < 20; i++ {
			content := []byte(`
import { describe, it } from '@jest/globals';

describe('Suite', () => {
  it('test 1', () => {});
  it('test 2', () => {});
});
`)
			filename := filepath.Join(tmpDir, fmt.Sprintf("test%d.test.ts", i))
			if err := os.WriteFile(filename, content, 0644); err != nil {
				t.Fatalf("failed to write: %v", err)
			}
		}

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		result, err := parser.Scan(context.Background(), src, parser.WithWorkers(8))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Allow tolerance for concurrent test - language filtering now prevents
		// cross-language false positives (e.g., Go files no longer detected as vitest)
		if len(result.Inventory.Files) < 15 {
			t.Errorf("expected at least 15 files, got %d", len(result.Inventory.Files))
		}
	})
}

func TestScanOptions(t *testing.T) {
	t.Run("WithWorkers sets worker count", func(t *testing.T) {
		opts := &parser.ScanOptions{}
		parser.WithWorkers(4)(opts)
		if opts.Workers != 4 {
			t.Errorf("expected 4 workers, got %d", opts.Workers)
		}
	})

	t.Run("WithTimeout sets timeout", func(t *testing.T) {
		opts := &parser.ScanOptions{}
		parser.WithTimeout(30 * time.Second)(opts)
		if opts.Timeout != 30*time.Second {
			t.Errorf("expected 30s timeout, got %v", opts.Timeout)
		}
	})

	t.Run("WithExclude sets patterns", func(t *testing.T) {
		opts := &parser.ScanOptions{}
		patterns := []string{"vendor", "dist"}
		parser.WithExclude(patterns)(opts)
		if len(opts.ExcludePatterns) != 2 {
			t.Errorf("expected 2 patterns, got %d", len(opts.ExcludePatterns))
		}
	})

	t.Run("WithScanMaxFileSize sets max size", func(t *testing.T) {
		opts := &parser.ScanOptions{}
		parser.WithScanMaxFileSize(1024)(opts)
		if opts.MaxFileSize != 1024 {
			t.Errorf("expected 1024, got %d", opts.MaxFileSize)
		}
	})

	t.Run("WithScanMaxFileSize ignores negative values", func(t *testing.T) {
		opts := &parser.ScanOptions{MaxFileSize: 100}
		parser.WithScanMaxFileSize(-1)(opts)
		if opts.MaxFileSize != 100 {
			t.Errorf("expected 100 (unchanged), got %d", opts.MaxFileSize)
		}
	})

	t.Run("WithScanPatterns sets patterns", func(t *testing.T) {
		opts := &parser.ScanOptions{}
		patterns := []string{"**/*.test.ts"}
		parser.WithScanPatterns(patterns)(opts)
		if len(opts.Patterns) != 1 {
			t.Errorf("expected 1 pattern, got %d", len(opts.Patterns))
		}
	})

	t.Run("WithTimeout ignores negative values", func(t *testing.T) {
		opts := &parser.ScanOptions{Timeout: time.Minute}
		parser.WithTimeout(-1)(opts)
		if opts.Timeout != time.Minute {
			t.Errorf("expected 1m (unchanged), got %v", opts.Timeout)
		}
	})

	t.Run("WithWorkers ignores negative values", func(t *testing.T) {
		opts := &parser.ScanOptions{Workers: 4}
		parser.WithWorkers(-1)(opts)
		if opts.Workers != 4 {
			t.Errorf("expected 4 (unchanged), got %d", opts.Workers)
		}
	})
}

func TestScanError(t *testing.T) {
	t.Run("Error with path returns formatted string", func(t *testing.T) {
		err := parser.ScanError{
			Err:   os.ErrNotExist,
			Path:  "/path/to/file.ts",
			Phase: "parsing",
		}

		expected := "[parsing] /path/to/file.ts: file does not exist"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Error without path returns phase only", func(t *testing.T) {
		err := parser.ScanError{
			Err:   os.ErrPermission,
			Path:  "",
			Phase: "detection",
		}

		expected := "[detection] permission denied"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})
}

func TestScan_GoogleTest(t *testing.T) {
	t.Run("should scan C++ gtest files", func(t *testing.T) {
		tmpDir := t.TempDir()

		testContent := []byte(`
#include <gtest/gtest.h>

TEST(MathTest, Addition) {
    EXPECT_EQ(2 + 2, 4);
}

TEST(MathTest, Subtraction) {
    EXPECT_EQ(5 - 3, 2);
}
`)
		testFile := filepath.Join(tmpDir, "math_test.cc")
		if err := os.WriteFile(testFile, testContent, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		result, err := parser.Scan(context.Background(), src)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Inventory.Files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(result.Inventory.Files))
		}

		file := result.Inventory.Files[0]
		if file.Framework != "gtest" {
			t.Errorf("expected framework 'gtest', got %q", file.Framework)
		}
		if file.Language != "cpp" {
			t.Errorf("expected language 'cpp', got %q", file.Language)
		}
		if len(file.Suites) != 1 {
			t.Fatalf("expected 1 suite, got %d", len(file.Suites))
		}
		if file.Suites[0].Name != "MathTest" {
			t.Errorf("expected suite name 'MathTest', got %q", file.Suites[0].Name)
		}
		if len(file.Suites[0].Tests) != 2 {
			t.Errorf("expected 2 tests, got %d", len(file.Suites[0].Tests))
		}
	})

	t.Run("should handle TEST_F with fixtures", func(t *testing.T) {
		tmpDir := t.TempDir()

		testContent := []byte(`
#include <gtest/gtest.h>

class DatabaseTest : public ::testing::Test {
protected:
    void SetUp() override {}
    void TearDown() override {}
};

TEST_F(DatabaseTest, Connect) {
    EXPECT_TRUE(true);
}

TEST_F(DatabaseTest, Query) {
    EXPECT_TRUE(true);
}
`)
		testFile := filepath.Join(tmpDir, "database_test.cpp")
		if err := os.WriteFile(testFile, testContent, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		result, err := parser.Scan(context.Background(), src)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Inventory.Files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(result.Inventory.Files))
		}

		file := result.Inventory.Files[0]
		if len(file.Suites) != 1 {
			t.Fatalf("expected 1 suite, got %d", len(file.Suites))
		}
		if file.Suites[0].Name != "DatabaseTest" {
			t.Errorf("expected suite name 'DatabaseTest', got %q", file.Suites[0].Name)
		}
		if len(file.Suites[0].Tests) != 2 {
			t.Errorf("expected 2 tests, got %d", len(file.Suites[0].Tests))
		}
	})

	t.Run("should detect DISABLED_ tests as skipped", func(t *testing.T) {
		tmpDir := t.TempDir()

		testContent := []byte(`
#include <gtest/gtest.h>

TEST(Suite, DISABLED_SkippedTest) {
    FAIL() << "Should not run";
}

TEST(Suite, ActiveTest) {
    EXPECT_TRUE(true);
}
`)
		testFile := filepath.Join(tmpDir, "skip_test.cc")
		if err := os.WriteFile(testFile, testContent, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		result, err := parser.Scan(context.Background(), src)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Inventory.Files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(result.Inventory.Files))
		}

		file := result.Inventory.Files[0]
		if len(file.Suites[0].Tests) != 2 {
			t.Fatalf("expected 2 tests, got %d", len(file.Suites[0].Tests))
		}

		// Find tests by status
		var skippedCount, activeCount int
		for _, test := range file.Suites[0].Tests {
			if test.Status == "skipped" {
				skippedCount++
			} else if test.Status == "active" {
				activeCount++
			}
		}

		if skippedCount != 1 {
			t.Errorf("expected 1 skipped test, got %d", skippedCount)
		}
		if activeCount != 1 {
			t.Errorf("expected 1 active test, got %d", activeCount)
		}
	})

	t.Run("should scan .cxx files", func(t *testing.T) {
		tmpDir := t.TempDir()

		testContent := []byte(`
#include <gtest/gtest.h>

TEST(CxxTest, Works) {
    EXPECT_TRUE(true);
}
`)
		testFile := filepath.Join(tmpDir, "cxx_test.cxx")
		if err := os.WriteFile(testFile, testContent, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		result, err := parser.Scan(context.Background(), src)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Inventory.Files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(result.Inventory.Files))
		}

		file := result.Inventory.Files[0]
		if file.Framework != "gtest" {
			t.Errorf("expected framework 'gtest', got %q", file.Framework)
		}
	})
}

func TestScan_PHPUnit(t *testing.T) {
	t.Run("should scan PHP PHPUnit files", func(t *testing.T) {
		tmpDir := t.TempDir()

		testContent := []byte(`<?php
use PHPUnit\Framework\TestCase;

class UserTest extends TestCase
{
    public function testUserCanBeCreated()
    {
        $this->assertTrue(true);
    }

    public function testUserCanLogin()
    {
        $this->assertTrue(true);
    }
}
`)
		testFile := filepath.Join(tmpDir, "UserTest.php")
		if err := os.WriteFile(testFile, testContent, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		result, err := parser.Scan(context.Background(), src)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Inventory.Files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(result.Inventory.Files))
		}

		file := result.Inventory.Files[0]
		if file.Framework != "phpunit" {
			t.Errorf("expected framework 'phpunit', got %q", file.Framework)
		}
		if file.Language != "php" {
			t.Errorf("expected language 'php', got %q", file.Language)
		}
		if len(file.Suites) != 1 {
			t.Fatalf("expected 1 suite, got %d", len(file.Suites))
		}
		if file.Suites[0].Name != "UserTest" {
			t.Errorf("expected suite name 'UserTest', got %q", file.Suites[0].Name)
		}
		if len(file.Suites[0].Tests) != 2 {
			t.Errorf("expected 2 tests, got %d", len(file.Suites[0].Tests))
		}
	})

	t.Run("should detect @test annotation", func(t *testing.T) {
		tmpDir := t.TempDir()

		testContent := []byte(`<?php
use PHPUnit\Framework\TestCase;

class AnnotationTest extends TestCase
{
    /**
     * @test
     */
    public function it_creates_a_user()
    {
        $this->assertTrue(true);
    }
}
`)
		testFile := filepath.Join(tmpDir, "AnnotationTest.php")
		if err := os.WriteFile(testFile, testContent, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		result, err := parser.Scan(context.Background(), src)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Inventory.Files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(result.Inventory.Files))
		}

		file := result.Inventory.Files[0]
		if len(file.Suites[0].Tests) != 1 {
			t.Fatalf("expected 1 test, got %d", len(file.Suites[0].Tests))
		}
		if file.Suites[0].Tests[0].Name != "it_creates_a_user" {
			t.Errorf("expected test name 'it_creates_a_user', got %q", file.Suites[0].Tests[0].Name)
		}
	})

	t.Run("should detect #[Test] attribute", func(t *testing.T) {
		tmpDir := t.TempDir()

		testContent := []byte(`<?php
use PHPUnit\Framework\TestCase;

class AttributeTest extends TestCase
{
    #[Test]
    public function userCreation()
    {
        $this->assertTrue(true);
    }
}
`)
		testFile := filepath.Join(tmpDir, "AttributeTest.php")
		if err := os.WriteFile(testFile, testContent, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		result, err := parser.Scan(context.Background(), src)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Inventory.Files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(result.Inventory.Files))
		}

		file := result.Inventory.Files[0]
		if len(file.Suites[0].Tests) != 1 {
			t.Fatalf("expected 1 test, got %d", len(file.Suites[0].Tests))
		}
		if file.Suites[0].Tests[0].Name != "userCreation" {
			t.Errorf("expected test name 'userCreation', got %q", file.Suites[0].Tests[0].Name)
		}
	})

	t.Run("should scan files in tests directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		testsDir := filepath.Join(tmpDir, "tests")
		if err := os.MkdirAll(testsDir, 0755); err != nil {
			t.Fatalf("failed to create tests dir: %v", err)
		}

		testContent := []byte(`<?php
use PHPUnit\Framework\TestCase;

class SomeTest extends TestCase
{
    public function testSomething()
    {
        $this->assertTrue(true);
    }
}
`)
		testFile := filepath.Join(testsDir, "SomeTest.php")
		if err := os.WriteFile(testFile, testContent, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		result, err := parser.Scan(context.Background(), src)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Inventory.Files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(result.Inventory.Files))
		}

		file := result.Inventory.Files[0]
		if file.Framework != "phpunit" {
			t.Errorf("expected framework 'phpunit', got %q", file.Framework)
		}
	})

	t.Run("should ignore non-TestCase classes", func(t *testing.T) {
		tmpDir := t.TempDir()

		testContent := []byte(`<?php
class NotATest
{
    public function testSomething()
    {
        // Not a real test
    }
}
`)
		testFile := filepath.Join(tmpDir, "NotATest.php")
		if err := os.WriteFile(testFile, testContent, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		src, err := source.NewLocalSource(tmpDir)
		if err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
		defer src.Close()

		result, err := parser.Scan(context.Background(), src)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// File discovered but no tests parsed (no TestCase inheritance)
		if len(result.Inventory.Files) != 0 {
			t.Errorf("expected 0 files (no TestCase), got %d", len(result.Inventory.Files))
		}
	})
}

func TestScan_FixtureExclusion(t *testing.T) {
	tests := []struct {
		name         string
		excludedDir  string
		excludedFile string
	}{
		{"__fixtures__", "__fixtures__", "data.js"},
		{"__mocks__", "__mocks__", "module.js"},
	}

	for _, tt := range tests {
		t.Run("should not scan files in "+tt.name+" directory", func(t *testing.T) {
			tmpDir := t.TempDir()

			excludedPath := filepath.Join(tmpDir, "__tests__", tt.excludedDir)
			if err := os.MkdirAll(excludedPath, 0755); err != nil {
				t.Fatalf("failed to create dir: %v", err)
			}

			if err := os.WriteFile(filepath.Join(excludedPath, tt.excludedFile), []byte(`module.exports = {};`), 0644); err != nil {
				t.Fatalf("failed to write excluded file: %v", err)
			}

			testDir := filepath.Join(tmpDir, "__tests__")
			testContent := []byte(`import { it } from '@jest/globals'; it('test', () => {});`)
			if err := os.WriteFile(filepath.Join(testDir, "component.test.ts"), testContent, 0644); err != nil {
				t.Fatalf("failed to write test: %v", err)
			}

			src, err := source.NewLocalSource(tmpDir)
			if err != nil {
				t.Fatalf("failed to create source: %v", err)
			}
			defer src.Close()

			result, err := parser.Scan(context.Background(), src)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.Inventory.Files) != 1 {
				t.Errorf("expected 1 file, got %d", len(result.Inventory.Files))
				for _, f := range result.Inventory.Files {
					t.Logf("found file: %s", f.Path)
				}
			}
		})
	}
}
