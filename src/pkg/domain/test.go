package domain

type Test struct {
	Name     string     `json:"name"`
	Status   TestStatus `json:"status"`
	Location Location   `json:"location"`
}

type TestSuite struct {
	Name     string      `json:"name"`
	Status   TestStatus  `json:"status"`
	Location Location    `json:"location"`
	Tests    []Test      `json:"tests,omitempty"`
	Suites   []TestSuite `json:"suites,omitempty"`
}

func (s *TestSuite) CountTests() int {
	count := len(s.Tests)
	for _, sub := range s.Suites {
		count += sub.CountTests()
	}
	return count
}
