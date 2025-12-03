package domain

import "testing"

func TestTestFile_CountTests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file TestFile
		want int
	}{
		{
			name: "should return zero for empty file",
			file: TestFile{},
			want: 0,
		},
		{
			name: "should count top-level tests",
			file: TestFile{
				Tests: []Test{{Name: "t1"}, {Name: "t2"}},
			},
			want: 2,
		},
		{
			name: "should count suite tests",
			file: TestFile{
				Tests: []Test{{Name: "t1"}},
				Suites: []TestSuite{
					{Tests: []Test{{Name: "s1"}, {Name: "s2"}}},
				},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// When
			got := tt.file.CountTests()

			// Then
			if got != tt.want {
				t.Errorf("CountTests() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestInventory_CountTests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		inv  Inventory
		want int
	}{
		{
			name: "should return zero for empty inventory",
			inv:  Inventory{},
			want: 0,
		},
		{
			name: "should count tests across multiple files",
			inv: Inventory{
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
			},
			want: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// When
			got := tt.inv.CountTests()

			// Then
			if got != tt.want {
				t.Errorf("CountTests() = %d, want %d", got, tt.want)
			}
		})
	}
}
