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

verify_generated_projects() {
    binary="$verify_home/gokart"
    GOCACHE="$verify_gocache" go build -o "$binary" ./cmd/gokart

    cases=(
        "flat|"
        "structured|--structured"
        "global|--structured --global"
        "sqlite|--db sqlite"
        "postgres|--db postgres"
        "ai|--ai"
        "redis|--redis"
    )
    for entry in "${cases[@]}"; do
        name="${entry%%|*}"
        flags="${entry#*|}"
        target="$verify_home/generated-$name"
        echo "== generated project $name =="
        HOME="$verify_home" GOCACHE="$verify_gocache" "$binary" new "$target" --no-verify $flags >/dev/null 2>&1
        case "$name" in
            global)
                (cd "$target" && GOWORK=off go mod edit -require=github.com/dotcommander/gokart@v0.12.0 -replace=github.com/dotcommander/gokart="$repo_root")
                ;;
            sqlite|postgres|redis)
                module="$name"
                [[ "$name" == "redis" ]] && module="cache"
                (cd "$target" && GOWORK=off go mod edit \
                    -require=github.com/dotcommander/gokart/$module@v0.12.0 \
                    -replace=github.com/dotcommander/gokart="$repo_root" \
                    -replace=github.com/dotcommander/gokart/$module="$repo_root/$module")
                ;;
        esac
        (cd "$target" && HOME="$verify_home" GOCACHE="$verify_gocache" GOWORK=off go mod tidy)
        (cd "$target" && HOME="$verify_home" GOCACHE="$verify_gocache" GOWORK=off go test -mod=readonly ./...)
        case "$name" in
            flat)
                output=$(cd "$target" && HOME="$verify_home" GOCACHE="$verify_gocache" GOWORK=off go run .)
                ;;
            structured)
                output=$(cd "$target" && HOME="$verify_home" GOCACHE="$verify_gocache" GOWORK=off go run ./cmd)
                ;;
            *)
                output=""
                ;;
        esac
        if [[ "$name" == "flat" || "$name" == "structured" ]] && [[ "$output" != *"Usage:"* ]]; then
            echo "generated $name run did not print usage" >&2
            return 1
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
