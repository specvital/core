package parser

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/tspool"
)

// ParseWithPool parses source using a pooled parser.
// Caller must close the returned tree.
func ParseWithPool(ctx context.Context, lang domain.Language, source []byte) (*sitter.Tree, error) {
	return tspool.Parse(ctx, lang, source)
}

// QueryWithCache executes a query with cached compilation.
func QueryWithCache(root *sitter.Node, source []byte, lang domain.Language, queryStr string) ([]QueryResult, error) {
	results, err := tspool.QueryWithCache(root, source, lang, queryStr)
	if err != nil {
		return nil, err
	}

	// Convert tspool.QueryResult to parser.QueryResult
	converted := make([]QueryResult, len(results))
	for i, r := range results {
		converted[i] = QueryResult{
			Node:     r.Node,
			Captures: r.Captures,
		}
	}
	return converted, nil
}

// ClearQueryCache removes all cached queries. Only for testing.
func ClearQueryCache() {
	tspool.ClearQueryCache()
}
