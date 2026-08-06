#!/bin/bash

# Servicepack update — apply phase.
#
# This script carries ALL the update POLICY (rsync exclude list, dependency
# merge, branch/commit dance) and is ALWAYS executed from the freshly
# downloaded framework copy, handed off to by the thin bootstrap
# servicepack_update.sh. A running updater cannot swap its own logic mid-run,
# so the policy lives here and runs from the download — the newest policy
# always drives the sync, on the very first update that ships it. Never move
# policy back into the bootstrap.
#
# Args (all passed by servicepack_update.sh):
#   $1 REPO_ROOT        - the downstream project's checkout (the repo to update)
#   $2 TEMP_DIR         - the freshly cloned latest framework
#   $3 CURRENT_VERSION  - the downstream's servicepack.version before this update
#   $4 LATEST_VERSION   - the framework version being installed

set -e

REPO_ROOT="$1"
TEMP_DIR="$2"
CURRENT_VERSION="$3"
LATEST_VERSION="$4"

# common.sh from the freshly downloaded framework (this script's own dir), so
# even the helper functions are the latest.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# All git + file operations run against the downstream project.
cd "$REPO_ROOT"

CURRENT_BRANCH=$(git branch --show-current)

section "Preparing Update"
warning "Update available! Creating backup before proceeding..."

# Create backup before updating
if ! make backup; then
    error "Failed to create backup. Update cancelled."
    exit 1
fi

section "Creating Update Branch"

UPDATE_BRANCH="servicepack_update_to_${LATEST_VERSION}"
info "Creating update branch: $UPDATE_BRANCH"

if git show-ref --verify --quiet "refs/heads/$UPDATE_BRANCH"; then
    error "Branch $UPDATE_BRANCH already exists."
    warning "Delete it first with: git branch -D $UPDATE_BRANCH"
    exit 1
fi

git checkout -b "$UPDATE_BRANCH"

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
#
# The list above is only the mechanical floor (git internals, the dependency
# merge, per-project prose). The opt-outs that nearly every project wants ship
# in the .servicepackupdateignore this repo hands you -- the framework's own
# publication workflows, agent skill, funding links, secret-scanning allowlist
# and docker context. Those are the framework's furniture rather than a
# baseline you customize, so they start ignored instead of leaving each project
# to find out the hard way. Read that file, not this list, when you want to
# know what your project is keeping for itself.
#
# Note that file is excluded from the sync (above), so it is yours from the
# moment you scaffold: a framework update can never rewrite your opt-outs, and
# equally never adds new default ones to an existing project. When a release
# adds an entry, existing downstreams have to copy it over by hand.
# An ARRAY, not a string, and passed to rsync without `eval`.
#
# This used to be one space-joined string expanded through `eval`, with the
# downstream's own .servicepackupdateignore lines concatenated onto it below.
# eval re-parses that whole command line, so any shell metacharacter in that
# file — a backtick, a $(...), a ; — executed during `make servicepack-update`.
# An array member is passed to rsync as a single literal argument, so the
# contents of that file can no longer be anything but a pattern.
#
# The `*` in the first entry is deliberate and must stay unexpanded: rsync does
# its own pattern matching, and quoting is what guarantees the shell keeps its
# hands off it.
EXCLUDE_ARGS=(
    "--exclude=internal/pkg/services/*"
    "--exclude=README.md"
    "--exclude=LICENSE"
    "--exclude=CHANGELOG.md"
    "--exclude=.git"
    "--exclude=.gitignore"
    "--exclude=.servicepackupdateignore"
    "--exclude=Makefile"
    "--exclude=Dockerfile"
    "--exclude=Dockerfile.dev"
    "--exclude=build/"
    "--exclude=coverage.txt"
    "--exclude=vendor/"
    "--exclude=go.mod"
    "--exclude=go.sum"
)

# Add excludes from .servicepackupdateignore if it exists
if [ -f ".servicepackupdateignore" ]; then
    info "Using .servicepackupdateignore exclusions..."
    while IFS= read -r line; do
        # Skip comments (lines starting with #) and empty lines
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ -z "${line// }" ]] && continue
        EXCLUDE_ARGS+=("--exclude=$line")
    done < .servicepackupdateignore
fi

# Update core framework files with exclusions.
#
# Capture rsync's own transfer list as it runs. _post_update.sh has to rewrite the
# framework's module path to yours, and this manifest is the exact set of files to
# do that to: every exclude above and every .servicepackupdateignore entry has
# already been applied to it. Deriving the set any other way -- notably a `find`
# over the working tree -- reaches files this update never delivered, including
# everything in gitignored scratch directories.
SYNCED_FILES=$(mktemp)
trap 'rm -f "$SYNCED_FILES"' EXIT

set -o pipefail
rsync -av --out-format='%n' "${EXCLUDE_ARGS[@]}" "$TEMP_DIR/" ./ | tee "$SYNCED_FILES"
set +o pipefail

section "Running Post-Update Script"
info "Executing post-update logic with latest framework code..."

# Call the post-update script FROM THE FRESH DOWNLOAD (this script's dir), so
# the merge/commit logic is the latest too — not whatever the downstream had
# before this update.
bash "$SCRIPT_DIR/_post_update.sh" \
    "$CURRENT_BRANCH" \
    "$UPDATE_BRANCH" \
    "$CURRENT_VERSION" \
    "$LATEST_VERSION" \
    "$USER_MODULE" \
    "$TEMP_DIR" \
    "$SYNCED_FILES"
