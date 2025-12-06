//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/specvital/core/pkg/parser"

	_ "github.com/specvital/core/pkg/parser/strategies/gotesting"
	_ "github.com/specvital/core/pkg/parser/strategies/jest"
	_ "github.com/specvital/core/pkg/parser/strategies/playwright"
	_ "github.com/specvital/core/pkg/parser/strategies/vitest"
)

const scanTimeout = 10 * time.Minute

func TestSingleFramework(t *testing.T) {
	repos, err := LoadRepos()
	if err != nil {
		t.Fatalf("load repos.yaml: %v", err)
	}

	for _, repo := range repos.Repositories {
		repo := repo
		t.Run(repo.Name, func(t *testing.T) {
			t.Parallel()

			result, err := CloneRepo(repo)
			if err != nil {
				t.Fatalf("clone %s: %v", repo.Name, err)
			}

			if result.FromCache {
				t.Logf("using cached repository: %s", result.Path)
			} else {
				t.Logf("cloned repository: %s", result.Path)
			}

			ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
			defer cancel()

			scanResult, err := parser.Scan(ctx, result.Path)
			if err != nil {
				t.Fatalf("scan %s: %v", repo.Name, err)
			}

			t.Logf("scan stats: files=%d, matched=%d, tests=%d, duration=%v",
				scanResult.Stats.FilesScanned,
				scanResult.Stats.FilesMatched,
				scanResult.Inventory.CountTests(),
				scanResult.Stats.Duration,
			)

			if scanResult.Stats.FilesMatched == 0 {
				t.Errorf("expected at least 1 matched file, got 0")
			}

			if scanResult.Inventory.CountTests() == 0 {
				t.Errorf("expected at least 1 test, got 0")
			}

			frameworkCount := countByFramework(scanResult)
			if count, ok := frameworkCount[repo.Framework]; !ok || count == 0 {
				t.Errorf("expected framework %s files, got: %v", repo.Framework, frameworkCount)
			}

			t.Logf("framework distribution: %v", frameworkCount)
		})
	}
}

func countByFramework(result *parser.ScanResult) map[string]int {
	counts := make(map[string]int)
	for _, file := range result.Inventory.Files {
		counts[file.Framework]++
	}
	return counts
}
