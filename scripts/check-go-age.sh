#!/bin/bash

set -euo pipefail

readonly MINIMUM_AGE_DAYS=7
readonly FIRST_PARTY_PREFIX="github.com/psyb0t/"
readonly LOG_FILE="${LOG_FILE:-/tmp/check-go-age.log}"

json_escape() {
	local value="$1"

	value="${value//\\/\\\\}"
	value="${value//\"/\\\"}"
	value="${value//$'\n'/\\n}"
	value="${value//$'\r'/\\r}"
	value="${value//$'\t'/\\t}"

	printf '%s' "$value"
}

log() {
	local level="$1"
	shift

	local escaped_message timestamp
	escaped_message="$(json_escape "$*")"
	timestamp="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

	printf '{"time":"%s","level":"%s","file":"%s","line":%d,"func":"%s","msg":"%s"}\n' \
		"$timestamp" \
		"$level" \
		"${BASH_SOURCE[1]##*/}" \
		"${BASH_LINENO[0]}" \
		"${FUNCNAME[1]:-main}" \
		"$escaped_message" >&2
}

fail() {
	log ERROR "$*"
	exit 1
}

on_error() {
	local status="$?"

	log ERROR "command failed exit=$status"
	exit "$status"
}

usage() {
	printf 'usage: %s module@version\n' "${0##*/}" >&2
}

trap on_error ERR
exec > >(tee -a "$LOG_FILE") 2>&1

main() {
	local module module_version normalized_published_at now_epoch published_at published_epoch version

	if [[ "$#" -ne 1 ]]; then
		usage
		exit 2
	fi

	module_version="$1"
	module="${module_version%@*}"
	version="${module_version##*@}"

	if [[ "$module" == "$module_version" || -z "$version" ]]; then
		usage
		fail "expected module@version"
	fi

	if [[ "$module" == "$FIRST_PARTY_PREFIX"* ]]; then
		log INFO "first-party module skips publication-age gate module=$module version=$version"

		return
	fi

	published_at="$(GONOSUMDB='' GOPROXY=https://proxy.golang.org,direct GOFLAGS=-mod=mod \
		go list -m -json "${module}@${version}" | jq -r '.Time // empty')"
	[[ -n "$published_at" ]] || fail "module publication time unavailable module=$module version=$version"

	normalized_published_at="${published_at%Z}"
	normalized_published_at="${normalized_published_at/T/ }"
	published_epoch="$(date -u -d "$normalized_published_at" '+%s')"
	now_epoch="$(date -u '+%s')"

	if (((now_epoch - published_epoch) < MINIMUM_AGE_DAYS * 86400)); then
		fail "module is newer than the supply-chain age gate module=$module version=$version"
	fi

	log INFO "module publication-age gate passed module=$module version=$version"
}

main "$@"
