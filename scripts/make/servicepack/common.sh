#!/bin/bash

# Common functions for servicepack scripts

# Colors for pretty output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Diagnostics go to STDERR, never stdout.
#
# Every script in this tree sources this file, so when these wrote to stdout any
# caller that captured a script's output — `VAR=$(some_script.sh)`, or piping it
# into another command — got the decorated banners mixed in with the data it
# actually wanted. stdout is the script's OUTPUT; stderr is where the running
# commentary belongs.
info() {
	echo -e "${BLUE}${BOLD}[INFO]${NC} $1" >&2
}

success() {
	echo -e "${GREEN}${BOLD}[SUCCESS]${NC} $1" >&2
}

warning() {
	echo -e "${YELLOW}${BOLD}[WARNING]${NC} $1" >&2
}

error() {
	echo -e "${RED}${BOLD}[ERROR]${NC} $1" >&2
}

section() {
	echo "" >&2
	echo -e "${BOLD}=== $1 ===${NC}" >&2
	echo "" >&2
}

# require_cmds — fail fast if any required external tool is missing, and list
# EVERY missing one at once so a single run surfaces them all. Call this BEFORE
# a script mutates any state (backups, branches, file writes). It is what keeps
# `servicepack-update` from stranding a repo on a half-made update branch
# because rsync / jq / etc. was only discovered missing partway through.
require_cmds() {
	local missing=()
	local cmd
	for cmd in "$@"; do
		command -v "$cmd" >/dev/null 2>&1 || missing+=("$cmd")
	done

	if [ "${#missing[@]}" -gt 0 ]; then
		error "Missing required tools: ${missing[*]}"
		warning "Install them and re-run — nothing has been changed yet."
		exit 1
	fi
}
