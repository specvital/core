---
name: project-structure
description: |
  Provides comprehensive project folder structure design guidelines and best practices. Defines standard directory organizations for diverse project types including monorepos, web frameworks, backend services, libraries, and extensions. Ensures scalable, maintainable architecture through consistent file organization patterns. Specializes in separation of concerns, modular architecture, and tooling integration.
  Use when: designing new project structures, organizing monorepo workspaces with tools like Turborepo/Nx, structuring NestJS backend projects, organizing React/Next.js frontend applications, designing Go service architectures, creating NPM package structures, organizing VSCode extension projects, structuring Chrome extension codebases, planning directory hierarchies, migrating legacy project structures, or establishing code organization conventions for teams.
---

# Project Structure Guide

## Monorepo

```
project-root/
├── src/                         # All services/apps
├── infra/                       # Shared infrastructure
├── docs/                        # Documentation
├── .devcontainer/               # Dev Container configuration
├── .github/                     # Workflows, templates
├── .vscode/                     # VSCode settings
├── .claude/                     # Claude settings
├── .gemini/                     # Gemini settings
├── package.json                 # Root package.json. For releases, version management
├── go.work                      # Go workspace (when using Go)
├── justfile                     # Just task runner
├── .gitignore
├── .prettierrc
├── .prettierignore
└── README.md
```

## Go

```
project-root/
├── cmd/                    # Execution entry points (main.go)
├── internal/               # Private packages
├── pkg/                    # Public packages
├── configs/                # Configuration files
├── scripts/                # Utility scripts
├── tests/                  # Integration tests
├── docs/                   # Documentation
├── go.mod
└── go.sum
```
