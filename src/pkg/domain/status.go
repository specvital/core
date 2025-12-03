package domain

// TestStatus represents the execution status of a test.
type TestStatus string

const (
	TestStatusPending TestStatus = "pending" // Normal test (will run)
	TestStatusSkipped TestStatus = "skipped" // .skip, xit, xtest, etc. cspell:disable-line
	TestStatusOnly    TestStatus = "only"    // .only (focus)
	TestStatusFixme   TestStatus = "fixme"   // test.fixme (Playwright)
)
