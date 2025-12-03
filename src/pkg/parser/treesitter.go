package parser

import (
	"context"
	"fmt"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/specvital/core/domain"
)

const MaxTreeDepth = 1000

type TSParser struct {
	parser *sitter.Parser
	lang   domain.Language
}

var (
	tsLang *sitter.Language
	jsLang *sitter.Language

	langOnce sync.Once
)

func initLanguages() {
	langOnce.Do(func() {
		tsLang = typescript.GetLanguage()
		jsLang = javascript.GetLanguage()
	})
}

func NewTSParser(lang domain.Language) *TSParser {
	initLanguages()

	parser := sitter.NewParser()
	if parser == nil {
		return nil
	}

	switch lang {
	case domain.LanguageTypeScript:
		parser.SetLanguage(tsLang)
	case domain.LanguageJavaScript:
		parser.SetLanguage(jsLang)
	default:
		parser.SetLanguage(tsLang)
	}

	return &TSParser{
		parser: parser,
		lang:   lang,
	}
}

func (p *TSParser) Parse(source []byte) (*sitter.Node, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, source)
	if err != nil {
		return nil, err
	}
	return tree.RootNode(), nil
}

func (p *TSParser) Close() {}

type QueryResult struct {
	Node     *sitter.Node
	Captures map[string]*sitter.Node
}

func Query(root *sitter.Node, source []byte, lang domain.Language, queryStr string) ([]QueryResult, error) {
	initLanguages()

	sitterLang := getSitterLanguage(lang)

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

func GetNodeText(node *sitter.Node, source []byte) string {
	return node.Content(source)
}

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

func FindChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			return child
		}
	}
	return nil
}

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

func WalkTree(node *sitter.Node, visitor func(*sitter.Node) bool) {
	walkTreeWithDepth(node, visitor, 0)
}

func walkTreeWithDepth(node *sitter.Node, visitor func(*sitter.Node) bool, depth int) {
	if depth > MaxTreeDepth {
		return
	}

	if !visitor(node) {
		return
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		walkTreeWithDepth(node.Child(i), visitor, depth+1)
	}
}

func getSitterLanguage(lang domain.Language) *sitter.Language {
	switch lang {
	case domain.LanguageJavaScript:
		return jsLang
	default:
		return tsLang
	}
}
