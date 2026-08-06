#!/bin/bash

set -e

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

APP_NAME=$(head -n 1 go.mod | awk '{print $2}' | awk -F'/' '{print $NF}')

# Pinned by digest, not by tag. This is the image `make build` actually uses --
# the Dockerfiles are a separate path (`make docker-build`), so pinning them
# alone leaves the default build consuming a mutable tag. Bump deliberately.
readonly GO_BUILD_IMAGE="golang:1.26-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2"

section "Building Application"
info "Building $APP_NAME binary using Docker..."

# Create build directory
mkdir -p ./build

# Build using Docker
docker run --rm \
    -v "$(pwd)":/app \
    -w /app \
    -e USER_UID="$(id -u)" \
    -e USER_GID="$(id -g)" \
    "$GO_BUILD_IMAGE" \
    sh -c "apk add --no-cache gcc musl-dev && \
        CGO_ENABLED=0 go build -a \
        -ldflags '-extldflags \"-static\" -X main.appName=$APP_NAME' \
        -o ./build/$APP_NAME ./cmd/... && \
        chown \$USER_UID:\$USER_GID ./build/$APP_NAME"

success "Binary built successfully: ./build/$APP_NAME"
