#!/bin/bash

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

trap 'error "command failed line=$LINENO exit=$?"' ERR

if [[ "$#" -ne 1 || -z "$1" ]]; then
	error "Service name is required"
	printf 'Usage: %s <service-name>\n' "$0" >&2
	printf 'Example: %s myservice\n' "$0" >&2
	exit 1
fi

readonly SERVICE_NAME="$1"

if [[ ! "$SERVICE_NAME" =~ ^[a-z][a-z0-9-]*$ ||
	"$SERVICE_NAME" == *--* ||
	"$SERVICE_NAME" == *- ]]; then
	error "Service name must use lowercase letters, digits, and single hyphens"
	exit 1
fi

readonly SERVICE_DIR="internal/pkg/services/$SERVICE_NAME"

if [[ ! -d "$SERVICE_DIR" ]]; then
	error "Service '$SERVICE_NAME' does not exist at $SERVICE_DIR"
	exit 1
fi

section "Removing Service"
warning "Removing service '$SERVICE_NAME'..."
rm -rf "$SERVICE_DIR"

success "Service '$SERVICE_NAME' removed from $SERVICE_DIR"

# Regenerate service registration
info "Regenerating service registration..."
bash "$SCRIPT_DIR/service_registration.sh"
