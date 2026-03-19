#!/bin/bash

set -e

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

section "Linting and Fixing Go Files"

info "Running modernize analysis with fixes..."
out=$(go tool modernize -fix -test ./... 2>&1 \
    | grep -v '\.gen\.go:') || true
if [ -n "$out" ]; then
    echo "$out"
    exit 1
fi
success "modernize passed!"

info "Running golangci-lint with fixes..."
go tool golangci-lint run --fix --timeout=30m0s ./...

success "Linting and fixing completed successfully!"