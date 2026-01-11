package tspool

import (
	"fmt"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
)

// QueryResult contains the result of a tree-sitter query match.
type QueryResult struct {
	// Node is the first captured node in this match.
	Node *sitter.Node
	// Captures maps capture names to their corresponding nodes.
	Captures map[string]*sitter.Node
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
		sitterLang := GetLanguage(lang)
		cached.query, cached.err = sitter.NewQuery([]byte(queryStr), sitterLang)
	})

	return cached.query, cached.err
}

// QueryWithCache executes a tree-sitter query with cached compilation.
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
