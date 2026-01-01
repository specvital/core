// Package cargotest implements Rust cargo test framework support.
package cargotest

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser"
	"github.com/specvital/core/pkg/parser/framework"
	"github.com/specvital/core/pkg/parser/framework/matchers"
)

const frameworkName = "cargo-test"

// Tree-sitter node types for Rust
const (
	nodeAttributeItem    = "attribute_item"
	nodeAttribute        = "attribute"
	nodeFunctionItem     = "function_item"
	nodeMacroDefinition  = "macro_definition"
	nodeMacroInvocation  = "macro_invocation"
	nodeMacroRule        = "macro_rule"
	nodeMetaItem         = "meta_item"
	nodeModItem          = "mod_item"
	nodeIdentifier       = "identifier"
	nodeScopedIdentifier = "scoped_identifier"
	nodeTokenTree        = "token_tree"
)

func init() {
	framework.Register(NewDefinition())
}

func NewDefinition() *framework.Definition {
	return &framework.Definition{
		Name:      frameworkName,
		Languages: []domain.Language{domain.LanguageRust},
		Matchers: []framework.Matcher{
			&CargoTestFileMatcher{},
			matchers.NewConfigMatcher("Cargo.toml"),
			&CargoTestContentMatcher{},
		},
		ConfigParser: nil,
		Parser:       &CargoTestParser{},
		Priority:     framework.PriorityGeneric,
	}
}

// CargoTestFileMatcher matches Rust test files.
type CargoTestFileMatcher struct{}

func (m *CargoTestFileMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileName {
		return framework.NoMatch()
	}

	filename := signal.Value

	if strings.HasSuffix(filename, "_test.rs") {
		return framework.PartialMatch(20, "Rust test file naming: *_test.rs")
	}

	if strings.HasSuffix(filename, ".rs") {
		if strings.Contains(filename, "/tests/") || strings.HasPrefix(filename, "tests/") {
			return framework.PartialMatch(20, "Rust test directory: tests/*.rs")
		}
	}

	return framework.NoMatch()
}

// CargoTestContentMatcher matches #[test] and #[cfg(test)] patterns.
type CargoTestContentMatcher struct{}

var cargoTestPatterns = []struct {
	pattern *regexp.Regexp
	desc    string
}{
	{regexp.MustCompile(`#\[test\]`), "#[test] attribute"},
	{regexp.MustCompile(`#\[cfg\(test\)\]`), "#[cfg(test)] attribute"},
	{regexp.MustCompile(`#\[ignore\]`), "#[ignore] attribute"},
	{regexp.MustCompile(`#\[should_panic`), "#[should_panic] attribute"},
	{regexp.MustCompile(`\w*test\w*!\s*\(`), "macro-based test pattern"},
}

func (m *CargoTestContentMatcher) Match(ctx context.Context, signal framework.Signal) framework.MatchResult {
	if signal.Type != framework.SignalFileContent {
		return framework.NoMatch()
	}

	content, ok := signal.Context.([]byte)
	if !ok {
		content = []byte(signal.Value)
	}

	for _, p := range cargoTestPatterns {
		if p.pattern.Match(content) {
			return framework.PartialMatch(40, "Found Rust pattern: "+p.desc)
		}
	}

	return framework.NoMatch()
}

// CargoTestParser extracts test definitions from Rust source files.
type CargoTestParser struct{}

