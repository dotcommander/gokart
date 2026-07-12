#!/usr/bin/env bash
set -euo pipefail

if ! command -v gitleaks >/dev/null 2>&1; then
    echo "gitleaks is required: https://github.com/gitleaks/gitleaks" >&2
    exit 2
fi

gitleaks detect \
    --source . \
    --log-opts='--all' \
    --redact=100 \
    --no-banner \
    --no-color
