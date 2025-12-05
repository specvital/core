package extraction

import (
	"bytes"
	"context"
	"regexp"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/tspool"
)

var commentStripRegex = regexp.MustCompile(`//.*|/\*[\s\S]*?\*/`)

var jsImportPattern = regexp.MustCompile(`(?:import\s+.*?\s+from|require\()\s*['"]([^'"]+)['"]`)

func ExtractJSImports(_ context.Context, content []byte) []string {
	matches := jsImportPattern.FindAllSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	imports := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, string(match[1]))
		}
	}
	return imports
}

// MatchPatternExcludingComments checks if pattern matches content after stripping comments.
// Uses tree-sitter TypeScript parser (handles JS too) to accurately identify comment nodes,
// avoiding false positives from comment-like patterns inside string literals (e.g., "**/*.ts").
// Falls back to regex-based stripping if tree-sitter parsing fails.
func MatchPatternExcludingComments(ctx context.Context, content []byte, pattern *regexp.Regexp) bool {
	// Fast path: no comment markers, use direct match
	if !bytes.Contains(content, []byte("//")) && !bytes.Contains(content, []byte("/*")) {
		return pattern.Match(content)
	}

	// Slow path: use tree-sitter for accurate comment detection
	tree, err := tspool.Parse(ctx, domain.LanguageTypeScript, content)
	if err != nil {
		// Fallback to regex-based stripping for malformed content
		noComments := commentStripRegex.ReplaceAll(content, []byte{})
		return pattern.Match(noComments)
	}
	defer tree.Close()

	cleaned := removeCommentNodes(tree.RootNode(), content)
	return pattern.Match(cleaned)
}

// removeCommentNodes removes all comment nodes from content using tree-sitter AST.
func removeCommentNodes(root *sitter.Node, content []byte) []byte {
	var commentRanges [][2]uint32
	collectCommentRanges(root, &commentRanges)

	if len(commentRanges) == 0 {
		return content
	}

	result := make([]byte, 0, len(content))
	lastEnd := uint32(0)

	for _, commentRange := range commentRanges {
		// Skip overlapping ranges (should not happen with tree-sitter, but defensive)
		if commentRange[0] < lastEnd {
			continue
		}
		if commentRange[0] > lastEnd {
			result = append(result, content[lastEnd:commentRange[0]]...)
		}
		lastEnd = commentRange[1]
	}

	if lastEnd < uint32(len(content)) {
		result = append(result, content[lastEnd:]...)
	}

	return result
}

// collectCommentRanges collects byte ranges of all comment nodes in DFS order.
func collectCommentRanges(node *sitter.Node, ranges *[][2]uint32) {
	collectCommentRangesWithDepth(node, ranges, 0)
}

func collectCommentRangesWithDepth(node *sitter.Node, ranges *[][2]uint32, depth int) {
	if depth > tspool.MaxTreeDepth {
		return
	}

	if node.Type() == "comment" {
		*ranges = append(*ranges, [2]uint32{node.StartByte(), node.EndByte()})
		return
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		collectCommentRangesWithDepth(node.Child(i), ranges, depth+1)
	}
}
