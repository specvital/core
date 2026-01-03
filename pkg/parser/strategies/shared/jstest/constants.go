package jstest

import (
	"regexp"

	"github.com/specvital/core/pkg/domain"
)

const (
	FuncBench    = "bench"
	FuncDescribe = "describe"
	FuncIt       = "it"
	FuncTest     = "test"

	// Mocha TDD interface functions
	FuncContext = "context"
	FuncSpecify = "specify"
	FuncSuite   = "suite"

	// jscodeshift test utility
	FuncDefineTest = "defineTest"

	// ESLint RuleTester method
	MethodRun = "run"

	ModifierConcurrent = "concurrent"
	ModifierEach       = "each"
	ModifierFor        = "for"
	ModifierOnly       = "only"
	ModifierSkip       = "skip"
	ModifierTodo       = "todo"

	DynamicCasesSuffix     = " (dynamic cases)"
	DynamicNamePlaceholder = "(dynamic)"
	ObjectPlaceholder      = "<object>"
)

var SkippedFunctionAliases = map[string]string{
	"xdescribe": FuncDescribe,
	"xit":       FuncIt,
	"xtest":     FuncTest,
	"xcontext":  FuncContext,
	"xspecify":  FuncSpecify,
}

var FocusedFunctionAliases = map[string]string{
	"fdescribe": FuncDescribe,
	"fit":       FuncIt,
	"fcontext":  FuncContext,
	"fspecify":  FuncSpecify,
}

var JestPlaceholderPattern = regexp.MustCompile(`%[sdpji#%]`)

// SupportedExtensions defines valid JavaScript/TypeScript file extensions.
var SupportedExtensions = map[string]bool{
	".ts":  true,
	".tsx": true,
	".js":  true,
	".jsx": true,
}

func ParseModifierStatus(modifier string) domain.TestStatus {
	switch modifier {
	case ModifierSkip:
		return domain.TestStatusSkipped
	case ModifierTodo:
		return domain.TestStatusTodo
	case ModifierOnly:
		return domain.TestStatusFocused
	default:
		return domain.TestStatusActive
	}
}
