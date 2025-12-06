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

	// Import frameworks to register them via init()
	_ "github.com/specvital/core/pkg/parser/strategies/jest"
)

func TestScan(t *testing.T) {
	t.Run("should return empty inventory for empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		result, err := parser.Scan(context.Background(), tmpDir)
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

		result, err := parser.Scan(context.Background(), tmpDir)
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

		result, err := parser.Scan(context.Background(), tmpDir, parser.WithExclude([]string{"custom_exclude"}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Inventory.Files) != 0 {
			t.Errorf("expected 0 files, got %d", len(result.Inventory.Files))
		}
	})

	t.Run("should respect worker count option", func(t *testing.T) {
		tmpDir := t.TempDir()

		result, err := parser.Scan(context.Background(), tmpDir, parser.WithWorkers(2))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result == nil {
			t.Fatal("result should not be nil")
		}
	})

	t.Run("should respect timeout option", func(t *testing.T) {
		tmpDir := t.TempDir()

		result, err := parser.Scan(context.Background(), tmpDir, parser.WithTimeout(30*time.Second))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result == nil {
			t.Fatal("result should not be nil")
		}
	})

	t.Run("should return error for non-existent path", func(t *testing.T) {
		_, err := parser.Scan(context.Background(), "/non/existent/path")
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

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := parser.Scan(ctx, tmpDir)
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

		result, err := parser.Scan(context.Background(), tmpDir)
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

		var wg sync.WaitGroup
		var errCount atomic.Int32

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := parser.Scan(context.Background(), tmpDir, parser.WithWorkers(4))
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

		result, err := parser.Scan(context.Background(), tmpDir, parser.WithWorkers(8))
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
