package domain

// Test represents a single test case.
type Test struct {
	Name     string     `json:"name"`
	Status   TestStatus `json:"status"`
	Location Location   `json:"location"`
}

// TestSuite represents a group of tests (describe block).
type TestSuite struct {
	Name     string      `json:"name"`
	Status   TestStatus  `json:"status"`
	Location Location    `json:"location"`
	Tests    []Test      `json:"tests,omitempty"`
	Suites   []TestSuite `json:"suites,omitempty"` // Nested describe blocks
}

// CountTests returns the total number of tests in a suite (recursive).
func (s *TestSuite) CountTests() int {
	count := len(s.Tests)
	for _, sub := range s.Suites {
		count += sub.CountTests()
	}
	return count
}
