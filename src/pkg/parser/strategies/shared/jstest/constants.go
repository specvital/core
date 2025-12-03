package jstest

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/specvital/core/domain"
)

const (
	FuncDescribe = "describe"
	FuncIt       = "it"
	FuncTest     = "test"

	ModifierSkip = "skip"
	ModifierOnly = "only"
	ModifierEach = "each"
	ModifierTodo = "todo"

	DynamicCasesSuffix = " (dynamic cases)"
)

var SkippedFunctionAliases = map[string]string{
	"xdescribe": FuncDescribe,
	"xit":       FuncIt,
	"xtest":     FuncTest,
}

var FocusedFunctionAliases = map[string]string{
	"fdescribe": FuncDescribe,
	"fit":       FuncIt,
}

var JestPlaceholderPattern = regexp.MustCompile(`%[sdpji#%]`)

var SupportedExtensions = map[string]bool{
	".ts":  true,
	".tsx": true,
	".js":  true,
	".jsx": true,
}

var testFilePattern = regexp.MustCompile(`\.(test|spec)\.(ts|tsx|js|jsx)$`)

const testsDir = "__tests__"

func IsTestFile(filename string) bool {
	if testFilePattern.MatchString(filename) {
		return true
	}

	if isInTestsDirectory(filename) {
		return hasSupportedExtension(filename)
	}

	return false
}

func isInTestsDirectory(filename string) bool {
	normalizedPath := filepath.ToSlash(filename)
	parts := strings.Split(normalizedPath, "/")
	return slices.Contains(parts, testsDir)
}

func hasSupportedExtension(filename string) bool {
	ext := filepath.Ext(filename)
	return SupportedExtensions[ext]
}

func ParseModifierStatus(modifier string) domain.TestStatus {
	switch modifier {
	case ModifierSkip, ModifierTodo:
		return domain.TestStatusSkipped
	case ModifierOnly:
		return domain.TestStatusOnly
	default:
		return domain.TestStatusPending
	}
}
