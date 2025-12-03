package domain

type TestFile struct {
	Path      string      `json:"path"`
	Language  Language    `json:"language"`
	Framework string      `json:"framework"`
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
	RootPath string     `json:"rootPath"`
	Files    []TestFile `json:"files"`
}

func (inv *Inventory) CountTests() int {
	count := 0
	for _, f := range inv.Files {
		count += f.CountTests()
	}
	return count
}
