#!/usr/bin/env bash
set -euo pipefail

usage() {
    echo "usage: scripts/verify-workspace.sh [build|test|vet|examples|standalone|all]" >&2
}

mode="${1:-all}"
case "$mode" in
    build|test|vet|examples|standalone|all) ;;
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
GOWORK="$verify_workspace" go work edit -replace=github.com/dotcommander/gokart@v0.11.0="$repo_root"
GOWORK="$verify_workspace" go work edit -replace=github.com/dotcommander/gokart/cli@v0.11.0="$repo_root/cli"
GOWORK="$verify_workspace" go work edit -replace=github.com/dotcommander/gokart/web@v0.11.0="$repo_root/web"
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
        examples/cli-app/main.go
        examples/http-service/main.go
    )

    for example in "${examples[@]}"; do
        echo "== compile $example =="
        GOCACHE="$verify_gocache" go test "$example"
    done
}

verify_standalone_modules() {
    for module in "${workspace_modules[@]}"; do
        echo "== standalone test $module =="
        safe_name="${module//\//-}"
        modfile="$verify_home/$safe_name.mod"
        cp "$module/go.mod" "$modfile"
        if [[ -f "$module/go.sum" ]]; then
            cp "$module/go.sum" "${modfile%.mod}.sum"
        fi
        for local_module in "${workspace_modules[@]}"; do
            module_path="$(awk '$1 == "module" { print $2; exit }' "$local_module/go.mod")"
            GOWORK=off go mod edit -modfile="$modfile" -replace="$module_path=$repo_root/$local_module"
        done
        (cd "$module" && HOME="$verify_home" GOCACHE="$verify_gocache" GOWORK=off go test -modfile="$modfile" -mod=readonly ./...)
    done
}

case "$mode" in
    build) run_workspace build ;;
    test) run_workspace test ;;
    vet) run_workspace vet ;;
    examples) compile_ignored_examples ;;
    standalone) verify_standalone_modules ;;
    all)
        run_workspace build
        run_workspace test
        run_workspace vet
        compile_ignored_examples
        verify_standalone_modules
        ;;
esac
