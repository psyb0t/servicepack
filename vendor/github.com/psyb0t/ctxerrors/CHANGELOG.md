# Changelog

All notable changes per release. Versions follow [semver](https://semver.org)
pre-1.0 conventions: minor bumps may include breaking API changes (called out
explicitly), patch bumps are docs / build / fixes only.

## v0.7.1 — 2026-08-11

Rewrites the `commerr` package doc comment. No code changed.

- The old comment called commerr "the vocabulary ctxerrors' error map translates
  foreign driver errors into" — implying it is the only set `SetErrorMap` can
  target — and framed the sentinels as "shared across services", too narrow for a
  public, general-purpose package. The new comment describes them as
  general-purpose sentinels any Go code can return and match with `errors.Is`,
  usable with any wrapping, and notes they work as `SetErrorMap` targets without
  claiming to be the only ones.

## v0.7.0 — 2026-08-10

Removes `commerr.ErrUnexpectedHTTPStatusCode` — it is HTTP-specific (61 → 60).

- **Breaking.** `ErrUnexpectedHTTPStatusCode` describes an HTTP status code
  coming back wrong, which is not transport-agnostic — the bar for `commerr`. It
  is removed. Code that needs an HTTP-status sentinel should define one in its
  HTTP package (`aichteeteapee` already ships `ErrUnexpectedResponseStatus`).
- Everything else stays: rate limiting and auth (`ErrRateLimited`,
  `ErrNotAuthenticated`, `ErrPermissionDenied`) are not HTTP-only — you can be
  rate-limited on gRPC or a queue and fail auth on any protocol — so they remain.

## v0.6.0 — 2026-08-10

Expands `commerr` with 16 more general-purpose sentinels (45 → 61).

- **Operation symmetry:** `ErrReadFailed` (the counterpart the existing
  `ErrWriteFailed` never had), plus `ErrOpenFailed`, `ErrCloseFailed` and
  `ErrExecFailed`.
- **Capability:** `ErrNotImplemented`, `ErrUnsupported`.
- **Validation:** `ErrValidationFailed`, `ErrOutOfRange`.
- **Lifecycle / state:** `ErrClosed` (resting state to the existing
  `ErrClosing`), `ErrNotReady`, `ErrInvalidState`, `ErrExpired`.
- **Concurrency / capacity:** `ErrLockHeld`, `ErrConflict`, `ErrExhausted`.
- **Access:** `ErrPermissionDenied` (authorization, distinct from the existing
  `ErrNotAuthenticated`).
- All are transport- and domain-agnostic, plain `errors.New` values that hold
  under `errors.Is` through any `Wrap` depth — same contract as the rest of the
  package. Nothing existing changed.

## v0.5.0 — 2026-08-10

Adds `commerr`, a subpackage holding the common sentinel errors.

- **`commerr`** ships the shared sentinel vocabulary — `ErrNotFound`,
  `ErrAlreadyExists`, `ErrFetchFailed`, `ErrTimeout`, `ErrUnavailable`,
  `ErrRateLimited`, and the rest — as plain `errors.New` values that survive any
  number of `Wrap` layers under `errors.Is`. Full list in `commerr/commerr.go`.
- These are the natural targets for `SetErrorMap`. The mapping example in the
  README now maps gorm's driver errors into `commerr.ErrNotFound` /
  `commerr.ErrAlreadyExists` rather than hand-declared locals, since the map and
  its targets finally live in one module.
- It is a subpackage, not the root package: importing `ctxerrors` for wrapping
  alone does not compile `commerr`, so a consumer that only wants file/line
  context pays nothing for the vocabulary.
- These same sentinels are also published by `common-go/errors`. That package is
  being turned into a thin, deprecated re-export of `commerr` — same error
  values, so `errors.Is` holds across both import paths — so existing imports
  keep working while the canonical home moves here.

## v0.4.5 — 2026-08-08

Documentation. No code change.

- The README said "All functions return a `*CTXError`". Three of the seven
  return nothing at all: `SetErrorMap`, `MapError` and `ClearErrorMap` manage the
  translation map. Corrected to name the four that do — `New`, `Wrap`, `Wrapf`
  and `Join` — and say what the other three are for.

## v0.4.4 — 2026-08-08

CI only, no library change.

- Enabled the imported-by badge. The README gains a count linking to
  `importers.md` on the `badges` branch — the repositories importing this
  module, grouped, package counts descending, and flagged when the owner is not
  `psyb0t`. Useful here specifically because this module has no stars and no
  external importers, yet is imported by ~17 of my own: the number is a
  blast-radius indicator for changes, not an adoption metric, and the list is
  what makes that legible.
- The pipeline now also runs on a weekly schedule so that count stays current.
  Weekly rather than daily because pkg.go.dev's crawl lags publication by days —
  a daily run would re-derive an unchanged number six extra times a week, each
  one dragging the full test suite along, since the badges job needs the
  coverage artifact. It refreshes the whole pipeline rather than the badges
  alone because the badge publisher republishes only what a run produced, so a
  badge-only job would delete the coverage, version and license badges.
- The cron slot is derived from a hash of the repository name rather than
  picked. GitHub cron has no randomness, so spreading has to be deterministic;
  the scheduler is also best-effort and sheds load at popular times, which
  round-numbered schedules land squarely in.

## v0.4.3 — 2026-08-01

CI only, no library change.

- Every branch and tag push is now mirrored to GitLab and Codeberg.
- The default branch and tags are archived to the Wayback Machine and Software
  Heritage, on push and monthly.
- Issues opened on the mirrors are pulled back into GitHub every six hours.
- Ignores a local `.telemetry/` scratch dir in git and Docker builds.

## v0.4.2 — 2026-07-31

CI only, no library change.

- Drops the `generate_command: "-"` opt-out the pipeline picked up in `v0.4.1`.
  The shared Go workflow's codegen-drift gate is now off unless a repo asks for
  it, so the explicit opt-out no longer does anything and the comment beside it
  described a mechanism that no longer exists.

## v0.4.1 — 2026-07-31

CI only, no library change.

- The pipeline now passes `generate_command: "-"` to the shared Go workflow.
  That workflow gained a codegen-drift gate which defaults to running
  `make generate` and failing if the tree moves afterwards; this repo has no
  generated files and no such target, so the job failed on `v0.4.0` and took
  the GitHub Release step with it. `-` opts out explicitly. The `v0.4.0` tag
  itself is fine and `go get` resolves it normally.

## v0.4.0 — 2026-07-31

Adds `Join`, for operations that fan out and can fail in more than one place.

- **`Join(errs ...error) error`** combines several errors into one that carries
  the call site. Nil errors are ignored and `Join` returns nil when every error
  is nil, so a caller can hand it a slice without first checking whether
  anything went wrong.
- The result unwraps to what the standard library's `errors.Join` produces, so
  `errors.Is` and `errors.As` still find any of the joined errors. What it adds
  over the standard library is the file, line and function where the join
  happened — the same context `New`, `Wrap` and `Wrapf` capture, and the reason
  to reach for this package rather than `errors` in the first place.
- Useful wherever bailing on the first failure would skip the remaining work:
  writing a record to several sinks, processing a batch of rows. See the
  "Joining errors" section of the README.

## v0.3.3 — 2026-07-27

Go 1.26 + lint tooling (`modernize` → built-in `go fix`).

- Bumped the `go` directive to 1.26 (`go.mod` + CI).
- `make lint` / `make lint-fix` now use Go 1.26's built-in `go fix` (`-diff`
  check in `lint`, apply in `lint-fix`) instead of the `modernize` analyzer, and
  the `modernize` tool directive is dropped from `go.mod`. No library code changed.

## v0.3.2 — 2026-07-27

Self-hosted README badges.

- **Coverage / version / license badges are self-rendered SVGs** served from
  `raw.githubusercontent.com/psyb0t/ctxerrors/badges/*.svg` — no third-party
  render service. `make test-coverage` writes the percentage to
  `coverage-percent.txt`, the pipeline uploads it, and a `badges` job bakes it
  into the SVG. The CI badge is switched to GitHub's native `badge.svg`. No
  library code changed.

## v0.3.1 — 2026-07-26

README badges.

- pkg.go.dev reference + GitHub Actions CI status badges. No library code
  changed.

## v0.3.0 — 2026-05-11

- Added an error mapping layer.

## v0.2.3 — 2026-01-19

- Removed the external logrus dependency; use stdlib `log/slog` for nil-error
  warnings. No API changes.

## v0.2.2 — 2026-01-16

- Updated the Go version in the CI pipeline.

## v0.2.1 — 2026-01-16

- Modernized tooling and updated dependencies.

## v0.2.0 — 2025-09-10

- Renamed the error struct to `CTXError`.

## v0.1.0 — 2025-09-07

- Initial release.
