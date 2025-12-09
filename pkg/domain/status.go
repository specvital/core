package domain

// TestStatus represents the execution behavior of a test.
// Maps to database ENUM: active, skipped, todo, focused, xfail
type TestStatus string

// Test status values aligned with DB schema.
const (
	// TestStatusActive indicates a normal test that runs and expects success.
	TestStatusActive TestStatus = "active"
	// TestStatusSkipped indicates a test intentionally excluded from execution.
	TestStatusSkipped TestStatus = "skipped"
	// TestStatusTodo indicates a test not yet implemented.
	TestStatusTodo TestStatus = "todo"
	// TestStatusFocused indicates a debugging-only test (.only, fit).
	// CI should warn when focused tests are committed.
	TestStatusFocused TestStatus = "focused"
	// TestStatusXfail indicates a test expected to fail (pytest xfail, RSpec pending).
	TestStatusXfail TestStatus = "xfail"
)
