// Package all imports all parser strategies for side-effect registration.
// Usage: _ "github.com/specvital/core/pkg/parser/strategies/all"
package all

import (
	_ "github.com/specvital/core/pkg/parser/strategies/cargotest"
	_ "github.com/specvital/core/pkg/parser/strategies/cypress"
	_ "github.com/specvital/core/pkg/parser/strategies/gotesting"
	_ "github.com/specvital/core/pkg/parser/strategies/gtest"
	_ "github.com/specvital/core/pkg/parser/strategies/jest"
	_ "github.com/specvital/core/pkg/parser/strategies/junit4"
	_ "github.com/specvital/core/pkg/parser/strategies/junit5"
	_ "github.com/specvital/core/pkg/parser/strategies/kotest"
	_ "github.com/specvital/core/pkg/parser/strategies/minitest"
	_ "github.com/specvital/core/pkg/parser/strategies/mocha"
	_ "github.com/specvital/core/pkg/parser/strategies/mstest"
	_ "github.com/specvital/core/pkg/parser/strategies/nunit"
	_ "github.com/specvital/core/pkg/parser/strategies/phpunit"
	_ "github.com/specvital/core/pkg/parser/strategies/playwright"
	_ "github.com/specvital/core/pkg/parser/strategies/pytest"
	_ "github.com/specvital/core/pkg/parser/strategies/rspec"
	_ "github.com/specvital/core/pkg/parser/strategies/testng"
	_ "github.com/specvital/core/pkg/parser/strategies/unittest"
	_ "github.com/specvital/core/pkg/parser/strategies/vitest"
	_ "github.com/specvital/core/pkg/parser/strategies/xctest"
	_ "github.com/specvital/core/pkg/parser/strategies/xunit"
)
