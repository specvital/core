package framework

// Priority constants determine the order in which frameworks are checked during detection.
// Higher priority frameworks are evaluated first.
//
// Use increments of 50 to allow for future insertions between priority levels.
const (
	// PriorityGeneric is for common, general-purpose test frameworks.
	// These frameworks typically support a wide range of test files and patterns.
	// Examples: Jest, Go testing
	PriorityGeneric = 100

	// PriorityE2E is for end-to-end testing frameworks.
	// These frameworks are more specialized and should be checked before generic frameworks.
	// Examples: Playwright, Cypress
	PriorityE2E = 150

	// PrioritySpecialized is for highly specialized frameworks with unique characteristics.
	// These should be checked first to avoid false matches with generic frameworks.
	// Examples: Vitest with globals mode (requires explicit import detection)
	PrioritySpecialized = 200
)

// Common framework names as constants to ensure consistency.
const (
	FrameworkCargoTest  = "cargo-test"
	FrameworkCypress    = "cypress"
	FrameworkGoTesting  = "go-testing"
	FrameworkGTest      = "gtest"
	FrameworkJest       = "jest"
	FrameworkJUnit4     = "junit4"
	FrameworkJUnit5     = "junit5"
	FrameworkKotest     = "kotest"
	FrameworkMinitest   = "minitest"
	FrameworkMocha      = "mocha"
	FrameworkMSTest     = "mstest"
	FrameworkNUnit      = "nunit"
	FrameworkPHPUnit    = "phpunit"
	FrameworkPlaywright = "playwright"
	FrameworkPytest     = "pytest"
	FrameworkRSpec        = "rspec"
	FrameworkSwiftTesting = "swift-testing"
	FrameworkTestNG       = "testng"
	FrameworkUnittest     = "unittest"
	FrameworkVitest       = "vitest"
	FrameworkXCTest       = "xctest"
	FrameworkXUnit        = "xunit"
)
