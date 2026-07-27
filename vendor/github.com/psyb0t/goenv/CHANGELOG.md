# Changelog

All notable changes per release. Versions follow [semver](https://semver.org).

## v1.0.3 — 2026-07-26

Coverage reporting to Codecov + README badges.

### Added
- **Codecov coverage upload.** `pipeline.yml` enables the reusable workflow's
  Codecov step. `make test-coverage` now keeps `coverage.txt` (previously
  deleted on exit) so CI can upload it.
- **README badges.** pkg.go.dev reference + GitHub Actions CI status badges.

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
