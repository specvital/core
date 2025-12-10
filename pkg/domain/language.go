// Package domain defines the core types for test file representation.
package domain

// Language represents a programming language.
type Language string

// Supported languages for test file parsing.
const (
	LanguageCpp        Language = "cpp"
	LanguageCSharp     Language = "csharp"
	LanguageGo         Language = "go"
	LanguageJava       Language = "java"
	LanguageJavaScript Language = "javascript"
	LanguagePHP        Language = "php"
	LanguagePython     Language = "python"
	LanguageRuby       Language = "ruby"
	LanguageRust       Language = "rust"
	LanguageTypeScript Language = "typescript"
)
