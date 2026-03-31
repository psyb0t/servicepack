#!/bin/bash

set -e

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

section "Linting and Fixing Go Files"

info "Running go fix..."
go fix ./...
success "go fix passed!"

info "Running golangci-lint with fixes..."
go tool golangci-lint run --fix --timeout=30m0s ./...

success "Linting and fixing completed successfully!"
