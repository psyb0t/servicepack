#!/bin/bash

set -e

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

APP_NAME=$(head -n 1 go.mod | awk '{print $2}' | awk -F'/' '{print $NF}')

section "Building Development Docker Image"
info "Building development Docker image: $APP_NAME-dev..."

docker build -f Dockerfile.dev -t "$APP_NAME-dev" .

success "Development Docker image built successfully: $APP_NAME-dev"