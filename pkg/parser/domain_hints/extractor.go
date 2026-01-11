package domain_hints

import (
	"context"

	"github.com/specvital/core/pkg/domain"
)

// Extractor extracts domain classification hints from source code.
type Extractor interface {
	// Extract analyzes source code and returns domain hints.
	// Returns nil if extraction is not supported for the language.
	Extract(ctx context.Context, source []byte) *domain.DomainHints
}

// GetExtractor returns the appropriate extractor for a language.
// Returns nil if no extractor is available.
func GetExtractor(lang domain.Language) Extractor {
	switch lang {
	case domain.LanguageGo:
		return &GoExtractor{}
	case domain.LanguageJavaScript, domain.LanguageTypeScript, domain.LanguageTSX:
		return &JavaScriptExtractor{lang: lang}
	case domain.LanguagePython:
		return &PythonExtractor{}
	case domain.LanguageJava:
		return &JavaExtractor{}
	case domain.LanguageKotlin:
		return &KotlinExtractor{}
	case domain.LanguageCSharp:
		return &CSharpExtractor{}
	default:
		return nil
	}
}
