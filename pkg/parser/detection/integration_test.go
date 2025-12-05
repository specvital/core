package detection_test

import (
	"os"
	"testing"

	"github.com/specvital/core/pkg/parser/detection/matchers"
	"github.com/specvital/core/pkg/parser/strategies"
	"github.com/specvital/core/pkg/parser/strategies/gotesting"
	"github.com/specvital/core/pkg/parser/strategies/jest"
	"github.com/specvital/core/pkg/parser/strategies/playwright"
	"github.com/specvital/core/pkg/parser/strategies/vitest"
)

func TestMain(m *testing.M) {
	strategies.DefaultRegistry().Clear()
	jest.RegisterDefault()
	vitest.RegisterDefault()
	playwright.RegisterDefault()
	gotesting.RegisterDefault()
	os.Exit(m.Run())
}

func TestMatcherStrategyNameConsistency(t *testing.T) {
	t.Parallel()

	matcherRegistry := matchers.DefaultRegistry()
	strategyRegistry := strategies.DefaultRegistry()

	for _, matcher := range matcherRegistry.All() {
		name := matcher.Name()
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			strategy := strategyRegistry.FindByName(name)
			if strategy == nil {
				t.Errorf("Matcher %q has no corresponding Strategy", name)
				return
			}
			if strategy.Name() != name {
				t.Errorf("Strategy name mismatch: Matcher=%q, Strategy=%q", name, strategy.Name())
			}
		})
	}
}

func TestAllStrategiesHaveMatchers(t *testing.T) {
	t.Parallel()

	matcherRegistry := matchers.DefaultRegistry()
	strategyRegistry := strategies.DefaultRegistry()

	for _, strategy := range strategyRegistry.GetStrategies() {
		name := strategy.Name()
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			matcher, found := matcherRegistry.FindByName(name)
			if !found {
				t.Errorf("Strategy %q has no corresponding Matcher", name)
				return
			}
			if matcher.Name() != name {
				t.Errorf("Matcher name mismatch: Strategy=%q, Matcher=%q", name, matcher.Name())
			}
		})
	}
}
