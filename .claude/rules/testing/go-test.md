---
paths:
  - "**/*_test.go"
---

# Go Testing Standards

## File Naming

Format: `{target-file-name}_test.go`

Example: `user.go` → `user_test.go`

## Test Functions

Format: `func TestXxx(t *testing.T)`. Write `TestMethodName` functions per method, compose subtests with `t.Run()`.

## Subtests

Pattern: `t.Run("case name", func(t *testing.T) {...})`. Each case should be independently executable. Call `t.Parallel()` when running in parallel.

## Table-Driven Tests

Recommended when multiple cases have similar structure. Define cases with `[]struct{ name, input, want, wantErr }`.

```go
tests := []struct {
    name    string
    input   int
    want    int
    wantErr bool
}{
    {"normal case", 5, 10, false},
    {"negative input", -1, 0, true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got, err := Func(tt.input)
        if (err != nil) != tt.wantErr { ... }
        if got != tt.want { ... }
    })
}
```

## Mocking

Utilize interface-based dependency injection. Prefer manual mocking; consider gomock for complex cases. Define test-only implementations within `_test.go`.

## Error Verification

Use `errors.Is()` and `errors.As()`. Avoid string comparison of error messages; verify with sentinel errors or error types instead.

## Setup/Teardown

Use `TestMain(m *testing.M)` for global setup/teardown. For individual test preparation, do it within each test function or extract to helper functions.

## Test Helpers

Extract repeated setup/verification into `testXxx(t *testing.T, ...)` helpers. Receive `*testing.T` as first argument and call `t.Helper()`.

## Benchmarks

Write `func BenchmarkXxx(b *testing.B)` for performance-critical code. Loop with `b.N` and use `b.ResetTimer()` to exclude setup time.
