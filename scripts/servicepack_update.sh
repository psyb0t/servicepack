#!/bin/bash

set -e

echo "Checking servicepack framework updates..."

# Check if we're in a git repo
if [ ! -d ".git" ]; then
    echo "Error: Not in a git repository. Framework updates require git."
    exit 1
fi

# Check for uncommitted changes
if [ -n "$(git status --porcelain)" ]; then
    echo "Error: You have uncommitted changes in your repository."
    echo "Please commit or stash your changes before updating the framework."
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

# Create update branch
UPDATE_BRANCH="servicepack_update_to_${LATEST_VERSION}"
echo "Creating update branch: $UPDATE_BRANCH"

if git show-ref --verify --quiet "refs/heads/$UPDATE_BRANCH"; then
    echo "Error: Branch $UPDATE_BRANCH already exists."
    echo "Delete it first with: git branch -D $UPDATE_BRANCH"
    exit 1
fi

git checkout -b "$UPDATE_BRANCH"

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

# Build exclude args - default excludes (protect user services only)
EXCLUDE_ARGS="--exclude=internal/pkg/services/* --exclude=.git* --exclude=build/ --exclude=coverage.txt --exclude=vendor/"

# Add excludes from .servicepackupdateignore if it exists
if [ -f ".servicepackupdateignore" ]; then
    echo "Using .servicepackupdateignore exclusions..."
    while IFS= read -r line; do
        # Skip comments (lines starting with #) and empty lines
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ -z "${line// }" ]] && continue
        EXCLUDE_ARGS="$EXCLUDE_ARGS --exclude=$line"
    done < .servicepackupdateignore
fi

# Update core framework files with exclusions
eval "rsync -av $EXCLUDE_ARGS \"$TEMP_DIR/\" ./"

# If user had a custom module name, restore it
if [ -n "$USER_MODULE" ]; then
    echo "Restoring user module name in go.mod..."
    sed -i "1s|.*|module $USER_MODULE|" go.mod
fi

# Save the new version
printf "%s" "$LATEST_VERSION" > servicepack.version
echo "Updated servicepack.version to: $LATEST_VERSION"

# Update dependencies
echo "Updating dependencies..."
make dep

# Commit changes to update branch
git add -A
git commit -m "Update servicepack framework to $LATEST_VERSION

- Updated from: $CURRENT_VERSION
- Updated to: $LATEST_VERSION
- Branch: $UPDATE_BRANCH

To review changes:
  git diff $CURRENT_BRANCH..HEAD

To apply update:
  git checkout $CURRENT_BRANCH
  git merge $UPDATE_BRANCH

To revert:
  git checkout $CURRENT_BRANCH
  git branch -D $UPDATE_BRANCH"

# Clean up
rm -rf "$TEMP_DIR"

echo ""
echo "Framework updated successfully in branch '$UPDATE_BRANCH'!"
echo "You are now on the update branch to review changes."
echo ""
echo "Next steps:"
echo "1. Review changes: git diff $CURRENT_BRANCH..HEAD"
echo "2. Test the update: make dep && make service-registration && make test"
echo "3. If satisfied: git checkout $CURRENT_BRANCH && git merge $UPDATE_BRANCH"
echo "4. If not satisfied, revert: git checkout $CURRENT_BRANCH && git branch -D $UPDATE_BRANCH"
echo ""
echo "Note: A backup was created before updating. Use 'make backup-restore' if needed."
