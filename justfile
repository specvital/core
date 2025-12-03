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
        cd {{ root_dir }}/src && gofmt -w .
        ;;
      *)
        echo "Unknown target: {{ target }}"
        exit 1
        ;;
    esac

test:
    cd {{ root_dir }}/src && go list -f '{{ "{{" }}.Dir{{ "}}" }}/...' -m | xargs go test
