#!/usr/bin/env bash
# Run the same golangci-lint checks as CI (.github/workflows/ci.yml).
# Fails closed if golangci-lint is unavailable, so a push never ships
# code that would fail the Lint steps.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if ! command -v golangci-lint >/dev/null 2>&1; then
    echo "error: golangci-lint not found on PATH; install it (https://golangci-lint.run) and retry" >&2
    echo "       refusing to report success without running the linters" >&2
    exit 1
fi

echo "== lint root module =="
golangci-lint run ./...

echo "== lint cmd/gokart module =="
(cd cmd/gokart && golangci-lint run ./...)

echo "lint: clean"
