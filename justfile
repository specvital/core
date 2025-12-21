set dotenv-load := true

root_dir := justfile_directory()

deps: deps-root

deps-root:
    pnpm install

lint target="all":
    #!/usr/bin/env bash
    set -euox pipefail
    case "{{ target }}" in
      all)
        just lint justfile
        just lint config
        just lint go
        ;;
      justfile)
        just --fmt --unstable
        ;;
      config)
        npx prettier --write "**/*.{json,yml,yaml,md}"
        ;;
      go)
        cd {{ root_dir }} && gofmt -w ./pkg
        ;;
      *)
        echo "Unknown target: {{ target }}"
        exit 1
        ;;
    esac

release:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "⚠️  WARNING: This will trigger a production release!"
    echo ""
    echo "GitHub Actions will automatically:"
    echo "  - Analyze commits to determine version bump"
    echo "  - Generate release notes"
    echo "  - Create tag and GitHub release"
    echo "  - Update CHANGELOG.md"
    echo ""
    echo "Progress: https://github.com/specvital/core/actions"
    echo ""
    read -p "Type 'yes' to continue: " confirm
    if [ "$confirm" != "yes" ]; then
        echo "Aborted."
        exit 1
    fi
    git checkout release
    git merge main
    git push origin release
    git checkout main
    echo "✅ Release triggered! Check GitHub Actions for progress."

snapshot-update repo="all":
    #!/usr/bin/env bash
    set -euox pipefail
    cd {{ root_dir }}
    if [ "{{ repo }}" = "all" ]; then
        go test -tags integration ./tests/integration/... -v -timeout 15m -update
    else
        go test -tags integration ./tests/integration/... -v -timeout 15m -update -run "TestSingleFramework/{{ repo }}"
    fi
    just lint config

sync-docs:
    baedal specvital/specvital.github.io/docs docs --exclude ".vitepress/**"

test target="all":
    #!/usr/bin/env bash
    set -euox pipefail
    cd {{ root_dir }}
    case "{{ target }}" in
      all)
        just test unit
        just test integration
        ;;
      unit)
        go test ./...
        ;;
      integration)
        go test -tags integration ./tests/integration/... -v -timeout 15m
        ;;
      *)
        echo "Unknown target: {{ target }}"
        echo "Available: unit, integration, all"
        exit 1
        ;;
    esac

# Scan a directory for tests (used by /validate-parser command)
scan path:
    go run {{ root_dir }}/scripts/scan.go {{ path }}
