#!/bin/bash

# Post-update script - runs after rsync to ensure latest framework logic
# This script itself gets updated with each framework update

set -e

# These variables are passed from the main update script
CURRENT_BRANCH="$1"
UPDATE_BRANCH="$2"
CURRENT_VERSION="$3"
LATEST_VERSION="$4"
USER_MODULE="$5"
TEMP_DIR="$6"
SYNCED_FILES="$7"

# Source common functions (from the newly updated framework)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

section "Restoring User Configuration"

# If user had a custom module name, restore it everywhere
if [ -n "$USER_MODULE" ]; then
    info "Restoring user module name in go.mod..."
    sed -i "1s|.*|module $USER_MODULE|" go.mod

    # Rewrite the framework's import path to yours -- but ONLY in the .go files
    # this sync actually delivered, listed by rsync itself in $SYNCED_FILES.
    #
    # This used to be `find . -name '*.go' -not -path './vendor/*'`, which walks
    # the WHOLE working tree. `find` knows nothing about .gitignore, so it also
    # descended into scratch dirs, research checkouts and nested clones. In a real
    # project that meant 62k files rewritten instead of ~50, and it silently
    # rewrote the imports of an unrelated servicepack checkout that happened to
    # live under the repo -- invisible in `git status`, because the wreckage was
    # all gitignored.
    #
    # go.mod needs no walk of its own either: rsync never copies it (see
    # do_update.sh), its module line is rewritten directly above, its requires are
    # merged below, and `make dep` tidies and vendors the rest. The old
    # `-name '*.mod'` find had no legitimate target at all.
    FRAMEWORK_MODULE=$(head -n 1 "$TEMP_DIR/go.mod" | awk '{print $2}')

    # Fail loudly rather than falling back to the old tree-wide find: a silent
    # skip would leave the synced framework importing servicepack's own path and
    # nothing would build, with no clue why.
    if [ ! -f "$SYNCED_FILES" ]; then
        error "No sync manifest at '$SYNCED_FILES'; cannot rewrite module paths."
        exit 1
    fi

    info "Replacing module references in synced framework files..."

    rewritten=0
    while IFS= read -r synced_file; do
        [ -f "$synced_file" ] || continue
        sed -i "s|$FRAMEWORK_MODULE|$USER_MODULE|g" "$synced_file"
        rewritten=$((rewritten + 1))
    done < <(grep '\.go$' "$SYNCED_FILES" || true)  # no .go in the sync is valid

    info "Rewrote module references in $rewritten synced .go file(s)"

    # Belt and braces: nothing the repo actually owns may still import the
    # framework path. `git ls-files -co --exclude-standard` is tracked plus
    # untracked-but-not-ignored -- i.e. the real project, deliberately excluding
    # the gitignored scratch the old `find` used to trash.
    if [ "$FRAMEWORK_MODULE" != "$USER_MODULE" ]; then
        leftovers=$(git ls-files -co --exclude-standard -- '*.go' 2>/dev/null |
            xargs -r grep -l -- "$FRAMEWORK_MODULE" 2>/dev/null || true)

        if [ -n "$leftovers" ]; then
            warning "These files still reference $FRAMEWORK_MODULE:"
            while IFS= read -r leftover; do
                printf '  %s\n' "$leftover" >&2
            done <<< "$leftovers"
            warning "The sync manifest may be incomplete; check before merging."
        fi
    fi
fi

# Save the new version
printf "%s" "$LATEST_VERSION" > servicepack.version
success "Updated servicepack.version to: $LATEST_VERSION"

section "Merging Framework Dependencies (upgrade-only)"

