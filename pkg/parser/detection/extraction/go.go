package extraction

import (
	"context"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

var (
	goLang     *sitter.Language
	goLangOnce sync.Once
)

func getGoLanguage() *sitter.Language {
	goLangOnce.Do(func() {
		goLang = golang.GetLanguage()
	})
	return goLang
}

// Go import query: captures import path from both single and grouped imports.
// Matches import_spec directly to handle both:
// - Single: import "testing"
// - Grouped: import ( "fmt" \n "testing" )
const goImportQuery = `(import_spec path: (interpreted_string_literal) @import)`

// ExtractGoImports extracts import paths from Go source using tree-sitter.
// Handles both single imports and grouped import blocks.
func ExtractGoImports(ctx context.Context, content []byte) []string {
	lang := getGoLanguage()

	parser := sitter.NewParser()
	parser.SetLanguage(lang)
	defer parser.Close()

	tree, err := parser.ParseCtx(ctx, nil, content)
	if err != nil {
		return nil
	}
	defer tree.Close()

	query, err := sitter.NewQuery([]byte(goImportQuery), lang)
	if err != nil {
		return nil
	}
	defer query.Close()

	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	cursor.Exec(query, tree.RootNode())

	var imports []string
	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}

		for _, capture := range match.Captures {
			name := query.CaptureNameForId(capture.Index)
			if name == "import" {
				// Remove quotes from import path
				path := capture.Node.Content(content)
				if len(path) >= 2 {
					path = path[1 : len(path)-1] // Strip quotes
				}
				imports = append(imports, path)
			}
		}
	}

	return imports
}
