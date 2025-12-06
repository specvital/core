//go:build integration

package integration

import (
	"testing"
)

func TestSnapshotDiff_Empty(t *testing.T) {
	diff := &SnapshotDiff{
		FrameworkCountDiffs: make(map[string]FrameworkDiff),
	}

	if !diff.IsEmpty() {
		t.Error("expected empty diff to be empty")
	}
}

func TestSnapshotDiff_FileCountDiff(t *testing.T) {
	diff := &SnapshotDiff{
		FileCountDiff:       5,
		FrameworkCountDiffs: make(map[string]FrameworkDiff),
	}

	if diff.IsEmpty() {
		t.Error("expected diff with file count change to not be empty")
	}

	output := diff.String()
	if output == "no differences" {
		t.Error("expected diff output to show differences")
	}
}

func TestSnapshotDiff_TestCountDiff(t *testing.T) {
	diff := &SnapshotDiff{
		TestCountDiff:       -10,
		FrameworkCountDiffs: make(map[string]FrameworkDiff),
	}

	if diff.IsEmpty() {
		t.Error("expected diff with test count change to not be empty")
	}
}

func TestSnapshotDiff_FrameworkCountDiff(t *testing.T) {
	diff := &SnapshotDiff{
		FrameworkCountDiffs: map[string]FrameworkDiff{
			"jest": {Expected: 10, Actual: 8},
		},
	}

	if diff.IsEmpty() {
		t.Error("expected diff with framework count change to not be empty")
	}
}

func TestSnapshotDiff_MissingFiles(t *testing.T) {
	diff := &SnapshotDiff{
		FrameworkCountDiffs: make(map[string]FrameworkDiff),
		MissingFiles:        []string{"src/test.ts"},
	}

	if diff.IsEmpty() {
		t.Error("expected diff with missing files to not be empty")
	}
}

func TestSnapshotDiff_ExtraFiles(t *testing.T) {
	diff := &SnapshotDiff{
		FrameworkCountDiffs: make(map[string]FrameworkDiff),
		ExtraFiles:          []string{"src/new-test.ts"},
	}

	if diff.IsEmpty() {
		t.Error("expected diff with extra files to not be empty")
	}
}

func TestCompareSnapshots_Identical(t *testing.T) {
	expected := &Snapshot{
		Repository:        "test",
		Ref:               "v1.0.0",
		ExpectedFramework: "jest",
		FileCount:         5,
		TestCount:         20,
		FrameworkCounts:   map[string]int{"jest": 5},
		SampleFiles: []SnapshotFile{
			{Path: "src/a.test.ts", Framework: "jest", SuiteCount: 1, TestCount: 4},
		},
	}

	actual := &Snapshot{
		Repository:        "test",
		Ref:               "v1.0.0",
		ExpectedFramework: "jest",
		FileCount:         5,
		TestCount:         20,
		FrameworkCounts:   map[string]int{"jest": 5},
		SampleFiles: []SnapshotFile{
			{Path: "src/a.test.ts", Framework: "jest", SuiteCount: 1, TestCount: 4},
		},
	}

	diff := CompareSnapshots(expected, actual)
	if !diff.IsEmpty() {
		t.Errorf("expected no diff for identical snapshots: %s", diff.String())
	}
}

func TestCompareSnapshots_FileCountChanged(t *testing.T) {
	expected := &Snapshot{
		FileCount:       5,
		TestCount:       20,
		FrameworkCounts: map[string]int{"jest": 5},
	}

	actual := &Snapshot{
		FileCount:       7,
		TestCount:       20,
		FrameworkCounts: map[string]int{"jest": 7},
	}

	diff := CompareSnapshots(expected, actual)
	if diff.IsEmpty() {
		t.Error("expected diff for different file counts")
	}

	if diff.FileCountDiff != 2 {
		t.Errorf("expected file count diff of 2, got %d", diff.FileCountDiff)
	}
}

func TestCompareSnapshots_TestCountChanged(t *testing.T) {
	expected := &Snapshot{
		FileCount:       5,
		TestCount:       20,
		FrameworkCounts: map[string]int{"jest": 5},
	}

	actual := &Snapshot{
		FileCount:       5,
		TestCount:       15,
		FrameworkCounts: map[string]int{"jest": 5},
	}

	diff := CompareSnapshots(expected, actual)
	if diff.IsEmpty() {
		t.Error("expected diff for different test counts")
	}

	if diff.TestCountDiff != -5 {
		t.Errorf("expected test count diff of -5, got %d", diff.TestCountDiff)
	}
}

func TestCompareSnapshots_FrameworkCountChanged(t *testing.T) {
	expected := &Snapshot{
		FileCount:       10,
		TestCount:       50,
		FrameworkCounts: map[string]int{"jest": 8, "vitest": 2},
	}

	actual := &Snapshot{
		FileCount:       10,
		TestCount:       50,
		FrameworkCounts: map[string]int{"jest": 6, "vitest": 4},
	}

	diff := CompareSnapshots(expected, actual)
	if diff.IsEmpty() {
		t.Error("expected diff for different framework counts")
	}

	if len(diff.FrameworkCountDiffs) != 2 {
		t.Errorf("expected 2 framework diffs, got %d", len(diff.FrameworkCountDiffs))
	}

	jestDiff, ok := diff.FrameworkCountDiffs["jest"]
	if !ok {
		t.Error("expected jest diff")
	}
	if jestDiff.Expected != 8 || jestDiff.Actual != 6 {
		t.Errorf("expected jest diff 8->6, got %d->%d", jestDiff.Expected, jestDiff.Actual)
	}
}

func TestCompareSnapshots_SampleFilesChanged(t *testing.T) {
	expected := &Snapshot{
		FileCount:       2,
		TestCount:       10,
		FrameworkCounts: map[string]int{"jest": 2},
		SampleFiles: []SnapshotFile{
			{Path: "src/a.test.ts"},
			{Path: "src/b.test.ts"},
		},
	}

	actual := &Snapshot{
		FileCount:       2,
		TestCount:       10,
		FrameworkCounts: map[string]int{"jest": 2},
		SampleFiles: []SnapshotFile{
			{Path: "src/a.test.ts"},
			{Path: "src/c.test.ts"},
		},
	}

	diff := CompareSnapshots(expected, actual)
	if diff.IsEmpty() {
		t.Error("expected diff for different sample files")
	}

	if len(diff.MissingFiles) != 1 || diff.MissingFiles[0] != "src/b.test.ts" {
		t.Errorf("expected missing file src/b.test.ts, got %v", diff.MissingFiles)
	}

	if len(diff.ExtraFiles) != 1 || diff.ExtraFiles[0] != "src/c.test.ts" {
		t.Errorf("expected extra file src/c.test.ts, got %v", diff.ExtraFiles)
	}
}
