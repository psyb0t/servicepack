#!/bin/bash

set -e

MODNAME="$1"

if [ -z "$MODNAME" ]; then
    echo "Error: Module name is required"
    echo "Usage: $0 <module-name>"
    echo "Example: $0 github.com/foo/bar"
    exit 1
fi

echo "Making this project your own with module name: $MODNAME"

# Get the current Go version from go.mod
REQUIRED_GO_VERSION=$(grep "^go " go.mod | awk '{print $2}')
echo "Required Go version from current go.mod: $REQUIRED_GO_VERSION"

# Get the user's Go version
USER_GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "Your Go version: $USER_GO_VERSION"

# Compare versions (simple string comparison works for Go versions)
if [ "$(printf '%s\n' "$REQUIRED_GO_VERSION" "$USER_GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_GO_VERSION" ]; then
    echo "ERROR: Your Go version ($USER_GO_VERSION) is less than the required version ($REQUIRED_GO_VERSION)"
    echo "Please upgrade to Go $REQUIRED_GO_VERSION or higher before proceeding."
    exit 1
fi

echo "Go version check passed!"

# Remove .git directory
if [ -d ".git" ]; then
    echo "Removing .git directory..."
    rm -rf .git
fi

# Get the old module name before removing go.mod
OLD_MODULE=$(grep "^module " go.mod | awk '{print $2}')
echo "Old module name: $OLD_MODULE"

# Remove go.sum
if [ -f "go.sum" ]; then
    echo "Removing go.sum..."
    rm -f go.sum
fi

# Remove go.mod
if [ -f "go.mod" ]; then
    echo "Removing go.mod..."
    rm -f go.mod
fi

# Create new go.mod with the provided module name and current Go version
echo "Creating new go.mod with module name: $MODNAME"
cat > go.mod << EOF
module $MODNAME

go $REQUIRED_GO_VERSION
EOF

echo "New go.mod created successfully!"

# Replace all occurrences of old module name with new module name in all files
echo "Replacing all references from '$OLD_MODULE' to '$MODNAME' in all files..."
find . -type f -name "*.go" -exec sed -i "s|$OLD_MODULE|$MODNAME|g" {} \;
find . -type f -name "*.mod" -exec sed -i "s|$OLD_MODULE|$MODNAME|g" {} \;
find . -type f -name "*.md" -exec sed -i "s|$OLD_MODULE|$MODNAME|g" {} \;

# Replace README.md with just the project name
PROJECT_NAME=$(echo "$MODNAME" | awk -F'/' '{print $NF}')
cat > README.md << EOF
# $PROJECT_NAME

---

*Built with spite using https://github.com/psyb0t/servicepack*
EOF

# Get current servicepack version and save it
if [ -f "servicepack.version" ]; then
    CURRENT_VERSION=$(cat servicepack.version)
    echo "Preserving servicepack version: $CURRENT_VERSION"
else
    # Get latest commit from servicepack repo to set as current version
    LATEST_VERSION=$(git ls-remote https://github.com/psyb0t/servicepack HEAD | cut -f1)
    printf "%s" "$LATEST_VERSION" > servicepack.version
    echo "Set servicepack version to: $LATEST_VERSION"
fi

echo "Module name replacement completed!"

# Install golangci-lint tool
echo "Installing golangci-lint tool..."
go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4

# Install gofindimpl
echo "Installing gofindimpl tool..."
go get -tool github.com/psyb0t/gofindimpl

# Update dependencies
echo "Updating dependencies..."
make dep

# Initialize git repository
echo "Initializing git repository..."
git init

# Rename branch to main
git branch -m main

# Add all files and create initial commit
echo "Creating initial commit..."
git add -A
git commit -m "Initial commit"

# Show commit log
git log
