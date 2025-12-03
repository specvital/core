package jest

import (
	"context"

	"github.com/specvital/core/domain"
	"github.com/specvital/core/parser/strategies"
	"github.com/specvital/core/parser/strategies/shared/jstest"
)

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
	return jstest.IsTestFile(filename)
}

func (s *Strategy) Parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	return parse(ctx, source, filename)
}