func (p *CargoTestParser) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	tree, err := parser.ParseWithPool(ctx, domain.LanguageRust, source)
	if err != nil {
		return nil, fmt.Errorf("cargo-test parser: failed to parse %s: %w", filename, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	file := &domain.TestFile{
		Path:      filename,
		Language:  domain.LanguageRust,
		Framework: frameworkName,
	}

	// Use WalkTree for depth-protected traversal (prevents stack overflow)
	parseRustAST(root, source, filename, file)
	return file, nil
}

// parseRustAST traverses the AST using depth-protected WalkTree.
// It handles test modules and test functions at the top level and within #[cfg(test)] modules.
// Uses 2-pass approach: first collects test-generating macro definitions, then processes tests.
func parseRustAST(root *sitter.Node, source []byte, filename string, file *domain.TestFile) {
	// Track test modules by node start byte position to associate tests with their parent suite
	testModules := make(map[uint32]*domain.TestSuite)

	// First pass: collect macro definitions that generate #[test] functions
	testMacros := collectTestMacroDefinitions(root, source)

	// Second pass: detect tests using both macro registry and name heuristic
	parser.WalkTree(root, func(node *sitter.Node) bool {
		switch node.Type() {
		case nodeModItem:
			if isTestModule(node, source) {
				name := extractModuleName(node, source)
				if name != "" {
					suite := &domain.TestSuite{
						Name:     name,
						Status:   domain.TestStatusActive,
						Location: parser.GetLocation(node, filename),
					}
					// Store suite keyed by start byte position
					testModules[node.StartByte()] = suite
				}
			}
			return true // Continue into children

		case nodeFunctionItem:
			attrs := collectAttributes(node, source)
			if !attrs.isTest {
				return false // Skip non-test functions
			}

			name := extractFunctionName(node, source)
			if name == "" {
				return false
			}

			test := buildTest(name, attrs, node, filename)

			// Find parent test module, if any
			parentSuite := findParentTestSuite(node, testModules)
			if parentSuite != nil {
				parentSuite.Tests = append(parentSuite.Tests, test)
			} else {
				file.Tests = append(file.Tests, test)
			}
			return false // No need to traverse into function body

		case nodeMacroInvocation:
			macroName, testName := extractMacroTest(node, source)
			if macroName == "" || testName == "" {
				return true // Continue traversal
			}

			if !isTestMacro(macroName, testMacros) {
				return true // Not a test macro, continue
			}

			test := domain.Test{
				Name:     testName,
				Status:   domain.TestStatusActive,
				Modifier: macroName + "!",
				Location: parser.GetLocation(node, filename),
			}

			// Find parent test module, if any
			parentSuite := findParentTestSuite(node, testModules)
			if parentSuite != nil {
				parentSuite.Tests = append(parentSuite.Tests, test)
			} else {
				file.Tests = append(file.Tests, test)
			}
			return false // No need to traverse into macro body
		}

		return true // Continue traversal for other node types
	})

	// Add non-empty test suites to file
	for _, suite := range testModules {
		if len(suite.Tests) > 0 || len(suite.Suites) > 0 {
			file.Suites = append(file.Suites, *suite)
		}
	}
}

// collectTestMacroDefinitions finds macro_rules! definitions that generate #[test] functions.
// Returns a map of macro names that should be treated as test macros.
func collectTestMacroDefinitions(root *sitter.Node, source []byte) map[string]bool {
	testMacros := make(map[string]bool)

	parser.WalkTree(root, func(node *sitter.Node) bool {
		if node.Type() == nodeMacroDefinition {
			name, generatesTest := analyzeMacroDefinition(node, source)
			if name != "" && generatesTest {
				testMacros[name] = true
			}
		}
		return true
	})

	return testMacros
}

// analyzeMacroDefinition checks if a macro_definition generates #[test] functions.
// Returns the macro name and whether it generates tests.
func analyzeMacroDefinition(node *sitter.Node, source []byte) (macroName string, generatesTest bool) {
	if node.Type() != nodeMacroDefinition {
		return "", false
	}

	// Find macro name (identifier after "macro_rules!")
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeIdentifier {
			macroName = parser.GetNodeText(child, source)
			break
		}
	}

	if macroName == "" {
		return "", false
	}

	// Check each macro_rule for #[test] in expansion body
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeMacroRule {
			if macroRuleHasTestAttribute(child, source) {
				return macroName, true
			}
		}
	}

	return macroName, false
}

