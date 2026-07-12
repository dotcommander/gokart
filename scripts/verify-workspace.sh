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
verify_gomodcache="$(go env GOMODCACHE)"

cleanup() {
    rm -rf "$verify_home" "$verify_gocache"
}
trap cleanup EXIT

verify_workspace="$verify_home/go.work"
workspace_modules=(. ai cache cli cmd/gokart fs logger migrate postgres sqlite web)
GOWORK="$verify_workspace" go work init
for module in "${workspace_modules[@]}"; do
    GOWORK="$verify_workspace" go work use "$repo_root/$module"
done
GOWORK="$verify_workspace" go work edit -replace=github.com/dotcommander/gokart@v0.10.2="$repo_root"
GOWORK="$verify_workspace" go work edit -replace=github.com/dotcommander/gokart/cli@v0.10.2="$repo_root/cli"
GOWORK="$verify_workspace" go work edit -replace=github.com/dotcommander/gokart/web@v0.10.2="$repo_root/web"
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
