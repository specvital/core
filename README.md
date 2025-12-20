# specvital-core

Shared Go library for the specvital ecosystem.

## Packages

| Package  | Description                                           |
| -------- | ----------------------------------------------------- |
| `parser` | Test file parsing (20+ frameworks, tree-sitter)       |
| `crypto` | NaCl SecretBox encryption (shared by web & collector) |
| `source` | Source abstraction (local filesystem, git repos)      |
| `domain` | Domain models (Inventory, TestFile, TestSuite)        |

## Installation

```bash
go get github.com/specvital/core
```

## Parser

### Quick Start

```go
package main

import (
    "context"
    "fmt"

    "github.com/specvital/core/pkg/parser"

    // Import frameworks to register them
    _ "github.com/specvital/core/pkg/parser/strategies/all" // All frameworks
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

### Scan Options

```go
result, err := parser.Scan(ctx, rootPath,
    parser.WithWorkers(4),                    // Parallel workers (default: GOMAXPROCS)
    parser.WithTimeout(2*time.Minute),        // Scan timeout (default: 5 minutes)
    parser.WithExclude([]string{"fixtures"}), // Additional skip directories
    parser.WithScanPatterns([]string{"**/*.test.ts"}), // Glob patterns
)
```

### Supported Frameworks

| Language      | Frameworks                               |
| ------------- | ---------------------------------------- |
| JavaScript/TS | Jest, Vitest, Playwright, Cypress, Mocha |
| Go            | go testing                               |
| Python        | pytest, unittest                         |
| Java          | JUnit 5, TestNG                          |
| Kotlin        | Kotest                                   |
| C#            | NUnit, xUnit, MSTest                     |
| Ruby          | RSpec, Minitest                          |
| PHP           | PHPUnit                                  |
| Rust          | cargo test                               |
| C++           | Google Test                              |
| Swift         | XCTest                                   |

### Selective Import

Import only needed frameworks for smaller binaries:

```go
import (
    _ "github.com/specvital/core/pkg/parser/strategies/jest"
    _ "github.com/specvital/core/pkg/parser/strategies/vitest"
    _ "github.com/specvital/core/pkg/parser/strategies/playwright"
)
```

### Data Structures

```go
type Inventory struct {
    RootPath string     // Scanned directory
    Files    []TestFile // Parsed test files
}

type TestFile struct {
    Path      string      // Relative file path
    Framework string      // "jest", "vitest", "playwright", "go", ...
    Language  Language    // "typescript", "javascript", "go", ...
    Suites    []TestSuite // Test suites (describe blocks)
    Tests     []Test      // Top-level tests
}

type TestSuite struct {
    Name     string      // Suite name
    Location Location    // Source location (line, column)
    Status   TestStatus  // "", "skipped", "only", ...
    Suites   []TestSuite // Nested suites
    Tests    []Test      // Tests in this suite
}

type Test struct {
    Name     string     // Test name
    Location Location   // Source location
    Status   TestStatus // "", "skipped", "only", "pending", "fixme"
}
```

## Crypto

NaCl SecretBox encryption for sensitive data (OAuth tokens, etc.).

### Quick Start

```go
import "github.com/specvital/core/pkg/crypto"

// Create encryptor from Base64-encoded key
enc, err := crypto.NewEncryptorFromBase64(os.Getenv("ENCRYPTION_KEY"))
if err != nil {
    log.Fatal(err)
}
defer enc.Close()

// Encrypt
encrypted, err := enc.Encrypt(sensitiveData)

// Decrypt
decrypted, err := enc.Decrypt(encrypted)
```

### Key Generation

```bash
openssl rand -base64 32
```

### Security

- XSalsa20 stream cipher (256-bit key)
- Poly1305 MAC (integrity protection)
- Random 192-bit nonce per encryption
- Thread-safe for concurrent use

## Source

Abstraction layer for reading files from different sources.

```go
import "github.com/specvital/core/pkg/source"

// Local filesystem
src, err := source.NewLocalSource("./my-project")
defer src.Close()

// Git repository (clones to temp dir)
src, err := source.NewGitSource(ctx, "https://github.com/org/repo.git",
    source.WithRef("main"),
    source.WithDepth(1),
)
defer src.Close() // Cleans up temp directory
```

## Development

```bash
# Unit tests
just test unit

# Integration tests (clones real GitHub repos, ~10min)
just test integration

# All tests
just test

# Update golden snapshots
just snapshot-update
```

## License

MIT
