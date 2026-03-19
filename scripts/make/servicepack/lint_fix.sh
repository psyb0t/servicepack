#!/bin/bash

set -e

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

section "Linting and Fixing Go Files"

info "Running modernize analysis with fixes..."
gen_files=$(find . -name '*.gen.go' -not -path './vendor/*')
out=$(go tool modernize -fix -test ./... 2>&1 \
    | grep -v '\.gen\.go:') || true
# revert any changes modernize made to generated files
if [ -n "$gen_files" ]; then
    echo "$gen_files" | xargs git checkout -- 2>/dev/null || true
fi
if [ -n "$out" ]; then
    echo "$out"
    exit 1
fi
success "modernize passed!"

info "Running golangci-lint with fixes..."
go tool golangci-lint run --fix --timeout=30m0s ./...

success "Linting and fixing completed successfully!"