// macroRuleHasTestAttribute checks if a macro_rule's expansion contains #[test].
func macroRuleHasTestAttribute(rule *sitter.Node, source []byte) bool {
	// macro_rule structure: token_tree_pattern => token_tree
	// We need to find the expansion token_tree (after =>)
	foundArrow := false
	for i := 0; i < int(rule.ChildCount()); i++ {
		child := rule.Child(i)

		// Look for => token
		if parser.GetNodeText(child, source) == "=>" {
			foundArrow = true
			continue
		}

		// After =>, check token_tree for #[test]
		if foundArrow && child.Type() == nodeTokenTree {
			return tokenTreeHasTestAttribute(child, source)
		}
	}
	return false
}

// tokenTreeHasTestAttribute recursively checks if a token_tree contains #[test] pattern.
// Pattern: # followed by token_tree containing identifier "test"
func tokenTreeHasTestAttribute(tokenTree *sitter.Node, source []byte) bool {
	for i := 0; i < int(tokenTree.ChildCount()); i++ {
		child := tokenTree.Child(i)

		// Look for # token
		if parser.GetNodeText(child, source) == "#" {
			// Next sibling should be token_tree with [test]
			if i+1 < int(tokenTree.ChildCount()) {
				nextChild := tokenTree.Child(i + 1)
				if nextChild.Type() == nodeTokenTree {
					// Check for identifier "test" inside
					for j := 0; j < int(nextChild.ChildCount()); j++ {
						inner := nextChild.Child(j)
						if inner.Type() == nodeIdentifier {
							if parser.GetNodeText(inner, source) == "test" {
								return true
							}
						}
					}
				}
			}
		}

		// Recurse into nested token_tree
		if child.Type() == nodeTokenTree {
			if tokenTreeHasTestAttribute(child, source) {
				return true
			}
		}
	}
	return false
}

// findParentTestSuite finds the nearest ancestor test module for a node.
func findParentTestSuite(node *sitter.Node, testModules map[uint32]*domain.TestSuite) *domain.TestSuite {
	current := node.Parent()
	for current != nil {
		if suite, ok := testModules[current.StartByte()]; ok {
			return suite
		}
		current = current.Parent()
	}
	return nil
}

// buildTest creates a Test from function attributes.
func buildTest(name string, attrs testAttributes, node *sitter.Node, filename string) domain.Test {
	status := domain.TestStatusActive
	modifier := ""

	if attrs.isIgnore {
		status = domain.TestStatusSkipped
		modifier = "#[ignore]"
	}

	if attrs.shouldPanic != "" {
		if modifier != "" {
			modifier += " " + attrs.shouldPanic
		} else {
			modifier = attrs.shouldPanic
		}
	}

	return domain.Test{
		Name:     name,
		Status:   status,
		Modifier: modifier,
		Location: parser.GetLocation(node, filename),
	}
}

type testAttributes struct {
	isTest      bool
	isIgnore    bool
	shouldPanic string // Full attribute text (e.g., "#[should_panic(expected = \"...\")]")
}

// getPrecedingAttributes returns attribute_item nodes immediately preceding the given node.
func getPrecedingAttributes(node *sitter.Node) []*sitter.Node {
	parent := node.Parent()
	if parent == nil {
		return nil
	}

	nodeIndex := -1
	for i := 0; i < int(parent.ChildCount()); i++ {
		if parent.Child(i) == node {
			nodeIndex = i
			break
		}
	}

	if nodeIndex == -1 {
		return nil
	}

	var attrs []*sitter.Node
	for i := nodeIndex - 1; i >= 0; i-- {
		child := parent.Child(i)
		if child.Type() != nodeAttributeItem {
			break
		}
		attrs = append(attrs, child)
	}

	return attrs
}

func collectAttributes(funcNode *sitter.Node, source []byte) testAttributes {
	attrs := testAttributes{}

	for _, attrNode := range getPrecedingAttributes(funcNode) {
		attrName := extractAttributeName(attrNode, source)
		switch attrName {
		case "test":
			attrs.isTest = true
		case "ignore":
			attrs.isIgnore = true
		case "should_panic":
			attrs.shouldPanic = parser.GetNodeText(attrNode, source)
		}
	}

	return attrs
}

