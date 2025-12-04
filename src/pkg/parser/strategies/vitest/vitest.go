package vitest

import (
	"context"
	"regexp"

	"github.com/specvital/core/src/pkg/domain"
	"github.com/specvital/core/src/pkg/parser/strategies"
	"github.com/specvital/core/src/pkg/parser/strategies/shared/jstest"
)

const (
	frameworkName = "vitest"
	// priorityOffset is added to DefaultPriority to ensure Vitest takes precedence
	// over Jest when a file contains vitest imports, since both frameworks share
	// similar test syntax.
	priorityOffset = 10
)

// vitestImportPattern matches import/require statements for 'vitest'.
// Matches: import ... from 'vitest', import ... from "vitest", require('vitest'), require("vitest")
var vitestImportPattern = regexp.MustCompile(`(?:import\s+.*\s+from|require\()\s*['"]vitest['"]`)

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
	return strategies.DefaultPriority + priorityOffset
}

func (s *Strategy) Languages() []domain.Language {
	return []domain.Language{domain.LanguageTypeScript, domain.LanguageJavaScript}
}

func (s *Strategy) CanHandle(filename string, content []byte) bool {
	if !jstest.IsTestFile(filename) {
		return false
	}

	return hasVitestImport(content)
}

func hasVitestImport(content []byte) bool {
	return vitestImportPattern.Match(content)
}

func (s *Strategy) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	return jstest.Parse(ctx, source, filename, frameworkName)
}
