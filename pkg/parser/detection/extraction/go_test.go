package extraction

import (
	"context"
	"slices"
	"testing"
)

func TestExtractGoImports(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "single import",
			content: `import "testing"`,
			want:    []string{"testing"},
		},
		{
			name:    "aliased import",
			content: `import t "testing"`,
			want:    []string{"testing"},
		},
		{
			name: "grouped imports",
			content: `import (
	"fmt"
	"testing"
)`,
			want: []string{"fmt", "testing"},
		},
		{
			name: "mixed grouped imports with aliases",
			content: `import (
	"context"
	t "testing"
	. "github.com/onsi/gomega"
)`,
			want: []string{"context", "testing", "github.com/onsi/gomega"},
		},
		{
			name: "full go file",
			content: `package main

import (
	"context"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestSomething(t *testing.T) {
	fmt.Println("hello")
}`,
			want: []string{"context", "testing", "github.com/stretchr/testify/assert"},
		},
		{
			name:    "no imports",
			content: `package main`,
			want:    nil,
		},
		{
			name: "multiple import declarations",
			content: `import "fmt"
import "testing"`,
			want: []string{"fmt", "testing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ExtractGoImports(context.Background(), []byte(tt.content))
			if !slices.Equal(got, tt.want) {
				t.Errorf("ExtractGoImports() = %v, want %v", got, tt.want)
			}
		})
	}
}
