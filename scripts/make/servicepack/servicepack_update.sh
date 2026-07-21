#!/bin/bash

# Servicepack update — bootstrap phase (thin + stable, keep it that way).
#
# This is a SELF-UPDATING updater: the script that performs the update is
# itself one of the files being updated. A running script cannot swap its own
# logic mid-run, so whatever policy lives HERE can never be fixed for the very
# update that installs the fix — the downstream would forever run one-version-
# old update logic. To escape that trap, this bootstrap does the bare minimum
# and then hands off to `do_update.sh` FROM THE FRESHLY DOWNLOADED framework,
# so the newest update policy always drives the sync.
#
# KEEP THIS FILE MINIMAL. It must only: verify preconditions, fetch the latest
# framework, decide whether an update is needed, and exec the fresh do_update.sh.
# It must contain NO rsync exclude list, NO dependency merge, NO file overwrites
# — all of that is POLICY and belongs in do_update.sh where it updates itself.
# The less this file does, the less it ever needs to change (and the changes it
# can't self-heal stay harmless).

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

REPO_ROOT="$(git rev-parse --show-toplevel)"

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

# Hand off to the update logic FROM THE FRESH DOWNLOAD, so the newest policy
# drives this update (not the possibly-stale copy installed in the downstream).
FRESH_DO_UPDATE="$TEMP_DIR/scripts/make/servicepack/do_update.sh"
if [ ! -f "$FRESH_DO_UPDATE" ]; then
    error "Downloaded framework is missing do_update.sh; cannot proceed."
    warning "Clean up: rm -rf $TEMP_DIR"
    exit 1
fi

section "Applying Update (latest logic)"
bash "$FRESH_DO_UPDATE" \
    "$REPO_ROOT" \
    "$TEMP_DIR" \
    "$CURRENT_VERSION" \
    "$LATEST_VERSION"
