package domain

import "testing"

func TestTestSuite_CountTests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		suite TestSuite
		want  int
	}{
		{
			name:  "should return zero for empty suite",
			suite: TestSuite{},
			want:  0,
		},
		{
			name: "should count direct tests only",
			suite: TestSuite{
				Tests: []Test{{Name: "t1"}, {Name: "t2"}},
			},
			want: 2,
		},
		{
			name: "should count nested suite tests",
			suite: TestSuite{
				Tests: []Test{{Name: "t1"}},
				Suites: []TestSuite{
					{Tests: []Test{{Name: "nested"}}},
				},
			},
			want: 2,
		},
		{
			name: "should count deeply nested tests",
			suite: TestSuite{
				Tests: []Test{{Name: "t1"}},
				Suites: []TestSuite{
					{
						Tests: []Test{{Name: "s1t1"}},
						Suites: []TestSuite{
							{Tests: []Test{{Name: "s2t1"}, {Name: "s2t2"}}},
						},
					},
				},
			},
			want: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// When
			got := tt.suite.CountTests()

			// Then
			if got != tt.want {
				t.Errorf("CountTests() = %d, want %d", got, tt.want)
			}
		})
	}
}
