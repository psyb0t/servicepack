# Changelog

All notable changes per release. Versions follow [semver](https://semver.org).

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