func extractAttributeName(attrItem *sitter.Node, source []byte) string {
	attr := parser.FindChildByType(attrItem, nodeAttribute)
	if attr == nil {
		return ""
	}

	ident := parser.FindChildByType(attr, nodeIdentifier)
	if ident != nil {
		return parser.GetNodeText(ident, source)
	}

	// Handle complex attributes like #[cfg(test)] where identifier is nested in meta_item
	meta := parser.FindChildByType(attr, nodeMetaItem)
	if meta != nil {
		ident = parser.FindChildByType(meta, nodeIdentifier)
		if ident != nil {
			return parser.GetNodeText(ident, source)
		}
	}

	return ""
}

func extractFunctionName(funcNode *sitter.Node, source []byte) string {
	name := funcNode.ChildByFieldName("name")
	if name == nil {
		return ""
	}
	return parser.GetNodeText(name, source)
}

func extractModuleName(modNode *sitter.Node, source []byte) string {
	name := modNode.ChildByFieldName("name")
	if name == nil {
		return ""
	}
	return parser.GetNodeText(name, source)
}

func isTestModule(modNode *sitter.Node, source []byte) bool {
	for _, attrNode := range getPrecedingAttributes(modNode) {
		attrName := extractAttributeName(attrNode, source)
		if attrName == "cfg" {
			attrText := parser.GetNodeText(attrNode, source)
			if strings.Contains(attrText, "cfg(test)") {
				return true
			}
		}
	}

	name := extractModuleName(modNode, source)
	return name == "tests"
}

// extractMacroTest extracts macro name and test name from a macro invocation.
// For patterns like `rgtest!(test_name, ...)`, returns ("rgtest", "test_name").
func extractMacroTest(node *sitter.Node, source []byte) (macroName, testName string) {
	if node.Type() != nodeMacroInvocation {
		return "", ""
	}

	// Extract macro name from the "macro" field or first child
	macroField := node.ChildByFieldName("macro")
	if macroField != nil {
		macroName = extractMacroName(macroField, source)
	} else if node.ChildCount() > 0 {
		firstChild := node.Child(0)
		if firstChild != nil {
			macroName = extractMacroName(firstChild, source)
		}
	}

	if macroName == "" {
		return "", ""
	}

	// Extract test name from first identifier in token_tree
	tokenTree := parser.FindChildByType(node, nodeTokenTree)
	if tokenTree == nil {
		return macroName, ""
	}

	testNameIdent := parser.FindChildByType(tokenTree, nodeIdentifier)
	if testNameIdent != nil {
		testName = parser.GetNodeText(testNameIdent, source)
	}

	return macroName, testName
}

// extractMacroName handles both simple identifiers and scoped identifiers.
// For `crate::macros::rgtest`, returns "rgtest" (last segment).
func extractMacroName(node *sitter.Node, source []byte) string {
	switch node.Type() {
	case nodeIdentifier:
		return parser.GetNodeText(node, source)
	case nodeScopedIdentifier:
		// For scoped identifiers, get the last segment (name field)
		name := node.ChildByFieldName("name")
		if name != nil {
			return parser.GetNodeText(name, source)
		}
		// Fallback: get last child identifier
		for i := int(node.ChildCount()) - 1; i >= 0; i-- {
			child := node.Child(i)
			if child.Type() == nodeIdentifier {
				return parser.GetNodeText(child, source)
			}
		}
	}
	return ""
}

// isTestMacro checks if a macro name indicates a test macro.
// First checks the local registry (macros defined in the same file with #[test]),
// then falls back to name-based heuristic (names containing "test").
func isTestMacro(macroName string, localRegistry map[string]bool) bool {
	// Check local registry first (macros defined in same file)
	if localRegistry[macroName] {
		return true
	}
	// Fallback to name-based heuristic for external macros
	return strings.Contains(strings.ToLower(macroName), "test")
}
