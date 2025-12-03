package domain

type TestFile struct {
	Framework string      `json:"framework"`
	Language  Language    `json:"language"`
	Path      string      `json:"path"`
	Suites    []TestSuite `json:"suites,omitempty"`
	Tests     []Test      `json:"tests,omitempty"`
}

func (f *TestFile) CountTests() int {
	count := len(f.Tests)
	for _, s := range f.Suites {
		count += s.CountTests()
	}
	return count
}

type Inventory struct {
	Files    []TestFile `json:"files"`
	RootPath string     `json:"rootPath"`
}

func (inv Inventory) CountTests() int {
	count := 0
	for _, f := range inv.Files {
		count += f.CountTests()
	}
	return count
}
