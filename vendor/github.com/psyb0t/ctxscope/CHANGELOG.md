# Changelog

All notable changes per release. Versions follow [semver](https://semver.org).

## v1.0.3 — 2026-08-08

Documentation. No code change.

- **Fixed a false claim.** The README listed the exported surface and said "that's
  all of it", but omitted all four exported types — `Handler`, `Scope`,
  `Attribute` and `Value`. The list now covers every exported symbol: 11
  functions and 4 types, matching `go doc` exactly.
- Every function is documented with its real signature rather than a bare name,
  so the reader can see what it takes and returns without leaving the page.
- Added a **Wiring it at the edges** section with the four boundaries that
  actually matter — HTTP middleware, publishing to a queue, re-seeding on the
  receiving side, and stamping process facts at startup. The README described
  crossing a boundary in prose but never showed one.
- Documented that the global tier is an atomic pointer to an immutable map
  (lock-free reads, copy-and-swap writes) and that the context tier needs no
  locking at all, since a `context.Context` is immutable.
- Added a table of contents, now that the page is past the length where one is
  needed.

## v1.0.2 — 2026-08-08

Stops forcing a linter upgrade on everything that imports this module. No
library code changed.

- **Fixed:** the `tool` directive pinned `golangci-lint` at `v2.12.2`, the
  latest at the time the module was created. A tool dependency still appears as
  a `require` in `go.mod`, and Go selects the maximum version across the whole
  module graph — so importing this package dragged every consumer's linter up
  to `v2.12.2` regardless of what they had pinned. The newer linter then
  reported dozens of findings in code those repos had not touched, turning a
  one-line import change into a red build.

  Now pinned to `v2.4.0`, matching `ctxerrors`, so importing this module cannot
  raise anyone's linter. Same reasoning applies to any library carrying a
  `tool` directive: pin conservatively, because it is not local to your module.

## v1.0.1 — 2026-08-08

Documentation correction and the test coverage that would have caught it. No
behaviour changed.

- **Corrected:** the README and `NewHandler`'s doc comment said a plain
  `slog.Info` call "hands the handler a background context, which has no scope
  on it". Only the per-context tier is missing — the global tier set by
  `SetGlobal` never came from a context, so it still lands on the line. The old
  wording implied a plain call was unattributed, which it is not.
- Added a test asserting both halves of that (`request_id` absent, `service`
  present). The previous test only asserted the absence, which let the
  documentation overclaim without failing anything.
- Added a test for the `slog.SetDefault` + package-level `slog.InfoContext`
  path the README recommends. Every existing handler test drove a logger
  constructed in the test, so the documented setup itself had no coverage.

## v1.0.0 — 2026-08-08

First release. This package was `github.com/psyb0t/common-go/scope`; it now
lives on its own so it can ship on its own schedule.

The starting version is `v1.0.0` rather than a continuation of common-go's
`v0.3.x` because a changed module path is a different module, not a new major
of the old one — the version sequence starts fresh.

- **Extracted from `common-go`.** The API is unchanged apart from the package
  name: `scope.Set` → `ctxscope.Set`, and so on for the whole surface. The
  motivation is release coupling, not dependencies — Go's module-graph pruning
  already kept common-go's gorm/echo/NATS/Temporal requirements out of a
  scope-only build. What it did not do was let a change to the scope package
  ship without also shipping whatever else in common-go happened to be in
  flight, or spare five importing repos from version bumps driven entirely by
  code they never compile.
- **New: `NewHandler`.** A `slog.Handler` that reads the scope off the context
  passed to `Handle` and applies it to every record, so plain
  `slog.InfoContext(ctx, ...)` carries the attributes — including from library
  code that has never heard of this package, which `GetLogger` cannot reach.
  Install it once at startup with `slog.SetDefault`.

  It replays `WithAttrs`/`WithGroup` calls *after* applying the scope, so scope
  attributes always land at the record's top level. Adding them to the record
  instead would nest them inside any open group, and a `request_id` under a
  group is not the one log queries match on.

  `NewHandler` and `GetLogger` are alternatives, not layers — `GetLogger`
  applies the scope itself, so using both emits every attribute twice.
- Stdlib-only apart from `ctxerrors`, and that is enforced by the module
  boundary now rather than by a note in a document. Transport adapters (Temporal
  `ContextPropagator`, NATS header injector, HTTP middleware) depend on this
  package, never the reverse.
