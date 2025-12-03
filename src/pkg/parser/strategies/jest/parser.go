package jest

import (
	"fmt"
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/domain"
	"github.com/specvital/core/parser"
)

// parse is the main entry point for parsing Jest test files.
func parse(source []byte, filename string) (*domain.TestFile, error) {
	lang := detectLanguage(filename)

	p := parser.NewTSParser(lang)
	if p == nil {
		return nil, fmt.Errorf("failed to create parser for language %s", lang)
	}
	defer p.Close()

	root, err := p.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	testFile := &domain.TestFile{
		Path:      filename,
		Language:  lang,
		Framework: frameworkName,
	}

	parseJestNode(root, source, filename, testFile, nil)

	return testFile, nil
}

// detectLanguage determines the language from file extension.
func detectLanguage(filename string) domain.Language {
	ext := filepath.Ext(filename)
	switch ext {
	case ".js", ".jsx":
		return domain.LanguageJavaScript
	default:
		return domain.LanguageTypeScript
	}
}

// parseJestNode recursively parses Jest test constructs.
func parseJestNode(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)

		switch child.Type() {
		case "expression_statement":
			if expr := parser.FindChildByType(child, "call_expression"); expr != nil {
				processCallExpression(expr, source, filename, file, currentSuite)
			}
		default:
			parseJestNode(child, source, filename, file, currentSuite)
		}
	}
}

// processCallExpression handles describe, it, test calls.
func processCallExpression(node *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return
	}

	args := node.ChildByFieldName("arguments")
	if args == nil {
		return
	}

	if funcNode.Type() == "call_expression" {
		processEachCall(node, funcNode, args, source, filename, file, currentSuite)
		return
	}

	funcName, status := parseFunctionName(funcNode, source)
	if funcName == "" {
		return
	}

	switch funcName {
	case funcDescribe:
		processSuite(node, args, source, filename, file, currentSuite, status)
	case funcIt, funcTest:
		processTest(node, args, source, filename, file, currentSuite, status)
	}
}
