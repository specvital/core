# specvital-core

Test file parser library for multiple test frameworks.

## Features

- **Multi-framework support**: Jest, Vitest, Playwright, Go testing
- **Parallel processing**: Concurrent file scanning with configurable worker pool
- **Tree-sitter based**: Accurate AST parsing for JavaScript, TypeScript, and Go
- **Performance optimized**: Parser pooling and query caching

## Installation

```bash
go get github.com/specvital/core
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"

    "github.com/specvital/core/pkg/parser"

    // Import strategies to register them
    _ "github.com/specvital/core/pkg/parser/strategies/gotesting"
    _ "github.com/specvital/core/pkg/parser/strategies/jest"
    _ "github.com/specvital/core/pkg/parser/strategies/playwright"
    _ "github.com/specvital/core/pkg/parser/strategies/vitest"
)

func main() {
    ctx := context.Background()

    result, err := parser.Scan(ctx, "./my-project")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Found %d test files with %d tests\n",
        len(result.Inventory.Files),
        result.Inventory.CountTests())
}
```

## API Reference

### Scan

Scans a directory for test files and parses them.

```go
result, err := parser.Scan(ctx, rootPath,
    parser.WithWorkers(4),               // Parallel workers (default: GOMAXPROCS)
    parser.WithTimeout(2*time.Minute),   // Scan timeout (default: 5 minutes)
    parser.WithExclude([]string{"fixtures"}), // Additional skip directories
    parser.WithScanPatterns([]string{"**/*.test.ts"}), // Glob patterns to filter
)
```

### DetectTestFiles

Detects test files without parsing.

```go
result, err := parser.DetectTestFiles(ctx, rootPath,
    parser.WithPatterns([]string{"src/**/*.spec.ts"}),
    parser.WithMaxFileSize(5*1024*1024), // 5MB max
    parser.WithSkipPatterns([]string{"node_modules", ".git"}),
)
```

## Supported Test Patterns

| Framework  | File Patterns                           | Test Functions      |
| ---------- | --------------------------------------- | ------------------- |
| Jest       | `*.test.ts`, `*.spec.ts`, `__tests__/*` | describe, it, test  |
| Vitest     | Same as Jest + vitest.config.ts         | describe, it, test  |
| Playwright | `*.test.ts` + playwright.config.ts      | test, test.describe |
| Go         | `*_test.go`                             | func TestXxx, t.Run |

## Data Structures

### Inventory

```go
type Inventory struct {
    RootPath string     // Scanned directory
    Files    []TestFile // Parsed test files
}
```

### TestFile

```go
type TestFile struct {
    Path      string      // File path
    Framework string      // "jest", "vitest", "playwright", "go"
    Language  Language    // "typescript", "javascript", "go"
    Suites    []TestSuite // Test suites (describe blocks)
    Tests     []Test      // Top-level tests
}
```

### TestSuite

```go
type TestSuite struct {
    Name     string      // Suite name
    Location Location    // Source location
    Status   TestStatus  // "", "skipped", "only", etc.
    Suites   []TestSuite // Nested suites
    Tests    []Test      // Tests in this suite
}
```

### Test

```go
type Test struct {
    Name     string     // Test name
    Location Location   // Source location
    Status   TestStatus // "", "skipped", "only", "pending", "fixme"
}
```

## Performance

- Parser pooling via `sync.Pool` for concurrent parsing
- Query compilation caching for repeated tree-sitter queries
- Configurable worker count for parallel file processing
- Context-based cancellation and timeout support

## Development

### Running Tests

```bash
# Unit tests only
just test unit

# Integration tests (clones real GitHub repos)
just test integration

# All tests
just test
```

### Integration Tests

Integration tests validate the parser against real open-source repositories:

- **Single-framework repos**: testing-library/react, vite, playwright, gin
- **Complex cases**: next.js, storybook, turborepo, grafana, trpc, prisma, remix, etcd

Repositories are shallow-cloned and cached in `tests/integration/testdata/cache/`.

To update golden snapshots after parser changes:

```bash
go test -tags integration ./tests/integration/... -update
```

## License

MIT
