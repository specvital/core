# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SpecVital Core - Shared Go library for specvital ecosystem

- Test file parsing (20+ frameworks, tree-sitter AST)
- Cryptography utilities (encryption shared by web & collector)
- Source abstraction (local filesystem, git repos)
- Domain models

## Documentation Map

| Context                 | Reference        |
| ----------------------- | ---------------- |
| API usage / Quick start | `README.md`      |
| Available commands      | `just --list`    |
| Coding rules            | `.claude/rules/` |
| Architecture details    | `README.md`      |

## Commands

Before running commands, check available commands via `just --list`

```bash
# Common commands
just test              # Run all tests
just test unit         # Unit tests only
just test integration  # Integration tests (clones real repos, ~10min)
just snapshot-update   # Update golden snapshots after intentional changes
```

## Project-Specific Rules

### Framework Registration Pattern (CRITICAL)

**Exception to init() rule**: Framework registration via init() is required.

Each framework MUST:

1. Define `framework.Definition` in `strategies/{name}/definition.go`
2. Register via `framework.Register()` in `init()`
3. Require blank import: `_ "github.com/specvital/core/pkg/parser/strategies/{name}"`

**Why**: Go's init() only runs if package is imported. Missing blank import = framework silently not detected.

### Detection Confidence Scoring

When modifying detection logic, preserve 4-stage hierarchy:

| Stage    | Score | Signal                        |
| -------- | ----- | ----------------------------- |
| Scope    | 80pts | Within config file scope      |
| Import   | 60pts | Explicit framework imports    |
| Content  | 40pts | Framework-specific patterns   |
| Filename | 20pts | Naming conventions (fallback) |

**Why**: Prevents false positives for similar frameworks (Jest vs Vitest).

### API Stability

- **Public API** (`pkg/parser/`): Breaking changes require major version bump
- **Internal** (`pkg/parser/{strategies,detection,framework,tspool}/`): Safe to refactor

### Tree-sitter Constraints

- Parser pooling is disabled (tree-sitter cancellation flag bug)
- Fresh parser created per-use via `tspool.Parse()`
- Language grammars initialized once via `sync.Once`

## Common Workflows

### Adding New Framework

1. Create `pkg/parser/strategies/{framework}/definition.go`
2. Implement matchers (scope, import, content, filename)
3. Implement config parser (if framework has config file)
4. Implement test parser using tree-sitter queries
5. Register in `init()` with `framework.Register()`
6. Add blank import to `README.md` Quick Start
7. Add integration test in `tests/integration/repos.yaml`

### Updating Integration Test Snapshots

**When**: After intentional parser changes (new features, bug fixes)
**Never**: To "fix" failing tests without understanding why output changed

```bash
just snapshot-update              # Update all
just snapshot-update repo=next.js # Update specific repo
```

## Architecture

```
pkg/
├── domain/           # Domain models (Inventory, TestFile, TestSuite, Test)
├── parser/
│   ├── scanner.go    # Public API: Scan()
│   ├── framework/    # Definition + Registry + Matchers
│   ├── detection/    # Confidence-based detection
│   ├── strategies/   # Framework implementations (jest, vitest, playwright, ...)
│   └── tspool/       # Tree-sitter parser lifecycle
└── source/           # Source abstraction (local, git)
```
