package jest

import (
	"context"

	"github.com/specvital/core/src/pkg/domain"
	"github.com/specvital/core/src/pkg/parser/strategies/shared/jstest"
)

const frameworkName = "jest"

func parse(ctx context.Context, source []byte, filename string) (*domain.TestFile, error) {
	return jstest.Parse(ctx, source, filename, frameworkName)
}
