# Changelog

All notable changes per release. Versions follow [semver](https://semver.org).

## v1.5.1 — 2026-07-26

Coverage reporting to Codecov + README badges.

- **Codecov coverage upload.** `pipeline.yml` enables the reusable workflow's
  Codecov step; `make test-coverage` keeps `coverage.txt` (previously deleted
  on exit) so CI can upload it.
- **README badges.** pkg.go.dev reference + GitHub Actions CI status badges.
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
