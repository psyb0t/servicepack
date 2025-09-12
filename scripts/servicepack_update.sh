#!/bin/bash

set -e

echo "Checking servicepack framework updates..."

# Check if we're in a git repo
if [ ! -d ".git" ]; then
    echo "Warning: Not in a git repository. Make sure to commit your changes first."
else
    # Check for uncommitted changes
    if [ -n "$(git status --porcelain)" ]; then
        echo "Error: You have uncommitted changes in your repository."
        echo "Please commit or stash your changes before updating the framework."
        echo ""
        echo "Current uncommitted changes:"
        git status --short
        exit 1
    fi
fi

# Get the current version SHA if it exists
CURRENT_VERSION=""
if [ -f "servicepack.version" ]; then
    CURRENT_VERSION=$(cat servicepack.version)
    echo "Current servicepack version: $CURRENT_VERSION"
fi

# Get the latest version (tag or commit SHA from main branch)
echo "Fetching latest servicepack version..."

# Check if there are any version tags
LATEST_TAG=$(git ls-remote --tags --sort=version:refname https://github.com/psyb0t/servicepack | grep -v '\^{}$' | tail -n1 | cut -f2 | sed 's|refs/tags/||')

if [ -n "$LATEST_TAG" ]; then
    LATEST_VERSION="$LATEST_TAG"
    echo "Latest servicepack version (tag): $LATEST_VERSION"
else
    # Fallback to commit SHA from main branch
    LATEST_VERSION=$(git ls-remote https://github.com/psyb0t/servicepack HEAD | cut -f1)
    echo "Latest servicepack version (commit): $LATEST_VERSION"
fi

# Check if update is needed
if [ "$CURRENT_VERSION" = "$LATEST_VERSION" ]; then
    echo "Servicepack is already up to date!"
    exit 0
fi

echo "Update available! Creating backup before proceeding..."

# Create backup before updating
if ! make backup; then
    echo "Error: Failed to create backup. Update cancelled."
    exit 1
fi
echo ""

echo "Proceeding with framework update..."

# Create temp directory
TEMP_DIR=$(mktemp -d)
echo "Downloading latest servicepack..."

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
    echo "Preserving user module name: $USER_MODULE"
fi

echo "Updating framework core files..."

# Update core framework files, but preserve user services
rsync -av \
    --exclude='internal/pkg/services/*' \
    --exclude='.git*' \
    --exclude='build/' \
    --exclude='coverage.txt' \
    --exclude='vendor/' \
    "$TEMP_DIR/" ./

# If user had a custom module name, restore it
if [ -n "$USER_MODULE" ]; then
    echo "Restoring user module name in go.mod..."
    sed -i "1s|.*|module $USER_MODULE|" go.mod
fi

# Save the new version
printf "%s" "$LATEST_VERSION" > servicepack.version
echo "Updated servicepack.version to: $LATEST_VERSION"

# Clean up
rm -rf "$TEMP_DIR"

echo "Framework updated successfully! Hopefully it didn't mess up your shit!"
echo ""
echo "Next steps:"
echo "1. Run 'make dep' to update dependencies"
echo "2. Run 'make service-registration' to regenerate service registration"
echo "3. Run 'make test' to make sure everything still works"
echo "4. Check the diff and commit your changes"
echo ""
echo "If you notice fucked up stuff, run 'make backup-restore' to go back to your backup."
