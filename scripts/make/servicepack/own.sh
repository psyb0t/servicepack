#!/bin/bash

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

MODNAME="${1:-}"
readonly FRAMEWORK_GO_MOD=".servicepack-framework.go.mod"
readonly FRAMEWORK_GO_SUM=".servicepack-framework.go.sum"

cleanup_framework_go_mod() {
	rm -f "$FRAMEWORK_GO_MOD"
	rm -f "$FRAMEWORK_GO_SUM"
}

if [ -z "$MODNAME" ]; then
	error "Module name is required"
	echo "Usage: $0 <module-name>"
	echo "Example: $0 github.com/foo/bar"
	exit 1
fi

section "Making Project Your Own"
info "Module name: $MODNAME"

section "Go Version Check"

# Get the current Go version from go.mod
REQUIRED_GO_VERSION=$(grep "^go " go.mod | awk '{print $2}')
info "Required Go version from current go.mod: $REQUIRED_GO_VERSION"

info "The Servicepack Make targets run Go in the development image."

if [ -e "$FRAMEWORK_GO_MOD" ] || [ -e "$FRAMEWORK_GO_SUM" ]; then
	error "Refusing to overwrite an existing framework dependency manifest"
	exit 1
fi

# `make own` deliberately starts a new module, but it must not rediscover
# framework dependencies from the registry. Preserve the exact manifest first;
# this keeps the scaffold's tested module graph and tool directives intact.
cp go.mod "$FRAMEWORK_GO_MOD"
cp go.sum "$FRAMEWORK_GO_SUM"
trap cleanup_framework_go_mod EXIT

section "Cleaning Project"

# Remove example services (keep only hello-world)
for dir in internal/pkg/services/example-*; do
	if [ -d "$dir" ]; then
		info "Removing example service: $dir"
		rm -rf "$dir"
	fi
done

# Remove .git directory
if [ -d ".git" ]; then
	info "Removing .git directory..."
	rm -rf .git
fi

# Get the old module name before removing go.mod
OLD_MODULE=$(grep "^module " go.mod | awk '{print $2}')
info "Old module name: $OLD_MODULE"

# Remove go.sum
if [ -f "go.sum" ]; then
	info "Removing go.sum..."
	rm -f go.sum
fi

# Remove go.mod
if [ -f "go.mod" ]; then
	info "Removing go.mod..."
	rm -f go.mod
fi

# Remove vendor directory
if [ -d "vendor" ]; then
	info "Removing vendor directory..."
	rm -rf vendor
fi

section "Creating New Module"

# Retain every pinned framework dependency and tool directive while replacing
# only the module declaration. `make dep` below tidies the graph after example
# services have been removed.
info "Creating new go.mod with module name and framework pins: $MODNAME"
sed "s|^module $OLD_MODULE$|module $MODNAME|" \
	"$FRAMEWORK_GO_MOD" >go.mod
cp "$FRAMEWORK_GO_SUM" go.sum

success "New go.mod created successfully!"

section "Replacing Module References"

# Replace all occurrences of old module name with new module name in all files
info "Replacing all references from '$OLD_MODULE' to '$MODNAME' in all files..."
find . -type f -name "*.go" -exec sed -i "s|$OLD_MODULE|$MODNAME|g" {} \;
find . -type f -name "*.mod" -exec sed -i "s|$OLD_MODULE|$MODNAME|g" {} \;
find . -type f -name "*.md" -exec sed -i "s|$OLD_MODULE|$MODNAME|g" {} \;

# Replace README.md with just the project name
PROJECT_NAME=$(echo "$MODNAME" | awk -F'/' '{print $NF}')
info "Creating new README.md for project: $PROJECT_NAME"
cat >README.md <<EOF
# $PROJECT_NAME

---

*Built with spite using https://github.com/psyb0t/servicepack*
EOF

# Get current servicepack version and save it
if [ -f "servicepack.version" ]; then
	CURRENT_VERSION=$(cat servicepack.version)
	info "Preserving servicepack version: $CURRENT_VERSION"
else
	# Get latest commit from servicepack repo to set as current version
	LATEST_VERSION=$(git ls-remote https://github.com/psyb0t/servicepack HEAD | cut -f1)
	printf "%s" "$LATEST_VERSION" >servicepack.version
	info "Set servicepack version to: $LATEST_VERSION"
fi

success "Module name replacement completed!"

section "Regenerating Service Registration"
info "Regenerating service registration without examples..."
make service-registration

section "Downloading Framework Dependencies"
make dep

cleanup_framework_go_mod
trap - EXIT

section "Initializing Git Repository"

# Initialize git repository
info "Initializing git repository..."
git init

# Rename branch to main
info "Renaming branch to main..."
git branch -m main

# Add all files and create initial commit
info "Creating initial commit..."
git add -A
git commit -m "Initial commit"

success "Project setup completed!"
