#!/bin/bash

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

section "Security Scan (govulncheck + semgrep)"

require_cmds go semgrep jq

SARIF_OUT="${SARIF_OUT:-sec.sarif}"
tmp="$(mktemp -d)"
trap 'rm -r "$tmp"' EXIT

info "Running govulncheck..."
# SARIF mode exits 0 even with findings, so it only produces the report; the text
# run below is what gates, exiting non-zero on a reachable vulnerability.
# go tool keeps the repo's own pinned govulncheck version.
go tool govulncheck -format sarif ./... >"$tmp/govulncheck.sarif"
gv_rc=0
go tool govulncheck ./... || gv_rc=$?

info "Running semgrep (p/golang + p/security-audit)..."
# --error gates on a finding; --sarif writes the report; --metrics=off keeps the
# scan from phoning home. One run does both.
sg_rc=0
semgrep scan --config p/golang --config p/security-audit \
	--sarif --output "$tmp/semgrep.sarif" --metrics=off --error . || sg_rc=$?

info "Merging SARIF into $SARIF_OUT..."
jq -s '{version:"2.1.0","$schema":"https://json.schemastore.org/sarif-2.1.0.json",runs:((.[0].runs // []) + (.[1].runs // []))}' \
	"$tmp/govulncheck.sarif" "$tmp/semgrep.sarif" >"$SARIF_OUT"

if [ "$gv_rc" -ne 0 ] || [ "$sg_rc" -ne 0 ]; then
	error "Security findings (govulncheck=$gv_rc, semgrep=$sg_rc). See $SARIF_OUT"
	exit 1
fi

success "No security findings."
