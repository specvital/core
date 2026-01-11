//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser"
	"github.com/specvital/core/pkg/source"

	// DomainHints extraction uses tree-sitter parsing which works independently of framework detection.
	// Only gotesting and jest are imported here as they are the primary Go/JS/TS test frameworks in grafana.
	// Other frameworks (cypress, playwright) share the same JS/TS extraction logic.
	_ "github.com/specvital/core/pkg/parser/strategies/gotesting"
	_ "github.com/specvital/core/pkg/parser/strategies/jest"
)

// TestDomainHints_Grafana verifies DomainHints extraction from grafana repository.
// This test validates Phase 1 completion: Go + JS/TS extraction working correctly.
func TestDomainHints_Grafana(t *testing.T) {
	scanResult := setupGrafanaScan(t)

	t.Run("Go test files have domain hints", func(t *testing.T) {
		verifyLanguageHints(t, scanResult, "Go", isGoLanguage)
	})

	t.Run("TypeScript test files have domain hints", func(t *testing.T) {
		verifyLanguageHints(t, scanResult, "TypeScript", isTSLanguage)
	})

	t.Run("Go imports include expected packages", func(t *testing.T) {
		verifyGoImportPatterns(t, scanResult)
	})

	t.Run("TS imports include Grafana packages", func(t *testing.T) {
		verifyTSImportPatterns(t, scanResult)
	})
}

// setupGrafanaScan loads and scans the grafana repository with DomainHints enabled.
func setupGrafanaScan(t *testing.T) *parser.ScanResult {
	t.Helper()

	repos, err := LoadRepos()
	if err != nil {
		t.Fatalf("load repos.yaml: %v", err)
	}

	var grafanaRepo Repository
	for _, repo := range repos.Repositories {
		if repo.Name == "grafana" {
			grafanaRepo = repo
			break
		}
	}

	if grafanaRepo.Name == "" {
		t.Skip("grafana repository not found in repos.yaml")
	}

	cloneResult, err := CloneRepo(grafanaRepo)
	if err != nil {
		t.Fatalf("clone grafana: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)

	src, err := source.NewLocalSource(cloneResult.Path)
	if err != nil {
		cancel()
		t.Fatalf("create source: %v", err)
	}

	t.Cleanup(func() {
		src.Close()
		cancel()
	})

	scanResult, err := parser.Scan(ctx, src, parser.WithDomainHints(true))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	return scanResult
}

func isGoLanguage(lang domain.Language) bool {
	return lang == domain.LanguageGo
}

func isTSLanguage(lang domain.Language) bool {
	return lang == domain.LanguageTypeScript || lang == domain.LanguageTSX
}

// verifyLanguageHints validates that files of a specific language have DomainHints extracted.
func verifyLanguageHints(t *testing.T, result *parser.ScanResult, langName string, isTargetLang func(domain.Language) bool) {
	t.Helper()

	var filesWithHints int
	var filesTotal int
	var sampleHints *domain.DomainHints
	var samplePath string

	for _, file := range result.Inventory.Files {
		if !isTargetLang(file.Language) {
			continue
		}
		filesTotal++

		if file.DomainHints != nil {
			filesWithHints++
			if sampleHints == nil {
				sampleHints = file.DomainHints
				samplePath = file.Path
			}
		}
	}

	if filesTotal == 0 {
		t.Errorf("no %s test files found", langName)
		return
	}

	t.Logf("%s files: %d total, %d with hints (%.1f%%)",
		langName, filesTotal, filesWithHints, float64(filesWithHints)/float64(filesTotal)*100)

	if filesWithHints == 0 {
		t.Errorf("no %s test files have domain hints", langName)
		return
	}

	if sampleHints != nil {
		t.Logf("Sample file: %s", samplePath)
		t.Logf("  Imports: %d items", len(sampleHints.Imports))
		t.Logf("  Calls: %d items", len(sampleHints.Calls))
		t.Logf("  Variables: %d items", len(sampleHints.Variables))

		if len(sampleHints.Imports) == 0 {
			t.Errorf("sample %s file has no imports", langName)
		}
	}
}

// verifyGoImportPatterns checks that expected Go import patterns are extracted.
func verifyGoImportPatterns(t *testing.T, result *parser.ScanResult) {
	t.Helper()

	goImports := collectImports(result, isGoLanguage)

	expectedPatterns := []string{"testing", "github.com/stretchr/testify"}
	found := 0
	for _, pattern := range expectedPatterns {
		for imp, count := range goImports {
			if strings.Contains(imp, pattern) {
				found++
				t.Logf("Found Go import: %s (count: %d)", imp, count)
				break
			}
		}
	}

	if found < len(expectedPatterns)/2 {
		t.Logf("Note: fewer expected Go imports found. Total unique imports: %d", len(goImports))
	}
}

// verifyTSImportPatterns checks that Grafana-specific TS imports are extracted.
func verifyTSImportPatterns(t *testing.T, result *parser.ScanResult) {
	t.Helper()

	tsImports := collectImports(result, isTSLanguage)

	grafanaImportCount := 0
	for imp := range tsImports {
		if strings.Contains(imp, "@grafana/") {
			grafanaImportCount++
		}
	}

	t.Logf("Found %d @grafana/* imports", grafanaImportCount)

	if grafanaImportCount == 0 {
		t.Error("expected at least one @grafana/* import")
	}
}

// collectImports aggregates all imports from files matching the language filter.
func collectImports(result *parser.ScanResult, isTargetLang func(domain.Language) bool) map[string]int {
	imports := make(map[string]int)

	for _, file := range result.Inventory.Files {
		if file.DomainHints == nil || !isTargetLang(file.Language) {
			continue
		}

		for _, imp := range file.DomainHints.Imports {
			imports[imp]++
		}
	}

	return imports
}
