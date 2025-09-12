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

echo "Module name replacement completed!"
echo ""
echo "Contents of new go.mod:"
cat go.mod