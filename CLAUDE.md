# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### justfile

- `just deps` - Install dependencies (pnpm install)
- `just lint` - Run all linters (justfile, config, go)
- `just lint go` - Format Go code with gofmt
- `just test` - Run Go tests
- `just release` - Trigger release (merge to release branch)

### Single Test Execution

```bash
go test ./pkg/parser/strategies/jest/... -run TestJest
go test ./pkg/domain -v
```

## Architecture

### Core Structure

```
pkg/
├── domain/           # Domain models (Inventory, TestFile, TestSuite, Test)
└── parser/
    ├── scanner.go    # Entry point: Scan(), DetectTestFiles()
    ├── framework/    # Unified framework definition system
    │   ├── definition.go  # Definition type (Matcher + ConfigParser + Parser)
    │   ├── registry.go    # Single registry for all frameworks
    │   ├── scope.go       # ConfigScope with root resolution
    │   └── matchers/      # Reusable matchers (import, config, content)
    ├── detection/    # Confidence-based framework detection
    │   ├── detector.go    # Multi-stage detection (Scope→Import→Content→Filename)
    │   └── result.go      # Detection result with evidence
    ├── strategies/   # Framework-specific implementations
    │   ├── jest/definition.go
    │   ├── vitest/definition.go
    │   ├── playwright/definition.go
    │   ├── gotesting/definition.go
    │   └── shared/jstest/  # Shared JS test parsing
    └── tspool/       # Tree-sitter parser pooling
```

### Unified Framework Definition

- Each framework provides single `framework.Definition` (Matchers + ConfigParser + Parser)
- Auto-registered via `framework.Register()` in `init()`
- Blank import required: `_ "github.com/specvital/core/pkg/parser/strategies/jest"`

### Confidence-Based Detection

Detection uses 4-stage scoring:

- **Scope (80pts)**: File within config scope (with root resolution)
- **Import (60pts)**: Explicit framework imports
- **Content (40pts)**: Framework-specific patterns (jest.fn, etc.)
- **Filename (20pts)**: File naming patterns

### Concurrency Model

- `scanner.go`: Parallel parsing with errgroup + semaphore
- `tspool/`: Tree-sitter parser reuse via sync.Pool
