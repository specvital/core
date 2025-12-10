package extraction

import (
	"context"
	"regexp"
)

// PHP use patterns:
// - use PHPUnit\Framework\TestCase;
// - use Illuminate\Tests\Mail;

var phpUsePattern = regexp.MustCompile(`(?m)^use\s+([^;]+);`)

// ExtractPHPUses extracts namespace names from PHP use statements.
func ExtractPHPUses(_ context.Context, content []byte) []string {
	matches := phpUsePattern.FindAllSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	uses := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) < 2 || len(match[1]) == 0 {
			continue
		}

		ns := string(match[1])

		if _, ok := seen[ns]; ok {
			continue
		}

		seen[ns] = struct{}{}
		uses = append(uses, ns)
	}

	return uses
}
