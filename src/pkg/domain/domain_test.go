package domain

import "testing"

func TestInventory_CountTests(t *testing.T) {
	inv := &Inventory{
		Files: []TestFile{
			{
				Tests: []Test{{Name: "t1"}, {Name: "t2"}},
				Suites: []TestSuite{
					{
						Tests: []Test{{Name: "s1t1"}},
						Suites: []TestSuite{
							{Tests: []Test{{Name: "s2t1"}, {Name: "s2t2"}}},
						},
					},
				},
			},
			{
				Tests: []Test{{Name: "f2t1"}},
			},
		},
	}

	// Total: 2 + 1 + 2 + 1 = 6
	if got := inv.CountTests(); got != 6 {
		t.Errorf("CountTests() = %d, want 6", got)
	}
}

func TestTestFile_CountTests(t *testing.T) {
	file := TestFile{
		Tests: []Test{{Name: "t1"}},
		Suites: []TestSuite{
			{Tests: []Test{{Name: "s1"}, {Name: "s2"}}},
		},
	}

	if got := file.CountTests(); got != 3 {
		t.Errorf("CountTests() = %d, want 3", got)
	}
}

func TestTestSuite_CountTests(t *testing.T) {
	suite := TestSuite{
		Tests: []Test{{Name: "t1"}},
		Suites: []TestSuite{
			{Tests: []Test{{Name: "nested"}}},
		},
	}

	if got := suite.CountTests(); got != 2 {
		t.Errorf("CountTests() = %d, want 2", got)
	}
}
