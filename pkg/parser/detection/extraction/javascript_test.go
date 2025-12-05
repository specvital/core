package extraction

import (
	"context"
	"regexp"
	"testing"
)

func TestMatchPatternExcludingComments(t *testing.T) {
	t.Parallel()

	pattern := regexp.MustCompile(`globals\s*:\s*true`)

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "pattern found without comments",
			content: `{ test: { globals: true } }`,
			want:    true,
		},
		{
			name:    "pattern in single-line comment",
			content: `{ test: { // globals: true } }`,
			want:    false,
		},
		{
			name:    "pattern in multi-line comment",
			content: `{ test: { /* globals: true */ } }`,
			want:    false,
		},
		{
			name:    "regression: glob pattern with /* in string",
			content: `{ test: { include: ["**/*.ts"], globals: true } }`,
			want:    true,
		},
		{
			name:    "regression: glob pattern with */ in string",
			content: `{ test: { exclude: ["view/**/*"], globals: true } }`,
			want:    true,
		},
		{
			name: "regression: multiple glob patterns",
			content: `{
				test: {
					include: ["extension/**/*.ts", "src/**/*.ts"],
					exclude: ["**/node_modules/**", "**/dist/**", "view/**/*"],
					globals: true,
				}
			}`,
			want: true,
		},
		{
			name:    "pattern after single-line comment",
			content: "{ test: { // comment\nglobals: true } }",
			want:    true,
		},
		{
			name:    "pattern after multi-line comment",
			content: `{ test: { /* comment */ globals: true } }`,
			want:    true,
		},
		{
			name:    "empty content",
			content: ``,
			want:    false,
		},
		{
			name:    "no pattern match",
			content: `{ test: { environment: 'node' } }`,
			want:    false,
		},
		// Fast path tests (no comment markers)
		{
			name:    "fast path: no comment markers",
			content: `{ test: { globals: true } }`,
			want:    true,
		},
		{
			name:    "fast path: no comment markers, no match",
			content: `{ test: { globals: false } }`,
			want:    false,
		},
		// Fallback path tests (malformed content)
		{
			name:    "fallback: malformed JS with pattern",
			content: `{ test: { globals: true } // unclosed`,
			want:    true,
		},
		{
			name:    "fallback: malformed JS pattern in comment",
			content: `{ test: { // globals: true`,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := MatchPatternExcludingComments(context.Background(), []byte(tt.content), pattern)
			if got != tt.want {
				t.Errorf("MatchPatternExcludingComments() = %v, want %v", got, tt.want)
			}
		})
	}
}