# go.mod / go.sum are intentionally NOT copied by rsync (see do_update.sh):
# a wholesale replace drops every `require` for the downstream's own deps, and the
# following `go mod tidy` would re-resolve them DOWN to the lowest version MVS
# allows -- a silent, dangerous downgrade (go mod tidy only grabs *latest* for a
# dep referenced NOWHERE in the graph; a dep that survives only as a transitive
# requirement resolves to that low floor instead).
#
# Instead, merge the freshly-downloaded framework's direct requires + tool
# directives into the downstream's existing go.mod UPGRADE-ONLY: add what's
# missing, bump what the framework raised, and NEVER touch a dep the downstream
# already pins at an equal-or-higher version. `go mod tidy` then computes the MVS
# max over the union, so the framework's bumps land while the downstream's own
# (possibly-newer) deps stay intact.
merge_framework_deps() {
    local framework_gomod="$TEMP_DIR/go.mod"
    if [ ! -f "$framework_gomod" ] || [ ! -f "go.mod" ]; then
        warning "Missing go.mod (framework or local); skipping dependency merge."
        return 0
    fi

    # Highest of two semver strings ("" counts as lowest so a missing local dep
    # always takes the framework version).
    _semver_max() {
        if [ -z "$1" ]; then
            printf '%s' "$2"
            return
        fi
        printf '%s\n%s\n' "$1" "$2" | sort -V | tail -n1
    }

    # Direct (non-indirect) requires from the framework. Indirect deps are left
    # to `go mod tidy` -- pinning them here would fight MVS.
    local path version current desired
    while IFS=' ' read -r path version; do
        [ -z "$path" ] && continue
        current=$(go mod edit -json | \
            jq -r --arg p "$path" '(.Require // [])[] | select(.Path==$p) | .Version' | head -n1)
        desired=$(_semver_max "$current" "$version")

        if [ "$current" = "$desired" ]; then
            continue  # downstream already at >= framework version; leave it
        fi

        if [ -z "$current" ]; then
            info "add framework dep $path@$version"
        else
            info "bump framework dep $path $current -> $desired"
        fi
        go mod edit -require="${path}@${desired}"
    done < <(go mod edit -json "$framework_gomod" | \
        jq -r '(.Require // [])[] | select(.Indirect|not) | "\(.Path) \(.Version)"')

    # Tool directives the framework declares but the downstream is missing
    # (e.g. the linter / codegen tools). Adding a tool never downgrades anything.
    local tool
    while IFS= read -r tool; do
        [ -z "$tool" ] && continue
        if go mod edit -json | jq -e --arg t "$tool" '(.Tool // []) | any(.Path==$t)' >/dev/null; then
            continue
        fi
        info "add framework tool $tool"
        go mod edit -tool="$tool"
    done < <(go mod edit -json "$framework_gomod" | jq -r '(.Tool // [])[].Path')
}

merge_framework_deps

section "Updating Dependencies"
make dep

# Regenerate service registration after dependency updates
info "Regenerating service registration..."
make service-registration

section "Creating Post-Update Commands"

# Create temp directory and scripts
mkdir -p scripts/.post-update-temp

# Create review script
cat > scripts/.post-update-temp/review.sh << EOF
#!/bin/bash
echo "=== Reviewing Servicepack Update ==="
echo "Showing changes from $CURRENT_BRANCH to $UPDATE_BRANCH:"
echo ""
git diff $CURRENT_BRANCH..HEAD -- . ':!vendor'
echo ""
echo "=== Update Summary ==="
echo "Current branch: \$(git branch --show-current)"
echo "Changes ready for review. Use:"
echo "  make servicepack-update-merge   - to merge and finish update"
echo "  make servicepack-update-revert  - to cancel and revert"
EOF

# Create merge script
cat > scripts/.post-update-temp/merge.sh << EOF
#!/bin/bash
echo "=== Merging Servicepack Update ==="
git checkout $CURRENT_BRANCH
git merge $UPDATE_BRANCH
git branch -d $UPDATE_BRANCH
echo ""
echo "✅ Update merged successfully!"
echo "🧹 Cleaning up temp files..."
rm -rf scripts/.post-update-temp
echo "✅ Update complete!"
EOF

# Create revert script
cat > scripts/.post-update-temp/revert.sh << EOF
#!/bin/bash
echo "=== Reverting Servicepack Update ==="
git checkout $CURRENT_BRANCH
git branch -D $UPDATE_BRANCH
echo ""
echo "❌ Update reverted successfully!"
echo "🧹 Cleaning up temp files..."
rm -rf scripts/.post-update-temp
echo "💡 Backup is still available. Use 'make backup-restore' if needed."
EOF

# Make scripts executable
chmod +x scripts/.post-update-temp/*.sh

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
echo "Update branch '$UPDATE_BRANCH' created successfully!"
echo ""
echo "Use these commands to manage the update:"
echo ""
echo "1. Review changes:"
echo -e "   ${BLUE}make servicepack-update-review${NC}"
echo ""
echo "2. Test the update:"
echo -e "   ${BLUE}make dep && make service-registration && make test${NC}"
echo ""
echo "3. If satisfied, merge the update:"
echo -e "   ${BLUE}make servicepack-update-merge${NC}"
echo ""
echo "4. If not satisfied, revert:"
echo -e "   ${BLUE}make servicepack-update-revert${NC}"
echo ""
warning "A backup was created before updating. Use 'make backup-restore' if needed."