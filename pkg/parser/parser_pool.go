package parser

import (
	"context"
	"fmt"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/tspool"
)

// ParseWithPool parses source using a pooled parser.
// Caller must close the returned tree.
// Deprecated: Use tspool.Parse instead.
func ParseWithPool(ctx context.Context, lang domain.Language, source []byte) (*sitter.Tree, error) {
	return tspool.Parse(ctx, lang, source)
}

type queryCacheKey struct {
	lang     domain.Language
	queryStr string
}

type cachedQuery struct {
	once  sync.Once
	query *sitter.Query
	err   error
}

var queryCache sync.Map

// getCachedQuery returns a compiled query. The returned query must NOT be closed.
func getCachedQuery(lang domain.Language, queryStr string) (*sitter.Query, error) {
	key := queryCacheKey{
		lang:     lang,
		queryStr: queryStr,
	}

	if val, ok := queryCache.Load(key); ok {
		cached, ok := val.(*cachedQuery)
		if !ok {
			return nil, fmt.Errorf("invalid cache entry type")
		}
		cached.once.Do(func() {})
		return cached.query, cached.err
	}

	cached := &cachedQuery{}
	actual, loaded := queryCache.LoadOrStore(key, cached)

	if loaded {
		var ok bool
		cached, ok = actual.(*cachedQuery)
		if !ok {
			return nil, fmt.Errorf("invalid cache entry type")
		}
	}

	cached.once.Do(func() {
		sitterLang := tspool.GetLanguage(lang)
		cached.query, cached.err = sitter.NewQuery([]byte(queryStr), sitterLang)
	})

	return cached.query, cached.err
}

// QueryWithCache executes a query with cached compilation.
func QueryWithCache(root *sitter.Node, source []byte, lang domain.Language, queryStr string) ([]QueryResult, error) {
	query, err := getCachedQuery(lang, queryStr)
	if err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

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

// ClearQueryCache removes all cached queries. Only for testing.
func ClearQueryCache() {
	var toClose []*sitter.Query

	queryCache.Range(func(key, value any) bool {
		queryCache.Delete(key)
		if cached, ok := value.(*cachedQuery); ok {
			cached.once.Do(func() {})
			if cached.query != nil && cached.err == nil {
				toClose = append(toClose, cached.query)
			}
		}
		return true
	})

	for _, q := range toClose {
		q.Close()
	}
}
