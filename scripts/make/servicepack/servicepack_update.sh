#!/bin/bash

set -e

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

info "Checking servicepack framework updates..."

# Check if we're in a git repo
if [ ! -d ".git" ]; then
    error "Not in a git repository. Framework updates require git."
    exit 1
fi

# Check for uncommitted changes
if [ -n "$(git status --porcelain)" ]; then
    error "You have uncommitted changes in your repository."
    warning "Please commit or stash your changes before updating the framework."
    echo ""
    echo "Current uncommitted changes:"
    git status --short
    exit 1
fi

# Get current branch name
CURRENT_BRANCH=$(git branch --show-current)

# Get the current version SHA if it exists
CURRENT_VERSION=""
if [ -f "servicepack.version" ]; then
    CURRENT_VERSION=$(cat servicepack.version)
    info "Current servicepack version: $CURRENT_VERSION"
fi

section "Fetching Latest Version"

# Check if there are any version tags
LATEST_TAG=$(git ls-remote --tags --sort=version:refname https://github.com/psyb0t/servicepack | grep -v '\^{}$' | tail -n1 | cut -f2 | sed 's|refs/tags/||')

if [ -n "$LATEST_TAG" ]; then
    LATEST_VERSION="$LATEST_TAG"
    info "Latest servicepack version (tag): $LATEST_VERSION"
else
    # Fallback to commit SHA from main branch
    LATEST_VERSION=$(git ls-remote https://github.com/psyb0t/servicepack HEAD | cut -f1)
    info "Latest servicepack version (commit): $LATEST_VERSION"
fi

# Check if update is needed
if [ "$CURRENT_VERSION" = "$LATEST_VERSION" ]; then
    success "Servicepack is already up to date!"
    exit 0
fi

section "Preparing Update"
warning "Update available! Creating backup before proceeding..."

# Create backup before updating
if ! make backup; then
    error "Failed to create backup. Update cancelled."
    exit 1
fi

section "Creating Update Branch"

# Create update branch
UPDATE_BRANCH="servicepack_update_to_${LATEST_VERSION}"
info "Creating update branch: $UPDATE_BRANCH"

if git show-ref --verify --quiet "refs/heads/$UPDATE_BRANCH"; then
    error "Branch $UPDATE_BRANCH already exists."
    warning "Delete it first with: git branch -D $UPDATE_BRANCH"
    exit 1
fi

git checkout -b "$UPDATE_BRANCH"

section "Downloading Framework"

# Create temp directory
TEMP_DIR=$(mktemp -d)
info "Downloading latest servicepack to $TEMP_DIR..."

# Clone the latest servicepack
if [ -n "$LATEST_TAG" ]; then
    # Clone specific tag
    git clone --branch "$LATEST_TAG" --depth 1 https://github.com/psyb0t/servicepack "$TEMP_DIR"
else
    # Clone latest commit from main branch
    git clone --depth 1 https://github.com/psyb0t/servicepack "$TEMP_DIR"
fi

# Backup user's go.mod module name
USER_MODULE=""
if [ -f "go.mod" ]; then
    USER_MODULE=$(head -n 1 go.mod | awk '{print $2}')
    info "Preserving user module name: $USER_MODULE"
fi

section "Updating Framework Files"

# Build exclude args - default excludes (protect user content).
#
# CHANGELOG.md is per-project (documents the DOWNSTREAM app's releases, not
# servicepack's) so the framework never owns it.
#
# go.mod / go.sum are NOT copied by rsync because rsync has no merge semantics:
# a wholesale replace would drop every `require` line for the downstream's own
# deps, and the following `go mod tidy` would then re-resolve them DOWN to the
# lowest version MVS allows (a silent, dangerous downgrade). Instead the
# framework's dependency bumps are merged into the downstream's existing go.mod
# UPGRADE-ONLY in _post_update.sh (see merge_framework_deps). This keeps the
# downstream's own, possibly-newer deps intact while still pulling servicepack's
# framework-dep floors forward so the freshly-synced framework code compiles.
#
# Everything else the framework owns and overwrites (incl. .golangci.yml). A
# downstream that has genuinely customized a framework-owned file lists it in
# ITS OWN .servicepackupdateignore -- that is the intended opt-out, not a
# blanket exclude baked into the framework.
EXCLUDE_ARGS="--exclude=internal/pkg/services/* \
    --exclude=README.md \
    --exclude=LICENSE \
    --exclude=CHANGELOG.md \
    --exclude=.git \
    --exclude=.gitignore \
    --exclude=.servicepackupdateignore \
    --exclude=Makefile \
    --exclude=Dockerfile \
    --exclude=Dockerfile.dev \
    --exclude=build/ \
    --exclude=coverage.txt \
    --exclude=vendor/ \
    --exclude=go.mod \
    --exclude=go.sum"

# Add excludes from .servicepackupdateignore if it exists
if [ -f ".servicepackupdateignore" ]; then
    info "Using .servicepackupdateignore exclusions..."
    while IFS= read -r line; do
        # Skip comments (lines starting with #) and empty lines
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ -z "${line// }" ]] && continue
        EXCLUDE_ARGS="$EXCLUDE_ARGS --exclude=$line"
    done < .servicepackupdateignore
fi

# Update core framework files with exclusions
eval "rsync -av $EXCLUDE_ARGS \"$TEMP_DIR/\" ./"

section "Running Post-Update Script"
info "Executing post-update logic with latest framework code..."

# Call the post-update script with all necessary parameters
bash scripts/make/servicepack/_post_update.sh \
    "$CURRENT_BRANCH" \
    "$UPDATE_BRANCH" \
    "$CURRENT_VERSION" \
    "$LATEST_VERSION" \
    "$USER_MODULE" \
    "$TEMP_DIR"
