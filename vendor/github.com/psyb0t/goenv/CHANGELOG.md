# Changelog

All notable changes per release. Versions follow [semver](https://semver.org).

## v1.0.10 — 2026-08-08

Documentation. No code change.

- The README's surface summary said "three functions and three constants" and
  omitted the exported `Type`. It is four things, not six: `Get`, `IsDev`,
  `IsProd`; `EnvVarName`, `Prod`, `Dev`; and `type Type = string`, which is what
  the constants are typed as.

## v1.0.9 — 2026-08-08

Repository infrastructure only. No library code changed.

- Added the imported-by badge: a count of the public packages importing this
  module, linking to `importers.md` on the `badges` branch — the importing
  repositories, grouped, package counts descending, and flagged when the owner
  differs from this repo's.
- It measures **blast radius, not adoption**. The number tells you how much
  breaks when an exported name moves; the external mark tells you whether any of
  that is someone else's problem, which is what decides how strictly the module
  has to be versioned. Stars answer neither question — nobody stars an env-var
  reader, they just import it.
- Refreshed weekly rather than daily, because pkg.go.dev's crawl lags
  publication by days and each run drags the full test suite along (the badges
  job needs the coverage artifact). The whole pipeline runs rather than a
  badges-only job: the badge publisher republishes only what a run produced, so
  a badge-only refresh would delete the coverage, version and license badges.
- The cron slot is derived from a hash of the repository name rather than
  chosen — GitHub cron has no randomness, and its scheduler sheds queued runs
  hardest at the round times a human would pick.

## v1.0.8 — 2026-08-01

Infrastructure only. No library code changed — every commit since v1.0.7
touches `.github/workflows/`.

- The pipeline was split: building and publishing stay in `pipeline.yml`, and
  everything that leaves the host now lives beside it in `mirror-and-archive.yml`.
- The repo is mirrored to Codeberg as well as GitLab.
- It is archived to the Wayback Machine, Software Heritage and archive.org.
- Issues opened on either mirror are copied back to GitHub every six hours, and
  closed here when the original closes.
- Pull requests are switched off on the mirrors: they are force-pushed from
  GitHub, so anything merged there would be destroyed by the next sync. Issues
  and forking stay enabled.

## v1.0.7 — 2026-07-27

Codex install command.

- The `### Codex` subsection under `## Agent integrations` was missing its
  install command — it told readers to add the marketplace and stopped
  there. Added `codex plugin add goenv@psyb0t` right after the
  `codex plugin marketplace add psyb0t/agents` line.
- Clarified the two distinct invocation forms: installed via the
  marketplace the skill invokes as `$goenv:goenv`; picked up automatically
  from a repo's own `.agents/skills/` (no install) it invokes as plain
  `$goenv`. No library code changed.

## v1.0.6 — 2026-07-27

Agent-plugin manifests.

- Added `.agents/.claude-plugin/plugin.json` and `.agents/.codex-plugin/plugin.json` so
  the existing `.agents/skills/goenv/` skill installs natively as a plugin in Claude Code
  and Codex, rooted at `.agents/` with `skills: "./skills/"`.
- Added a `## Agent integrations` README section with the `claude plugin` / `codex plugin`
  / `openclaw skills install` commands. No library code changed.

## v1.0.5 — 2026-07-27

Go 1.26 + lint tooling.

- Bumped the `go` directive to 1.26 (`go.mod` + CI).
- `make lint` now runs Go 1.26's built-in `go fix -diff` (read-only check)
  alongside `go vet`, and a `make lint-fix` target was added to apply fixes. No
  library code changed.

## v1.0.4 — 2026-07-27

Self-hosted README badges.

- **Coverage / version / license badges are self-rendered SVGs** served from
  `raw.githubusercontent.com/psyb0t/goenv/badges/*.svg` — no third-party render
  service. `make test-coverage` writes the percentage to `coverage-percent.txt`,
  the pipeline uploads it, and a `badges` job bakes it into the SVG. The CI badge
  is switched to GitHub's native `badge.svg`. No library code changed.

## v1.0.3 — 2026-07-26

README badges.

### Added
- pkg.go.dev reference + GitHub Actions CI status badges.

No library code changed — `goenv.go` is untouched.

## v1.0.2 — 2026-07-26

### Added
- **ClawHub agent skill.** Added `.agents/skills/goenv/` (SKILL.md + `references/setup.md`) documenting the full API — `Get()` / `IsProd()` / `IsDev()`, the `Prod` / `Dev` / `EnvVarName` constants, the exact `"dev"`-or-else-`prod` fail-safe mapping (only the literal `"dev"` counts as dev; everything else, including unset, resolves to `prod`), and the caveat that `Type` is a `string` alias with no compile-time enum enforcement.
- **Skill publish CI.** `pipeline.yml` gains a tag-gated `publish-to-clawhub` job that runs after lint + tests and publishes the skill to ClawHub on release tags.

No library code changed — `goenv.go` is untouched.

## v1.0.1 — 2026-03-14

- README wording tweak. No code change.

## v1.0.0 — 2026-03-14

- Initial release. Reads the `ENV` environment variable and reports the environment via `Get()` (`"prod"` / `"dev"`), `IsProd()`, and `IsDev()`, defaulting to `prod` when `ENV` is unset or anything other than the literal `"dev"`. Zero dependencies beyond the standard library.
