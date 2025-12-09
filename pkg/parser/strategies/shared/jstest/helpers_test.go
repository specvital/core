package jstest

import (
	"testing"

	"github.com/specvital/core/pkg/domain"
)

func TestUnquoteString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "should unquote double quotes",
			input: `"hello"`,
			want:  "hello",
		},
		{
			name:  "should unquote single quotes",
			input: `'hello'`,
			want:  "hello",
		},
		{
			name:  "should unquote backticks",
			input: "`hello`",
			want:  "hello",
		},
		{
			name:  "should return short string as-is",
			input: "a",
			want:  "a",
		},
		{
			name:  "should return unquoted string as-is",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "should handle mismatched quotes",
			input: `"hello'`,
			want:  `"hello'`,
		},
		{
			name:  "should handle escaped single quotes",
			input: `'it\'s working'`,
			want:  "it's working",
		},
		{
			name:  "should handle escaped double quotes in double quoted string",
			input: `"say \"hello\""`,
			want:  `say "hello"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := UnquoteString(tt.input)

			if got != tt.want {
				t.Errorf("UnquoteString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatEachName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		template string
		data     string
		want     string
	}{
		{
			name:     "should replace %s placeholder",
			template: "test %s",
			data:     "value",
			want:     "test value",
		},
		{
			name:     "should replace %d placeholder",
			template: "test %d",
			data:     "123",
			want:     "test 123",
		},
		{
			name:     "should replace %p placeholder",
			template: "test %p",
			data:     "data",
			want:     "test data",
		},
		{
			name:     "should replace multiple placeholders",
			template: "test %s and %d",
			data:     "foo, 42",
			want:     "test foo and 42",
		},
		{
			name:     "should keep unreplaced placeholders",
			template: "test %s %s %s",
			data:     "first, second",
			want:     "test first second %s",
		},
		{
			name:     "should handle %% escape",
			template: "100%% complete",
			data:     "",
			want:     "100% complete",
		},
		{
			name:     "should return template if no placeholder",
			template: "no placeholder",
			data:     "data",
			want:     "no placeholder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := FormatEachName(tt.template, tt.data)

			if got != tt.want {
				t.Errorf("FormatEachName(%q, %q) = %q, want %q", tt.template, tt.data, got, tt.want)
			}
		})
	}
}

func TestParseModifierStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		modifier string
		want     domain.TestStatus
	}{
		{
			name:     "should return skipped for skip",
			modifier: "skip",
			want:     domain.TestStatusSkipped,
		},
		{
			name:     "should return todo for todo",
			modifier: "todo",
			want:     domain.TestStatusTodo,
		},
		{
			name:     "should return focused for only",
			modifier: "only",
			want:     domain.TestStatusFocused,
		},
		{
			name:     "should return active for unknown",
			modifier: "unknown",
			want:     domain.TestStatusActive,
		},
		{
			name:     "should return active for empty",
			modifier: "",
			want:     domain.TestStatusActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ParseModifierStatus(tt.modifier)

			if got != tt.want {
				t.Errorf("ParseModifierStatus(%q) = %q, want %q", tt.modifier, got, tt.want)
			}
		})
	}
}

func TestSkippedFunctionAliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		alias string
		want  string
	}{
		{"xdescribe", FuncDescribe},
		{"xit", FuncIt},
		{"xtest", FuncTest},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			t.Parallel()

			got, ok := SkippedFunctionAliases[tt.alias]
			if !ok {
				t.Fatalf("SkippedFunctionAliases[%q] not found", tt.alias)
			}
			if got != tt.want {
				t.Errorf("SkippedFunctionAliases[%q] = %q, want %q", tt.alias, got, tt.want)
			}
		})
	}
}

func TestFocusedFunctionAliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		alias string
		want  string
	}{
		{"fdescribe", FuncDescribe},
		{"fit", FuncIt},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			t.Parallel()

			got, ok := FocusedFunctionAliases[tt.alias]
			if !ok {
				t.Fatalf("FocusedFunctionAliases[%q] not found", tt.alias)
			}
			if got != tt.want {
				t.Errorf("FocusedFunctionAliases[%q] = %q, want %q", tt.alias, got, tt.want)
			}
		})
	}
}

func TestSupportedExtensions(t *testing.T) {
	t.Parallel()

	supported := []string{".ts", ".tsx", ".js", ".jsx"}
	unsupported := []string{".go", ".py", ".rb", ".java"}

	for _, ext := range supported {
		t.Run("should support "+ext, func(t *testing.T) {
			t.Parallel()
			if !SupportedExtensions[ext] {
				t.Errorf("SupportedExtensions[%q] = false, want true", ext)
			}
		})
	}

	for _, ext := range unsupported {
		t.Run("should not support "+ext, func(t *testing.T) {
			t.Parallel()
			if SupportedExtensions[ext] {
				t.Errorf("SupportedExtensions[%q] = true, want false", ext)
			}
		})
	}
}
