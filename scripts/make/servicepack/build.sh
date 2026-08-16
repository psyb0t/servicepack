#!/bin/bash

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

APP_NAME="$(head -n 1 go.mod | awk '{print $2}' | awk -F'/' '{print $NF}')"
readonly APP_NAME
BUILD_COMMIT="$(git rev-parse --verify HEAD 2>/dev/null || true)"
readonly BUILD_COMMIT
BUILD_VERSION="$(git describe --tags --exact-match HEAD 2>/dev/null || printf 'dev')"
readonly BUILD_VERSION

# Pinned by digest, not by tag. This is the image `make build` actually uses --
# the Dockerfiles are a separate path (`make docker-build`), so pinning them
# alone leaves the default build consuming a mutable tag. Bump deliberately.
readonly GO_BUILD_IMAGE="golang:1.26.4-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648"

section "Building Application"
info "Building $APP_NAME binary using Docker..."
info "Build commit: ${BUILD_COMMIT:-unavailable}"
info "Build version: $BUILD_VERSION"

# Create build directory
mkdir -p ./build

# Build using Docker
docker run --rm \
	-v "$(pwd)":/app \
	-w /app \
	-e USER_UID="$(id -u)" \
	-e USER_GID="$(id -g)" \
	-e "APP_NAME=$APP_NAME" \
	-e "BUILD_COMMIT=$BUILD_COMMIT" \
	-e "BUILD_VERSION=$BUILD_VERSION" \
	"$GO_BUILD_IMAGE" \
	sh -ceu '
        apk add --no-cache gcc musl-dev && \
        CGO_ENABLED=0 go build -a \
        -ldflags "-X main.appName=${APP_NAME} -X main.buildCommit=${BUILD_COMMIT} -X main.buildVersion=${BUILD_VERSION}" \
        -o "./build/${APP_NAME}" ./cmd && \
        chown "${USER_UID}:${USER_GID}" "./build/${APP_NAME}"
    '

success "Binary built successfully: ./build/$APP_NAME"
