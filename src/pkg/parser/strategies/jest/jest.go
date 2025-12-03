// Package jest provides Jest test framework parsing strategy.
package jest

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/specvital/core/domain"
	"github.com/specvital/core/parser/strategies"
)

// Strategy parses Jest test files.
type Strategy struct{}

// Supported file extensions for Jest tests.
var supportedExtensions = map[string]bool{
	".ts":  true,
	".tsx": true,
	".js":  true,
	".jsx": true,
}

var (
	jestFilePattern = regexp.MustCompile(`\.(test|spec)\.(ts|tsx|js|jsx)$`)
	testsDir        = "__tests__"
)

// NewStrategy creates a new Jest strategy instance.
func NewStrategy() *Strategy {
	return &Strategy{}
}

// RegisterDefault registers Jest strategy to the default registry.
func RegisterDefault() {
	strategies.Register(NewStrategy())
}

// Name returns the framework name.
func (s *Strategy) Name() string {
	return frameworkName
}

// Priority returns the strategy priority.
func (s *Strategy) Priority() int {
	return strategies.DefaultPriority
}

// Languages returns supported languages.
func (s *Strategy) Languages() []domain.Language {
	return []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript}
}

// CanHandle checks if this strategy can parse the given file.
func (s *Strategy) CanHandle(filename string, _ []byte) bool {
	if jestFilePattern.MatchString(filename) {
		return true
	}

	if isInTestsDirectory(filename) {
		return hasSupportedExtension(filename)
	}

	return false
}

// isInTestsDirectory checks if the file is in a __tests__ directory.
func isInTestsDirectory(filename string) bool {
	normalizedPath := filepath.ToSlash(filename)
	return strings.Contains(normalizedPath, testsDir+"/")
}

// hasSupportedExtension checks if the file has a supported extension.
func hasSupportedExtension(filename string) bool {
	ext := filepath.Ext(filename)
	return supportedExtensions[ext]
}

// Parse extracts test information from source code.
func (s *Strategy) Parse(source []byte, filename string) (*domain.TestFile, error) {
	return parse(source, filename)
}
