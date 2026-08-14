#!/bin/bash

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

section "Linting and Fixing Go and Shell Files"

mapfile -t shell_files < <(find scripts -type f -name '*.sh' -print)

info "Formatting shell scripts..."
shfmt -w "${shell_files[@]}"
success "Shell formatting completed!"

info "Running go fix..."
go fix ./...
success "go fix passed!"

info "Running golangci-lint with fixes..."
go tool golangci-lint run --fix --timeout=30m0s ./...

success "Linting and fixing completed successfully!"
