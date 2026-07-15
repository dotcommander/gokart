#!/usr/bin/env bash
set -euo pipefail

usage() {
    echo "usage: scripts/verify-workspace.sh [build|test|vet|examples|generated|standalone|all]" >&2
}

mode="${1:-all}"
case "$mode" in
    build|test|vet|examples|generated|standalone|all) ;;
    *)
        usage
        exit 2
        ;;
esac

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

verify_home="$(mktemp -d "${TMPDIR:-/tmp}/gokart-verify-home.XXXXXX")"
verify_gocache="$(mktemp -d "${TMPDIR:-/tmp}/gokart-verify-gocache.XXXXXX")"
verify_gomodcache="$(go env GOMODCACHE)"

cleanup() {
    rm -rf "$verify_home" "$verify_gocache"
}
trap cleanup EXIT

verify_workspace="$verify_home/go.work"
workspace_modules=()
while IFS= read -r module; do
    workspace_modules+=("$module")
done < <(
    awk '
        /^use \(/ { in_use = 1; next }
        in_use && /^\)/ { in_use = 0; next }
        in_use { sub(/^\.\//, "", $1); print $1 }
        /^use [^\(]/ { module = $2; sub(/^\.\//, "", module); print module }
    ' go.work
)
GOWORK="$verify_workspace" go work init
for module in "${workspace_modules[@]}"; do
    GOWORK="$verify_workspace" go work use "$repo_root/$module"
done
export GOWORK="$verify_workspace"
export GOMODCACHE="$verify_gomodcache"

run_workspace() {
    local action="$1"

    for module in "${workspace_modules[@]}"; do
        echo "== $action $module =="
        case "$action" in
            build) (cd "$module" && GOCACHE="$verify_gocache" go build ./...) ;;
            test) (cd "$module" && HOME="$verify_home" GOCACHE="$verify_gocache" go test ./...) ;;
            vet) (cd "$module" && GOCACHE="$verify_gocache" go vet ./...) ;;
        esac
    done
}

compile_ignored_examples() {
    local examples=(
        docs/examples/cache/main.go
        docs/examples/config/main.go
        docs/examples/full-service/main.go
        docs/examples/http-server/main.go
        docs/examples/logger/main.go
        docs/examples/postgres/main.go
        docs/examples/sqlite/main.go
    )

    for example in "${examples[@]}"; do
        echo "== compile $example =="
        GOCACHE="$verify_gocache" go test "$example"
    done
}

verify_standalone_modules() {
    for module in "${workspace_modules[@]}"; do
        echo "== standalone tidy $module =="
        (cd "$module" && HOME="$verify_home" GOCACHE="$verify_gocache" GOWORK=off go mod tidy -diff)
        echo "== standalone verify $module =="
        (cd "$module" && HOME="$verify_home" GOCACHE="$verify_gocache" GOWORK=off go mod verify)
        echo "== standalone test $module =="
        (cd "$module" && HOME="$verify_home" GOCACHE="$verify_gocache" GOWORK=off go test -mod=readonly ./...)
    done
}

verify_standalone_examples() {
    for example in examples/cli-app examples/http-service; do
        echo "== standalone tidy $example =="
        (cd "$example" && HOME="$verify_home" GOCACHE="$verify_gocache" GOWORK=off go mod tidy -diff)
        echo "== standalone verify $example =="
        (cd "$example" && HOME="$verify_home" GOCACHE="$verify_gocache" GOWORK=off go mod verify)
        echo "== standalone test $example =="
        (cd "$example" && HOME="$verify_home" GOCACHE="$verify_gocache" GOWORK=off go test -mod=readonly ./...)
    done
}

prepare_local_generated_module() {
    local target="$1"
    local module="$2"
    local local_module module_path

    mkdir -p "$target"
    (cd "$target" && GOWORK=off go mod init "$module" >/dev/null)
    for local_module in "${workspace_modules[@]}"; do
        module_path="$(awk '$1 == "module" { print $2; exit }' "$local_module/go.mod")"
        (cd "$target" && GOWORK=off go mod edit -replace="$module_path=$repo_root/$local_module")
    done
}

verify_generated_projects() {
    binary="$verify_home/gokart"
    GOCACHE="$verify_gocache" go build -o "$binary" ./cmd/gokart

    cases=(
        "flat|||flat"
        "flat-example|--example|example|flat"
        "flat-global-example|--global --example|example|flat"
        "structured|--structured||structured"
        "structured-example|--structured --example|example|structured"
        "structured-global-example|--structured --global --example|example|structured"
        "structured-unmanaged|--structured --no-manifest||structured"
        "sqlite|--db sqlite||structured"
        "postgres|--db postgres||structured"
        "ai|--ai||structured"
        "redis-example|--redis --example|example|structured"
        "combo-example|--db postgres --ai --redis --example|example|structured"
    )
    for entry in "${cases[@]}"; do
        IFS='|' read -r name flags example layout <<< "$entry"
        target="$verify_home/generated-$name"
        echo "== generated project $name =="
        # Preserve a local-only go.mod so pending release versions resolve from
        # this checkout before their public tags exist.
        prepare_local_generated_module "$target" "generated-$name"
        HOME="$verify_home" GOCACHE="$verify_gocache" "$binary" new "$target" --no-verify --skip-existing $flags >/dev/null 2>&1
        case "$name" in
            flat-global-example|structured-global-example)
                (cd "$target" && GOWORK=off go mod edit -require=github.com/dotcommander/gokart@v0.13.0 -replace=github.com/dotcommander/gokart="$repo_root")
                ;;
            sqlite|postgres|redis-example)
                module="$name"
                [[ "$name" == "redis-example" ]] && module="cache"
                (cd "$target" && GOWORK=off go mod edit \
                    -require=github.com/dotcommander/gokart/$module@v0.13.0 \
                    -replace=github.com/dotcommander/gokart="$repo_root" \
                    -replace=github.com/dotcommander/gokart/$module="$repo_root/$module")
                ;;
            combo-example)
                (cd "$target" && GOWORK=off go mod edit \
                    -require=github.com/dotcommander/gokart/postgres@v0.13.0 \
                    -require=github.com/dotcommander/gokart/cache@v0.13.0 \
                    -replace=github.com/dotcommander/gokart/postgres="$repo_root/postgres" \
                    -replace=github.com/dotcommander/gokart/cache="$repo_root/cache")
                ;;
        esac
        (cd "$target" && HOME="$verify_home" GOCACHE="$verify_gocache" GOWORK=off go mod tidy)
        (cd "$target" && HOME="$verify_home" GOCACHE="$verify_gocache" GOWORK=off go test -mod=readonly ./...)
        generated_binary="$verify_home/$name"
        if [[ "$layout" == "flat" ]]; then
            (cd "$target" && HOME="$verify_home" GOCACHE="$verify_gocache" GOWORK=off go build -o "$generated_binary" .)
        else
            (cd "$target" && HOME="$verify_home" GOCACHE="$verify_gocache" GOWORK=off go build -o "$generated_binary" ./cmd)
        fi

        output=$(cd "$target" && HOME="$verify_home" "$generated_binary")
        if [[ "$output" != *"Usage:"* ]]; then
            echo "generated $name run did not print usage" >&2
            return 1
        fi

        if [[ "$example" == "example" ]]; then
            output=$(cd "$target" && HOME="$verify_home" "$generated_binary" greet --name World)
            if [[ "$output" != "Hello, World" ]]; then
                echo "generated $name greet output mismatch: $output" >&2
                return 1
            fi
        fi

        if [[ "$layout" == "flat" ]]; then
            if [[ -e "$target/.gokart-manifest.json" ]]; then
                echo "generated $name unexpectedly wrote a manifest" >&2
                return 1
            fi
        elif [[ "$name" == "structured-unmanaged" ]]; then
            if [[ -e "$target/.gokart-manifest.json" ]]; then
                echo "generated $name unexpectedly wrote a manifest" >&2
                return 1
            fi
        elif [[ ! -e "$target/.gokart-manifest.json" ]]; then
            echo "generated $name omitted its manifest" >&2
            return 1
        fi

        if [[ "$name" == "structured-example" ]]; then
            (cd "$target" && HOME="$verify_home" GOCACHE="$verify_gocache" "$binary" add sqlite --dry-run >/dev/null)
        fi
    done
}

case "$mode" in
    build) run_workspace build ;;
    test) run_workspace test ;;
    vet) run_workspace vet ;;
    examples) compile_ignored_examples ;;
    generated) verify_generated_projects ;;
    standalone)
        verify_standalone_modules
        verify_standalone_examples
        ;;
    all)
        run_workspace build
        run_workspace test
        run_workspace vet
        compile_ignored_examples
        verify_standalone_modules
        verify_standalone_examples
        verify_generated_projects
        ;;
esac
