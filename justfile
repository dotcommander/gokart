# gokart — Go toolkit multi-module repo

# All submodules (order: leaves first, then modules with cross-deps)
# Submodules with tracked content (cache and cmd/gokart excluded — not published)
modules := "ai cli fs kv logger migrate postgres sqlite web"

# Build all modules
build:
    go build ./...

# Test all modules
test:
    go test ./...

# Vet all modules
vet:
    go vet ./...

# Tag all submodules + root with the given version, then push.
# Usage: just tag v0.8.0
tag version:
    #!/usr/bin/env bash
    set -euo pipefail
    v="{{version}}"
    if [[ ! "$v" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "error: version must match vX.Y.Z (got: $v)" >&2
        exit 1
    fi
    # Verify clean tree
    if ! git diff --quiet || ! git diff --cached --quiet; then
        echo "error: working tree is dirty — commit first" >&2
        exit 1
    fi
    # Update version refs in go.mod files that cross-reference submodules
    echo "Updating go.mod cross-references to $v..."
    sed -i '' "s|gokart/logger v[^ ]*|gokart/logger $v|" go.mod
    # Verify build
    echo "Building..."
    go build ./...
    go vet ./...
    # Commit version bump
    git add go.mod
    if ! git diff --cached --quiet; then
        git commit -m "chore: bump cross-module refs to $v"
    fi
    # Tag all submodules
    for mod in {{modules}}; do
        tag="$mod/$v"
        echo "  tagging $tag"
        git tag "$tag"
    done
    # Tag root
    echo "  tagging $v"
    git tag "$v"
    # Push
    echo "Pushing commit + tags..."
    git push
    git push --tags
    echo "Done. All modules tagged at $v."
