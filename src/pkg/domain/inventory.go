package domain

// TestFile represents a parsed test file.
type TestFile struct {
	Path      string      `json:"path"`
	Language  Language    `json:"language"`
	Framework string      `json:"framework"`
	Suites    []TestSuite `json:"suites,omitempty"`
	Tests     []Test      `json:"tests,omitempty"` // Top-level tests (not in describe)
}

// CountTests returns the total number of tests in a file.
func (f *TestFile) CountTests() int {
	count := len(f.Tests)
	for _, s := range f.Suites {
		count += s.CountTests()
	}
	return count
}

// Inventory represents the complete test inventory of a project.
type Inventory struct {
	RootPath string     `json:"rootPath"`
	Files    []TestFile `json:"files"`
}

// CountTests returns the total number of tests in the inventory.
func (inv *Inventory) CountTests() int {
	count := 0
	for _, f := range inv.Files {
		count += f.CountTests()
	}
	return count
}
