#!/bin/bash

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

MIN_TEST_COVERAGE=${MIN_TEST_COVERAGE:-90}

section "Running Tests with Coverage Check"

module="$(go list -m)"
info "module=$module coverage floor=${MIN_TEST_COVERAGE}%"

# Covdata drop point for a service that runs OUT OF PROCESS — a real container an
# integration test starts. The test process reads SERVICEPACK_COVDATA_DIR and
# mounts it into that container as GOCOVERDIR; we textfmt + merge it below. An
# in-process service needs none of this: its coverage comes straight from the
# -coverpkg run, because every package is instrumented across every test binary.
covdata_dir="${SERVICEPACK_COVDATA_DIR:-$PWD/.cover/covdata}"
export SERVICEPACK_COVDATA_DIR="$covdata_dir"
rm -rf "$covdata_dir"
mkdir -p "$covdata_dir"

profile_raw="coverage.txt"
profile_covdata="coverage_covdata.txt"
profile_merged="coverage_merged.txt"
profile_filtered="coverage_filtered.txt"

trap 'rm -f "$profile_raw" "$profile_covdata" "$profile_merged" "$profile_filtered"' EXIT

# merge_profiles unions coverage profiles, keeping the highest hit count per
# block, so a service covered both by a unit test and by its out-of-process
# integration run is credited for the union of both.
merge_profiles() {
	local out="$1"
	shift

	{
		printf 'mode: atomic\n'
		awk '
			FNR == 1 && $1 == "mode:" { next }
			{
				key = $1 SUBSEP $2
				if (!(key in count) || $3 > count[key]) {
					count[key] = $3
				}
				block[key] = $1 " " $2
			}
			END {
				for (key in block) {
					print block[key], count[key]
				}
			}
		' "$@" | sort -k1,1 -k2,2
	} >"$out"
}

# EVERY package is instrumented (-coverpkg=<module>/...) and integration-tagged
# tests run too, so a test under tests/ credits coverage to the service package
# it drives. Nothing is hand-selected, so a new package or service can never
# silently escape the gate.
info "running unit + integration tests with coverage..."
if ! go test -race -count=1 -timeout=600s -tags=integration \
	-coverpkg="$module/..." -coverprofile="$profile_raw" ./...; then
	error "Tests failed"
	exit 1
fi

# Fold in any out-of-process covdata the integration run produced.
if find "$covdata_dir" -type f -name 'covcounters.*' -print -quit | grep -q .; then
	info "merging out-of-process service covdata..."
	go tool covdata textfmt -i="$covdata_dir" -o="$profile_covdata"
	merge_profiles "$profile_merged" "$profile_raw" "$profile_covdata"
else
	cp "$profile_raw" "$profile_merged"
fi

# Drop from the gate only what is not a real project's hand-written code under
# test:
#   - cmd mains (thin wiring),
#   - test infrastructure under tests/ (the harness itself runs, so it credits
#     coverage to the production packages it drives, but its own setup/teardown
#     is not code under test),
#   - generated code (*.gen.go),
#   - the framework's service-manager mocks,
#   - the framework's own demo services (example-* and hello-world), which ship
#     in the template purely as illustrations and are deleted by every real
#     project — so their absence of tests must not drag a downstream's floor,
#     and a real service (any other name) is always counted.
# Everything else — a project's real services included — counts toward the floor.
gate_exclude='(/cmd/|/tests/|\.gen\.go:|/internal/pkg/service-manager/mocks\.go:|/internal/pkg/services/(example-|hello-world/))'
awk -v exclude="$gate_exclude" '$1 !~ exclude' \
	"$profile_merged" >"$profile_filtered"

coverage_summary=$(go tool cover -func="$profile_filtered" | awk '$1 == "total:" { print $3 }')
pct=${coverage_summary%%%}
integer_pct=${pct%%.*}

# Persist the decimal percentage for the badge pipeline (survives the trap above).
printf '%s\n' "$pct" >coverage-percent.txt

if [ -z "$pct" ]; then
	warning "No test coverage information available"

	exit 1
elif [ "$integer_pct" -lt "$MIN_TEST_COVERAGE" ]; then
	error "Coverage $pct% is less than the minimum $MIN_TEST_COVERAGE%"
	exit 1
else
	success "Coverage $pct% meets the minimum requirement of $MIN_TEST_COVERAGE%"
fi
