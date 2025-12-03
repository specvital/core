package domain

type TestStatus string

const (
	TestStatusPending TestStatus = "pending"
	TestStatusSkipped TestStatus = "skipped"
	TestStatusOnly    TestStatus = "only"
	TestStatusFixme   TestStatus = "fixme"
)
