//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/specvital/core/pkg/parser"
)

// Snapshot represents a golden snapshot for a repository scan result.
type Snapshot struct {
	Repository         string         `json:"repository"`
	Ref                string         `json:"ref"`
	ExpectedFrameworks []string       `json:"expectedFrameworks"`
	FileCount          int            `json:"fileCount"`
	TestCount          int            `json:"testCount"`
	FrameworkCounts    map[string]int `json:"frameworkCounts"`
	SampleFiles        []SnapshotFile `json:"sampleFiles"`
	Stats              SnapshotStats  `json:"stats"`
}

// SnapshotFile represents a sample of parsed test files in the snapshot.
type SnapshotFile struct {
	Path       string `json:"path"`
	Framework  string `json:"framework"`
	SuiteCount int    `json:"suiteCount"`
	TestCount  int    `json:"testCount"`
}

// SnapshotStats contains scan statistics for comparison.
type SnapshotStats struct {
	FilesMatched int `json:"filesMatched"`
	FilesScanned int `json:"filesScanned"`
}

// SnapshotFromScanResult creates a Snapshot from a scan result.
func SnapshotFromScanResult(repo Repository, result *parser.ScanResult, rootPath string) *Snapshot {
	frameworkCounts := make(map[string]int)
	for _, file := range result.Inventory.Files {
		frameworkCounts[file.Framework]++
	}

	sampleFiles := extractSampleFiles(result, rootPath, 20)

	return &Snapshot{
		Repository:         repo.Name,
		Ref:                repo.Ref,
		ExpectedFrameworks: repo.Frameworks,
		FileCount:          len(result.Inventory.Files),
		TestCount:          result.Inventory.CountTests(),
		FrameworkCounts:    frameworkCounts,
		SampleFiles:        sampleFiles,
		Stats: SnapshotStats{
			FilesMatched: result.Stats.FilesMatched,
			FilesScanned: result.Stats.FilesScanned,
		},
	}
}

// extractSampleFiles extracts up to maxSamples files, sorted by path for determinism.
func extractSampleFiles(result *parser.ScanResult, rootPath string, maxSamples int) []SnapshotFile {
	files := result.Inventory.Files
	if len(files) == 0 {
		return nil
	}

	// Sort by path for deterministic output
	sorted := make([]SnapshotFile, 0, len(files))
	for _, f := range files {
		relPath, err := filepath.Rel(rootPath, f.Path)
		if err != nil {
			relPath = f.Path
		}
		sorted = append(sorted, SnapshotFile{
			Path:       relPath,
			Framework:  f.Framework,
			SuiteCount: len(f.Suites),
			TestCount:  f.CountTests(),
		})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Path < sorted[j].Path
	})

	if len(sorted) > maxSamples {
		sorted = sorted[:maxSamples]
	}
	return sorted
}

// SaveSnapshot saves a snapshot to the golden directory.
func SaveSnapshot(snapshot *Snapshot) error {
	goldenDir, err := getGoldenDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(goldenDir, 0755); err != nil {
		return fmt.Errorf("create golden dir: %w", err)
	}

	path := filepath.Join(goldenDir, snapshotFilename(snapshot.Repository, snapshot.Ref))
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}

	return nil
}

// LoadSnapshot loads a snapshot from the golden directory.
func LoadSnapshot(repoName, ref string) (*Snapshot, error) {
	goldenDir, err := getGoldenDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(goldenDir, snapshotFilename(repoName, ref))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("snapshot not found: %s (run with -update to create)", path)
		}
		return nil, fmt.Errorf("read snapshot: %w", err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}

	return &snapshot, nil
}

// SnapshotDiff represents differences between expected and actual snapshots.
type SnapshotDiff struct {
	FileCountDiff       int
	TestCountDiff       int
	FrameworkCountDiffs map[string]FrameworkDiff
	MissingFiles        []string
	ExtraFiles          []string
}

// FrameworkDiff represents the difference in file count for a framework.
type FrameworkDiff struct {
	Expected int
	Actual   int
}

