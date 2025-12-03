package jest

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/specvital/core/domain"
	"github.com/specvital/core/parser/strategies"
)

var supportedExtensions = map[string]bool{
	".ts":  true,
	".tsx": true,
	".js":  true,
	".jsx": true,
}

const testsDir = "__tests__"

var jestFilePattern = regexp.MustCompile(`\.(test|spec)\.(ts|tsx|js|jsx)$`)

type Strategy struct{}

func NewStrategy() *Strategy {
	return &Strategy{}
}

func RegisterDefault() {
	strategies.Register(NewStrategy())
}

func (s *Strategy) Name() string {
	return frameworkName
}

func (s *Strategy) Priority() int {
	return strategies.DefaultPriority
}

func (s *Strategy) Languages() []domain.Language {
	return []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript}
}

func (s *Strategy) CanHandle(filename string, _ []byte) bool {
	if jestFilePattern.MatchString(filename) {
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
	return supportedExtensions[ext]
}

func (s *Strategy) Parse(source []byte, filename string) (*domain.TestFile, error) {
	return parse(source, filename)
}
