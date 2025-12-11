package jstest

import (
	"regexp"

	"github.com/specvital/core/pkg/domain"
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
