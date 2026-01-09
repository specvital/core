---
paths:
  - "**/*.go"
---

# Go Standards

## Error Handling (Go-Specific)

- Use %w for error chains, %v for simple logging
- Wrap internal errors not to be exposed with %v
- Never ignore return errors from functions; handle them explicitly
- Sentinel errors: For expected conditions that callers must handle, use `var ErrNotFound = errors.New("not found")`

## File Structure

### Element Order in File

1. package declaration
2. import statements (grouped)
3. Constant definitions (const)
4. Variable definitions (var)
5. Type/Interface/Struct definitions
6. Constructor functions (New\*)
7. Methods (grouped by receiver type, alphabetically ordered)
8. Helper functions (alphabetically ordered)

## Interfaces and Structs

### Interface Definition Location

- Define interfaces in the package that uses them (Accept interfaces, return structs)
- Only separate shared interfaces used by multiple packages

### Pointer Receiver Rules

- Use pointer receivers for state modification, large structs (3+ fields), or when consistency is needed
- Use value receivers otherwise

## Context Usage

### Context Parameter

- Always pass as the first parameter
- Use `context.Background()` only in main and tests

## Testing

### Testing Libraries

- Prefer standard library's if + t.Errorf over assertion libraries like testify
- Prefer manual mocking over gomock

## Forbidden Practices

### init() Functions

**AI abuses init() for convenience. Enforce strict prohibition.**

- Avoid unless necessary for registration patterns (database drivers, plugins)
- Prefer explicit initialization functions for business logic
- Acceptable uses:
  - Driver/plugin registration (e.g., `database/sql` drivers)
  - Static route/handler registration with no I/O
  - Complex constant initialization without side effects
- Forbidden uses:
  - External I/O (database, file, network)
  - Global state mutation
  - Error-prone initialization (use constructors that return errors)
  - **Any convenience initialization that AI suggests** - always question init() in code review

## Package Structure

### internal Package

- Actively use for libraries, use only when necessary for applications

## Recommended Libraries

- Web: chi
- DB: Bun, SQLBoiler (when managing migrations externally)
- Logging: slog
- CLI: cobra
- Utilities: samber/lo, golang.org/x/sync
- Configuration: koanf (viper if cobra integration needed)
- Validation: go-playground/validator/v10
- Scheduling: github.com/go-co-op/gocron
- Image processing: github.com/h2non/bimg