// IsEmpty returns true if there are no differences.
func (d *SnapshotDiff) IsEmpty() bool {
	return d.FileCountDiff == 0 &&
		d.TestCountDiff == 0 &&
		len(d.FrameworkCountDiffs) == 0 &&
		len(d.MissingFiles) == 0 &&
		len(d.ExtraFiles) == 0
}

// String returns a human-readable diff summary.
func (d *SnapshotDiff) String() string {
	if d.IsEmpty() {
		return "no differences"
	}

	var sb strings.Builder

	if d.FileCountDiff != 0 {
		sb.WriteString(fmt.Sprintf("  file count: %+d\n", d.FileCountDiff))
	}
	if d.TestCountDiff != 0 {
		sb.WriteString(fmt.Sprintf("  test count: %+d\n", d.TestCountDiff))
	}

	for fw, diff := range d.FrameworkCountDiffs {
		sb.WriteString(fmt.Sprintf("  framework %s: expected %d, got %d\n", fw, diff.Expected, diff.Actual))
	}

	if len(d.MissingFiles) > 0 {
		sb.WriteString(fmt.Sprintf("  missing files (%d):\n", len(d.MissingFiles)))
		for _, f := range d.MissingFiles {
			if len(d.MissingFiles) <= 10 {
				sb.WriteString(fmt.Sprintf("    - %s\n", f))
			}
		}
		if len(d.MissingFiles) > 10 {
			sb.WriteString(fmt.Sprintf("    ... and %d more\n", len(d.MissingFiles)-10))
		}
	}

	if len(d.ExtraFiles) > 0 {
		sb.WriteString(fmt.Sprintf("  extra files (%d):\n", len(d.ExtraFiles)))
		for i, f := range d.ExtraFiles {
			if i < 10 {
				sb.WriteString(fmt.Sprintf("    + %s\n", f))
			}
		}
		if len(d.ExtraFiles) > 10 {
			sb.WriteString(fmt.Sprintf("    ... and %d more\n", len(d.ExtraFiles)-10))
		}
	}

	return sb.String()
}

// CompareSnapshots compares an expected snapshot with an actual scan result.
func CompareSnapshots(expected *Snapshot, actual *Snapshot) *SnapshotDiff {
	diff := &SnapshotDiff{
		FileCountDiff:       actual.FileCount - expected.FileCount,
		TestCountDiff:       actual.TestCount - expected.TestCount,
		FrameworkCountDiffs: make(map[string]FrameworkDiff),
	}

	// Compare framework counts
	allFrameworks := make(map[string]bool)
	for fw := range expected.FrameworkCounts {
		allFrameworks[fw] = true
	}
	for fw := range actual.FrameworkCounts {
		allFrameworks[fw] = true
	}

	for fw := range allFrameworks {
		expectedCount := expected.FrameworkCounts[fw]
		actualCount := actual.FrameworkCounts[fw]
		if expectedCount != actualCount {
			diff.FrameworkCountDiffs[fw] = FrameworkDiff{
				Expected: expectedCount,
				Actual:   actualCount,
			}
		}
	}

	// Compare sample files
	expectedPaths := make(map[string]bool)
	for _, f := range expected.SampleFiles {
		expectedPaths[f.Path] = true
	}

	actualPaths := make(map[string]bool)
	for _, f := range actual.SampleFiles {
		actualPaths[f.Path] = true
	}

	for path := range expectedPaths {
		if !actualPaths[path] {
			diff.MissingFiles = append(diff.MissingFiles, path)
		}
	}

	for path := range actualPaths {
		if !expectedPaths[path] {
			diff.ExtraFiles = append(diff.ExtraFiles, path)
		}
	}

	sort.Strings(diff.MissingFiles)
	sort.Strings(diff.ExtraFiles)

	return diff
}

func getGoldenDir() (string, error) {
	testDataDir, err := getTestDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(testDataDir, "golden"), nil
}

func snapshotFilename(repoName, ref string) string {
	safeName := unsafePathChars.ReplaceAllString(repoName, "_")
	safeRef := unsafePathChars.ReplaceAllString(ref, "_")
	return fmt.Sprintf("%s-%s.json", safeName, safeRef)
}
