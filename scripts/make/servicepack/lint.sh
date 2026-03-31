#!/bin/bash

set -e

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

section "Linting Go Files"

info "Running go fix (diff only)..."
out=$(go fix -diff ./... 2>&1) || true
if [ -n "$out" ]; then
    echo "$out"
    error "go fix found issues. Run 'make lint-fix' to apply."
    exit 1
fi
success "go fix passed!"

info "Running golangci-lint..."
go tool golangci-lint run --timeout=30m0s ./...

success "Linting completed successfully!"
