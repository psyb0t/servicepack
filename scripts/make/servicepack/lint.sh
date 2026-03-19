#!/bin/bash

set -e

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

section "Linting Go Files"

info "Running modernize analysis..."
out=$(go tool modernize -test ./... 2>&1 \
    | grep -v '\.gen\.go:') || true
if [ -n "$out" ]; then
    echo "$out"
    exit 1
fi
success "modernize passed!"

info "Running golangci-lint..."
go tool golangci-lint run --timeout=30m0s ./...

success "Linting completed successfully!"