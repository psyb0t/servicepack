#!/bin/bash

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

section "Linting Go and Shell Files"

mapfile -t shell_files < <(find scripts -type f -name '*.sh' -print)

info "Checking shell formatting..."
shfmt -d "${shell_files[@]}"
success "Shell formatting passed!"

info "Running shellcheck..."
shellcheck -x -P "$SCRIPT_DIR" "${shell_files[@]}"
success "shellcheck passed!"

info "Running go fix (diff only)..."
out=$(go fix -diff ./... 2>&1) || true
if [ -n "$out" ]; then
	printf '%s\n' "$out" >&2
	error "go fix found issues. Run 'make lint-fix' to apply."
	exit 1
fi
success "go fix passed!"

info "Running golangci-lint..."
go tool golangci-lint run --timeout=30m0s ./...

success "Linting completed successfully!"
