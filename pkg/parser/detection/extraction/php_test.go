package extraction

import (
	"context"
	"reflect"
	"testing"
)

func TestExtractPHPUses(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "single use statement",
			content: `<?php
use PHPUnit\Framework\TestCase;
`,
			expected: []string{"PHPUnit\\Framework\\TestCase"},
		},
		{
			name: "multiple use statements",
			content: `<?php
use PHPUnit\Framework\TestCase;
use Illuminate\Tests\Mail;
use App\Models\User;
`,
			expected: []string{
				"PHPUnit\\Framework\\TestCase",
				"Illuminate\\Tests\\Mail",
				"App\\Models\\User",
			},
		},
		{
			name: "use with alias",
			content: `<?php
use PHPUnit\Framework\TestCase as BaseTestCase;
`,
			expected: []string{"PHPUnit\\Framework\\TestCase as BaseTestCase"},
		},
		{
			name: "no use statements",
			content: `<?php
class SomeClass {}
`,
			expected: nil,
		},
		{
			name: "dedup duplicate uses",
			content: `<?php
use PHPUnit\Framework\TestCase;
use App\Models\User;
use PHPUnit\Framework\TestCase;
`,
			expected: []string{
				"PHPUnit\\Framework\\TestCase",
				"App\\Models\\User",
			},
		},
		{
			name: "use function and const ignored by matcher",
			content: `<?php
use function App\Helpers\format;
use const App\Constants\VERSION;
use PHPUnit\Framework\TestCase;
`,
			expected: []string{
				"function App\\Helpers\\format",
				"const App\\Constants\\VERSION",
				"PHPUnit\\Framework\\TestCase",
			},
		},
		{
			name: "grouped use statements",
			content: `<?php
use PHPUnit\Framework\{TestCase, Assert};
`,
			expected: []string{"PHPUnit\\Framework\\{TestCase, Assert}"},
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPHPUses(ctx, []byte(tt.content))
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
