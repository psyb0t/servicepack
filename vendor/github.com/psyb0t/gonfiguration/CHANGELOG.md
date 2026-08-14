# Changelog

All notable changes per release. Versions follow [semver](https://semver.org).

## v1.6.4 — 2026-08-08

Documentation. No code change.

- The section headed "Complete API Reference" documents the eight exported
  functions but not the sentinel errors, which live under Error Handling. It is
  now "API Reference" and points at where the sentinels are, so the heading no
  longer promises more than the section delivers.

## v1.6.3 — 2026-08-08

Repository infrastructure only, no library change.

- Added the imported-by badge: a count of the public packages importing this
  module, linking to `importers.md` on the `badges` branch — the importing
  repositories, grouped, package counts descending, and flagged when the owner
  differs from this repo's.
- It measures **blast radius, not adoption**. The distinction matters here
  because this library has no stars and is nonetheless imported by more packages
  than anything else in the fleet — nobody stars a config parser, they just
  import it. The count is what tells you how much breaks if an exported name
  moves; the external mark is what tells you whether any of it is someone else's
  problem.
- Refreshed weekly rather than daily. pkg.go.dev's crawl lags publication by
  days, so a daily run would re-derive an unchanged number six extra times a
  week, each dragging the full test suite along because the badges job needs the
  coverage artifact. The whole pipeline runs, not a badges-only job: the badge
  publisher republishes only what a run produced, so a badge-only refresh would
  delete the coverage, version and license badges.
- The cron slot is derived from a hash of the repository name rather than
  chosen. GitHub cron has no randomness, and its scheduler sheds queued runs
  hardest at the round times a human would pick.

## v1.6.2 — 2026-08-01

Repository infrastructure only, no library change.

- Every push now mirrors the repo to GitLab and Codeberg, so the source stays
  fetchable if GitHub is unavailable. Gitee is wired but left off — it binds
  repo creation to a mobile number and silently creates the repo private
  without one.
- Pushes to the default branch and every tag are archived to the Wayback
  Machine and Software Heritage, through the authenticated Save Page Now API,
  with README outlinks captured too. Feature-branch pushes are skipped because
  the archive is rate-limited.
- Issues filed on the Codeberg and GitLab mirrors are pulled back into GitHub
  every six hours, so a bug reported on a mirror reaches the same tracker.
  Scheduled runs jitter to avoid stampeding the mirrors; a manual run does not.
- `.dockerignore` keeps the local `.telemetry/` scratch dir out of any build
  context.

The Go code, the public API and `go.mod` are untouched — `v1.6.2` is byte-for-byte
`v1.6.1` as far as the library is concerned.

## v1.6.1 — 2026-07-31

CI only, no library change.

- Restores a green pipeline and the GitHub Release artifact. The shared Go
  workflow had gained a codegen-drift gate that defaulted to running
  `make generate` and failing if the tree moved afterwards. This repo generates
  nothing and has no such target, so that job failed on `v1.6.0` and the release
  step, which depends on it, was skipped along with it. The gate is now opt-in
  upstream and stays off here. The `v1.6.0` tag itself is fine and `go get`
  resolves it normally.

## v1.6.0 — 2026-07-31

Errors now carry the file, line and function they came from.

- Every error is built with [ctxerrors](https://github.com/psyb0t/ctxerrors)
  instead of `fmt.Errorf`, across `Parse`, the field walker and every type
  setter. A parse failure now names the exact field and the exact setter that
  rejected it, along with each hop it was wrapped at, rather than a bare
  message you have to trace back by hand:

  ```
  failed to parse fields: failed to set field value: field API_KEY: required field not set
    [gonfiguration.go:169 in fillFieldValue]
    [gonfiguration.go:112 in parseDstFields]
    [gonfiguration.go:44 in Parse]
  ```

- The exported sentinels in `errors.go` are unchanged and still declared with
  `errors.New`, so `errors.Is(err, gonfiguration.ErrRequiredFieldNotSet)` and
  friends match exactly as before, through the added wrapping.
- **No longer dependency-free.** `github.com/psyb0t/ctxerrors` is now a direct
  dependency, replacing the previous stdlib-only guarantee. It is a small
  package with no dependencies of its own beyond the standard library.

## v1.5.4 — 2026-07-27

Go 1.26 + lint tooling (`modernize` → built-in `go fix`).

- Bumped the `go` directive to 1.26 (`go.mod` + CI).
- `make lint` / `make lint-fix` now use Go 1.26's built-in `go fix` (`-diff`
  check in `lint`, apply in `lint-fix`) instead of the `modernize` analyzer, and
  the `modernize` tool directive is dropped from `go.mod`.
- `go fix` modernized one deprecated stdlib reference: `reflect.Ptr` →
  `reflect.Pointer` in `gonfiguration.go` (`reflect.Ptr` is a deprecated alias for
  the same constant — no behavior change).

## v1.5.3 — 2026-07-27

Self-hosted README badges.

- **Coverage / version / license badges are self-rendered SVGs** served from
  `raw.githubusercontent.com/psyb0t/gonfiguration/badges/*.svg` — no third-party
  render service. `make test-coverage` writes the percentage to
  `coverage-percent.txt`, the pipeline uploads it, and a `badges` job bakes it
  into the SVG. The CI badge is switched to GitHub's native `badge.svg`. No
  library code changed.

## v1.5.2 — 2026-07-27

- Bump `github.com/stretchr/testify` 1.10.0 → 1.11.1 (test dependency).

## v1.5.1 — 2026-07-26

README badges + repo housekeeping.

- pkg.go.dev reference + GitHub Actions CI status badges.
- Added a GitHub Sponsors funding config; CI restricted to collaborators only;
  README tweaks. No library code changed.

## v1.5.0 — 2026-03-13

- Added default-value support via a struct tag.

## v1.4.1 — 2026-01-16

- Modernized tooling and updated the Go version.

## v1.4.0 — 2026-01-16

- Added required-field support via `env:"VAR,required"`.
- Added `MustParse()` that panics on error.
- Added `ErrNilDestination`, `ErrRequiredFieldNotSet`, `ErrDefaultTypeMismatch`.
- Removed `pkg/errors`; use stdlib `fmt.Errorf` with `%w`. Cleaned up error
  messages.

## v1.3.1 — 2025-09-11

- Maintenance release.

## v1.3.0 — 2025-09-11

- Added `[]string` support.

## v1.2.0 — 2025-09-07

- Added `time.Duration` support.

## v1.1.0 — 2025-09-07

- Maintenance release.

## v1.0.0 — 2023-11-04

- Initial release.
