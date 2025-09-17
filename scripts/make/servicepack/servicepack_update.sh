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

# Build exclude args - default excludes (protect user content)
EXCLUDE_ARGS="--exclude=internal/pkg/services/* \
    --exclude=README.md \
    --exclude=LICENSE \
    --exclude=.git \
    --exclude=.gitignore \
    --exclude=.servicepackupdateignore \
    --exclude=Makefile \
    --exclude=Dockerfile \
    --exclude=Dockerfile.dev \
    --exclude=build/ \
    --exclude=coverage.txt \
    --exclude=vendor/"

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

section "Restoring User Configuration"

# If user had a custom module name, restore it everywhere
if [ -n "$USER_MODULE" ]; then
    info "Restoring user module name in go.mod..."
    sed -i "1s|.*|module $USER_MODULE|" go.mod

    info "Replacing module references in all files..."
    # Get the original module name from the downloaded framework
    FRAMEWORK_MODULE=$(head -n 1 "$TEMP_DIR/go.mod" | awk '{print $2}')

    # Replace all references to framework module with user module
    find . -type f -name "*.go" -not -path "./vendor/*" -exec sed -i "s|$FRAMEWORK_MODULE|$USER_MODULE|g" {} \;
    find . -type f -name "*.mod" -not -path "./vendor/*" -exec sed -i "s|$FRAMEWORK_MODULE|$USER_MODULE|g" {} \;
fi

# Save the new version
printf "%s" "$LATEST_VERSION" > servicepack.version
success "Updated servicepack.version to: $LATEST_VERSION"

section "Updating Dependencies"
make dep

# Regenerate service registration after dependency updates
info "Regenerating service registration..."
make service-registration

# Commit changes to update branch
git add -A
git commit -m "Updated servicepack from $CURRENT_VERSION to $LATEST_VERSION"

section "Finalizing Update"

# Clean up
rm -rf "$TEMP_DIR"
info "Cleaned up temporary files"

success "Framework updated successfully in branch '$UPDATE_BRANCH'!"
info "You are now on the update branch to review changes."

section "Next Steps"
echo "1. Review changes:"
echo -e "   ${BLUE}git diff $CURRENT_BRANCH..HEAD${NC}"
echo ""
echo "2. Test the update:"
echo -e "   ${BLUE}make dep && make service-registration && make test${NC}"
echo ""
echo "3. If satisfied, merge the update:"
echo -e "   ${BLUE}git checkout $CURRENT_BRANCH && git merge $UPDATE_BRANCH${NC}"
echo ""
echo "4. If not satisfied, revert:"
echo -e "   ${BLUE}git checkout $CURRENT_BRANCH && git branch -D $UPDATE_BRANCH${NC}"
echo ""
warning "A backup was created before updating. Use 'make backup-restore' if needed."
