# gokart — Go toolkit multi-module repo

# All published submodules, including the independently installable CLI.
modules := "cache cli cmd/gokart logger migrate postgres sqlite web"

# Build all modules
build:
    scripts/verify-workspace.sh build

# Install gokart CLI with version from git tag
install:
    go install -ldflags "-X main.gokartVersion=$(git describe --tags --match 'v[0-9]*' --always --dirty)" ./cmd/gokart

# Test all modules
test:
    scripts/verify-workspace.sh test

# Vet all modules
vet:
    scripts/verify-workspace.sh vet

# Verify all workspace modules and ignored example files
verify:
    scripts/verify-workspace.sh all

# Scan reachable history for committed credentials
leaks:
    scripts/check-public-leaks.sh

# Create local tags for all submodules and the root. Pushing is a separate gate.
# Usage: just tag v0.12.0
tag version:
    #!/usr/bin/env bash
    set -euo pipefail
    v="{{version}}"
    if [[ ! "$v" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "error: version must match vX.Y.Z (got: $v)" >&2
        exit 1
    fi
    # Verify clean tree
    if [[ -n "$(git status --porcelain)" ]]; then
        echo "error: working tree is dirty — commit first" >&2
        exit 1
    fi
    # Preflight the complete tag set before any mutation.
    for mod in {{modules}}; do
        tag="$mod/$v"
        if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
            echo "error: tag already exists: $tag" >&2
            exit 1
        fi
    done
    if git rev-parse -q --verify "refs/tags/$v" >/dev/null; then
        echo "error: tag already exists: $v" >&2
        exit 1
    fi
    # Tag all submodules.
    for mod in {{modules}}; do
        tag="$mod/$v"
        echo "  tagging $tag"
        git tag -a "$tag" -m "Release $tag"
    done
    # Tag root
    echo "  tagging $v"
    git tag -a "$v" -m "Release $v"
    echo "Done. Local tags created at $v; nothing was pushed."
