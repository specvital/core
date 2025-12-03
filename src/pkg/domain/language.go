// Package domain defines core domain types for test parsing.
package domain

// Language represents a programming language.
type Language string

const (
	LanguageTypeScript Language = "typescript"
	LanguageJavaScript Language = "javascript"
	LanguageGo         Language = "go"
)
