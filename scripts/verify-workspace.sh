#!/usr/bin/env bash
set -euo pipefail

usage() {
    echo "usage: scripts/verify-workspace.sh [build|test|vet|examples|all]" >&2
}

mode="${1:-all}"
case "$mode" in
    build|test|vet|examples|all) ;;
    *)
        usage
        exit 2
        ;;
esac

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

verify_home="$(mktemp -d "${TMPDIR:-/tmp}/gokart-verify-home.XXXXXX")"
verify_gocache="$(mktemp -d "${TMPDIR:-/tmp}/gokart-verify-gocache.XXXXXX")"
trap 'rm -rf "$verify_home" "$verify_gocache"' EXIT

mapfile -t workspace_modules < <(
    awk '
        /^use[[:space:]]+\(/ { in_use = 1; next }
        in_use && /^\)/ { in_use = 0; next }
        in_use {
            path = $1
            gsub(/"/, "", path)
            if (path != "") print path
        }
        /^use[[:space:]]+[^[:space:](]/ {
            path = $2
            gsub(/"/, "", path)
            if (path != "") print path
        }
    ' go.work
)

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
        docs/examples/openai/main.go
        docs/examples/postgres/main.go
        docs/examples/sqlite/main.go
        examples/cli-app/main.go
        examples/http-service/main.go
    )

    for example in "${examples[@]}"; do
        echo "== compile $example =="
        GOCACHE="$verify_gocache" go test "$example"
    done
}

case "$mode" in
    build) run_workspace build ;;
    test) run_workspace test ;;
    vet) run_workspace vet ;;
    examples) compile_ignored_examples ;;
    all)
        run_workspace build
        run_workspace test
        run_workspace vet
        compile_ignored_examples
        ;;
esac
