// Package tspool provides tree-sitter parsers for concurrent parsing.
//
// This package centralizes parser management to:
//   - Provide consistent parser creation across components
//   - Ensure thread-safe language initialization
//
// Note: Parser pooling is disabled due to tree-sitter cancellation flag issues.
// When a context is cancelled during ParseCtx, the parser's internal cancel flag
// is set but not properly reset, causing subsequent parses to fail with
// "operation limit was hit". Creating fresh parsers avoids this issue.
//
// Thread-safety: Parsers returned by Get are NOT safe for concurrent use.
// Each goroutine must Get its own parser or use the Parse helper.
package tspool

import (
	"context"
	"fmt"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/specvital/core/pkg/domain"
)

// MaxTreeDepth is the maximum recursion depth when walking AST trees.
const MaxTreeDepth = 1000

var (
	cppLang  *sitter.Language
	csLang   *sitter.Language
	goLang   *sitter.Language
	javaLang *sitter.Language
	jsLang   *sitter.Language
	phpLang  *sitter.Language
	pyLang   *sitter.Language
	rbLang   *sitter.Language
	rsLang   *sitter.Language
	tsLang   *sitter.Language

	langOnce sync.Once
)

func initLanguages() {
	langOnce.Do(func() {
		cppLang = cpp.GetLanguage()
		csLang = csharp.GetLanguage()
		goLang = golang.GetLanguage()
		javaLang = java.GetLanguage()
		jsLang = javascript.GetLanguage()
		phpLang = php.GetLanguage()
		pyLang = python.GetLanguage()
		rbLang = ruby.GetLanguage()
		rsLang = rust.GetLanguage()
		tsLang = typescript.GetLanguage()
	})
}

// GetLanguage returns the tree-sitter language for the given domain language.
func GetLanguage(lang domain.Language) *sitter.Language {
	initLanguages()
	switch lang {
	case domain.LanguageCpp:
		return cppLang
	case domain.LanguageCSharp:
		return csLang
	case domain.LanguageGo:
		return goLang
	case domain.LanguageJava:
		return javaLang
	case domain.LanguageJavaScript:
		return jsLang
	case domain.LanguagePHP:
		return phpLang
	case domain.LanguagePython:
		return pyLang
	case domain.LanguageRuby:
		return rbLang
	case domain.LanguageRust:
		return rsLang
	default:
		return tsLang
	}
}

// Get returns a parser for the given language.
// The returned parser is NOT safe for concurrent use.
// Caller MUST call parser.Close() when done to free resources.
func Get(lang domain.Language) *sitter.Parser {
	initLanguages()
	parser := sitter.NewParser()
	parser.SetLanguage(GetLanguage(lang))
	return parser
}

// Parse parses source using a fresh parser.
// Caller MUST call tree.Close() to free resources.
func Parse(ctx context.Context, lang domain.Language, source []byte) (*sitter.Tree, error) {
	parser := Get(lang)
	defer parser.Close()

	tree, err := parser.ParseCtx(ctx, nil, source)
	if err != nil {
		return nil, fmt.Errorf("parse %s failed: %w", lang, err)
	}

	return tree, nil
}
