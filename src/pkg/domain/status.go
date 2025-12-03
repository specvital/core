package domain

type TestStatus string

const (
	TestStatusFixme   TestStatus = "fixme"
	TestStatusOnly    TestStatus = "only"
	TestStatusPending TestStatus = "pending"
	TestStatusSkipped TestStatus = "skipped"
)
