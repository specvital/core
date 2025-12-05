package parser

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/tspool"
)

// MaxTreeDepth is the maximum recursion depth when walking AST trees.
// Deprecated: Use tspool.MaxTreeDepth instead.
const MaxTreeDepth = tspool.MaxTreeDepth

// TSParser wraps a tree-sitter parser for a specific language.
type TSParser struct {
	parser *sitter.Parser
	lang   domain.Language
}

// QueryResult contains the result of a tree-sitter query match.
type QueryResult struct {
	// Node is the first captured node in this match.
	Node *sitter.Node
	// Captures maps capture names to their corresponding nodes.
	Captures map[string]*sitter.Node
}

// NewTSParser creates a new tree-sitter parser for the given language.
func NewTSParser(lang domain.Language) *TSParser {
	parser := sitter.NewParser()
	parser.SetLanguage(tspool.GetLanguage(lang))

	return &TSParser{
		parser: parser,
		lang:   lang,
	}
}

// Parse parses the source code and returns the AST tree.
func (p *TSParser) Parse(ctx context.Context, source []byte) (*sitter.Tree, error) {
	return p.parser.ParseCtx(ctx, nil, source)
}

// Query executes a tree-sitter query and returns all matches.
// The query is compiled fresh each time; for repeated queries, use [QueryWithCache].
func Query(root *sitter.Node, source []byte, lang domain.Language, queryStr string) ([]QueryResult, error) {
	sitterLang := tspool.GetLanguage(lang)

	query, err := sitter.NewQuery([]byte(queryStr), sitterLang)
	if err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}
	defer query.Close()

	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	cursor.Exec(query, root)

	var results []QueryResult
	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}

		result := QueryResult{
			Captures: make(map[string]*sitter.Node),
		}

		for _, capture := range match.Captures {
			name := query.CaptureNameForId(capture.Index)
			result.Captures[name] = capture.Node
			if result.Node == nil {
				result.Node = capture.Node
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// GetNodeText returns the source text for the given AST node.
func GetNodeText(node *sitter.Node, source []byte) string {
	return node.Content(source)
}

// GetLocation converts a tree-sitter node position to a [domain.Location].
// Line numbers are converted to 1-based indexing.
func GetLocation(node *sitter.Node, filename string) domain.Location {
	start := node.StartPoint()
	end := node.EndPoint()

	return domain.Location{
		File:      filename,
		StartLine: int(start.Row) + 1, // Convert to 1-based
		EndLine:   int(end.Row) + 1,
		StartCol:  int(start.Column),
		EndCol:    int(end.Column),
	}
}

// FindChildByType returns the first direct child with the given node type.
func FindChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			return child
		}
	}
	return nil
}

// FindChildrenByType returns all direct children with the given node type.
func FindChildrenByType(node *sitter.Node, nodeType string) []*sitter.Node {
	var children []*sitter.Node
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			children = append(children, child)
		}
	}
	return children
}

func walkTreeWithDepth(node *sitter.Node, visitor func(*sitter.Node) bool, depth int) {
	if depth > tspool.MaxTreeDepth {
		return
	}

	if !visitor(node) {
		return
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		walkTreeWithDepth(node.Child(i), visitor, depth+1)
	}
}

// WalkTree recursively visits all nodes in the AST.
// The visitor function returns false to stop traversing into children.
func WalkTree(node *sitter.Node, visitor func(*sitter.Node) bool) {
	walkTreeWithDepth(node, visitor, 0)
}
