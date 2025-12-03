package domain

type Test struct {
	Location Location   `json:"location"`
	Name     string     `json:"name"`
	Status   TestStatus `json:"status"`
}

type TestSuite struct {
	Location Location    `json:"location"`
	Name     string      `json:"name"`
	Status   TestStatus  `json:"status"`
	Suites   []TestSuite `json:"suites,omitempty"`
	Tests    []Test      `json:"tests,omitempty"`
}

func (s *TestSuite) CountTests() int {
	count := len(s.Tests)
	for _, sub := range s.Suites {
		count += sub.CountTests()
	}
	return count
}
