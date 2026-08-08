# Changelog

All notable changes per release. Versions follow [semver](https://semver.org)
pre-1.0 conventions: minor bumps may include breaking API changes (called out
explicitly), patch bumps are docs / build / fixes only.

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
