// Package all imports all parser strategies for side-effect registration.
// Usage: _ "github.com/specvital/core/pkg/parser/strategies/all"
package all

import (
	_ "github.com/specvital/core/pkg/parser/strategies/gotesting"
	_ "github.com/specvital/core/pkg/parser/strategies/jest"
	_ "github.com/specvital/core/pkg/parser/strategies/junit5"
	_ "github.com/specvital/core/pkg/parser/strategies/playwright"
	_ "github.com/specvital/core/pkg/parser/strategies/pytest"
	_ "github.com/specvital/core/pkg/parser/strategies/unittest"
	_ "github.com/specvital/core/pkg/parser/strategies/vitest"
)
