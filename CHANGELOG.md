# Changelog

All notable changes per release. Versions follow [semver](https://semver.org)
pre-1.0 conventions: minor bumps may include breaking REST changes (called
out explicitly), patch bumps are docs / build / fixes only.

## v1.2.16 — 2026-08-06

Two service-manager tests synchronized on `time.Sleep`. Both are fixed. Test
files only — no framework behaviour changed, but because these files ship with
the framework, every consumer inherits the flake until it updates.

### Fixed

- **`TestServiceManager_Stop` raced the manager.** It slept 5ms captioned "give
  services time to start", then called `Stop` and asserted every service had
  `Stop` called. `Stop` iterates `startGroups`, and a group lands there only
  after every service goroutine has launched and `waitGroupReady` has returned
  — so a `Stop` arriving before that append finds nothing to stop and every
  assertion fails. The sleep was the synchronization, not a courtesy pause:
  setting it to zero fails the test outright. It now waits for the manager to
  register the services, which is the precondition `Stop` actually has.
- **The same test used a 10ms context deadline as both a hang guard and a
  timer.** Under load the deadline could fire before `Stop` was ever called,
  ending the run for the wrong reason. `Stop` cancels the manager's context and
  closes each mock's channel, so the run ends on its own; the bound is now long
  enough to only ever mean "hung".
- **`TestServiceManager_Run` had the same sleep**, before asserting every
  service had `Run` called, and three of its rows spawned a goroutine that slept
  10ms and then cancelled the context — racing the manager to start the very
  services the row was about to assert on. Cancellation moved into the
  `stopMethod` switch, where it happens after the wait. The `contextSetup`
  field is gone: two rows returned a plain `WithCancel`, so the field's only
  real content was that race.

Under enough load to delay the manager past those windows, both tests failed for
reasons unrelated to the code under test — surfacing as an unexplained flake in
a consumer's CI, one package deep in an unrelated repo.

- **`TestIntegration_RetryWithDependencies` asserted an ordering the framework
  does not promise.** It required `db` to reach `Run` before `api`, and that was
  wrong roughly once in 600 runs. `ReadyNotifier`'s own contract says a service
  which does not implement it is "considered ready immediately after their
  goroutine is launched" — so the manager orders the LAUNCH of the groups, not
  the moment each service enters `Run`. `db`'s goroutine is started and has
  signalled before `api`'s exists, but it can be descheduled between that signal
  and its own `Run` body. Neither mock in that test implements `ReadyNotifier`,
  so the ordering was a coin flip weighted heavily enough to pass almost always.
  The assertion is gone; the guarantee is still covered by
  `TestServiceManager_ReadyNotifier`, which drives two `ReadyMockService`s and
  checks the exact sequence deterministically.

### Added

- **`.gitleaks.toml`** — this repo had no secret-scanning config at all. The
  allowlist covers only genuinely untracked paths (each one verified with
  `git ls-files`) plus `vendor/`, whose contents are third-party source
  reproduced verbatim. It allowlists by PATH only, never by a regex on the
  matched text, since the latter would silence a real credential in any file
  whose name merely looks like a fixture.

### Documentation

- **`.agents/skills/servicepack/SKILL.md` now says what `Dependent` alone
  actually guarantees.** It ordered the LAUNCH and the docs did not distinguish
  that from readiness, so "dependency-ordered startup" reads as a promise the
  framework only keeps when the dependency also implements `ReadyNotifier`.
  Spelled out, since that distinction is precisely what made the removed test
  assertion wrong.

## v1.2.15 — 2026-08-01

CI/infrastructure only. No code in this repo changed — the whole diff since v1.2.14 is under
`.github/workflows/`.

- **Split the pipeline** — building and publishing stay in `pipeline.yml`, and everything that
  leaves the host now lives in its own file beside it.
- **Mirrored to Codeberg as well as GitLab.**
- **Archived to the Wayback Machine, Software Heritage and archive.org.**
- **Issues opened on either mirror are copied back to GitHub** every six hours, and closed here
  when the original closes.
- **Pull requests are switched off on both mirrors** — they are force-pushed from GitHub, so
  anything merged there would be destroyed by the next sync. Issues and forking stay enabled.

## v1.2.14 — 2026-07-27

Codex README subsection was missing its install command.

- **Fixed the Codex subsection under "Agent integrations"** — it told readers to run
  `codex plugin marketplace add psyb0t/agents` and then stopped, never showing the actual
  install command. Added `codex plugin add servicepack@psyb0t` right after it.
- **Clarified the two invocation forms** — installed via the marketplace, the skill invokes
  as `$servicepack:servicepack`; picked up automatically from a repo's own
  `.agents/skills/` with no install, it invokes as plain `$servicepack`.

## v1.2.13 — 2026-07-27

Agent-client distribution manifests.

- **Added `.agents/.claude-plugin/plugin.json` and `.agents/.codex-plugin/plugin.json`** —
  metadata manifests that make the existing `.agents/skills/servicepack` skill installable
  as a plugin in Claude Code and Codex, rooted at `.agents/` so both clients discover the
  skill directory with no extra config.
- **Added an "Agent integrations" README section** with the copy-pasteable install commands
  for Claude Code (`claude plugin marketplace add psyb0t/agents` +
  `claude plugin install servicepack@psyb0t`), Codex
  (`codex plugin marketplace add psyb0t/agents`), and OpenClaw
  (`openclaw skills install @psyb0t/servicepack`).

## v1.2.12 — 2026-07-27

Self-hosted README badges.

- **Coverage / version / license badges are self-rendered SVGs** served from
  `raw.githubusercontent.com/psyb0t/servicepack/badges/*.svg` — no third-party
  render service. `make test-coverage` now writes the coverage percentage to
  `coverage-percent.txt`, the pipeline uploads it as an artifact, and a `badges`
  job bakes it into the SVG. The CI badge is switched to GitHub's native
  `badge.svg`.
- The `coverage-percent.txt` write lives in the framework-distributed
  `scripts/make/servicepack/test_coverage.sh`, so servicepack-based projects pick
  it up on `make servicepack-update`.

## v1.2.11 — 2026-07-27

Dependency bumps (own libraries).

- Bump `github.com/psyb0t/ctxerrors` 0.2.3 → 0.3.1,
  `github.com/psyb0t/goenv` 1.0.0 → 1.0.3, and
  `github.com/psyb0t/gonfiguration` 1.5.0 → 1.5.1. Vendored tree re-synced. No
  framework code changed.

## v1.2.10 — 2026-07-27

Ship a starter Dependabot config; keep it downstream-owned.

- **Added `.github/dependabot.yml`** — a starter age-gated Dependabot config.
  Weekly version-update PRs for `gomod` + `github-actions`, quarantined via
  `cooldown` (major 30 / minor 14 / patch 7 days) so a freshly-published release
  must sit public before it can be proposed. The starter excludes
  `github.com/psyb0t/*` from the cooldown (own packages update immediately) —
  change that prefix to your own module namespace.
- **`.servicepackupdateignore` now ignores `.github/dependabot.yml`** by
  default, so `make servicepack-update` won't overwrite your customized
  dependency policy. Delete that line if you'd rather the framework keep it in
  sync.

## v1.2.9 — 2026-07-27

Deflake the service-manager integration tests.

- **Fixed flaky integration tests.** Several `service-manager` integration tests
  cancelled the run after a fixed `time.Sleep` (100–200ms) and then asserted that
  the async startup sequence (retry, dependency-gated start, ordering) had
  finished within that window. On slow or loaded CI runners under `-race` the
  window was occasionally too short, so `TestIntegration_RetryWithDependencies`
  and its siblings flaked. They now wait for the expected end-state via a
  `waitThenCancel` helper that polls a condition (with a safety timeout) before
  cancelling — deterministic, no wall-clock assumption. Test-only; no framework
  code changed.

## v1.2.8 — 2026-07-26

README badges.

- Added pkg.go.dev reference + GitHub Actions CI status badges. No framework
  code changed.

## v1.2.7 — 2026-07-25

ClawHub agent skill + publish wiring.

- Added the ClawHub agent skill under `.agents/skills/` and a tag-gated
  `publish-to-clawhub` job in `pipeline.yml` that publishes it on release tags.
  No framework code changed.

## v1.2.6 — 2026-07-21

Make the self-updating updater future-proof: the newest update logic always
drives the update.

- `make servicepack-update` is a self-updating updater — the script that
  performs the update is itself one of the files being updated, so any policy
  baked into it (rsync excludes, dependency merge) could never protect the very
  update that installs the fix. `scripts/make/servicepack/servicepack_update.sh`
  is now a thin, stable bootstrap: it only verifies preconditions, fetches the
  latest framework, and hands off to `scripts/make/servicepack/do_update.sh`
  **from the freshly downloaded copy**.
- New `scripts/make/servicepack/do_update.sh` carries all the update policy
  (rsync exclude list, `go.mod` upgrade-only dependency merge, branch/commit
  flow) and always runs from the fresh download. Future policy changes now take
  effect on the first update that ships them — no manual bootstrap needed.

## v1.2.5 — 2026-07-21

Make `make servicepack-update` safe: never downgrade or drop the downstream
project's own dependencies.

- `scripts/make/servicepack/servicepack_update.sh` now excludes `go.mod`,
  `go.sum`, and `CHANGELOG.md` from the rsync sync. Previously rsync copied
  servicepack's own `go.mod` over the downstream project's, dropping every
  `require` line for the project's own deps; the subsequent `go mod tidy`
  then re-resolved those deps DOWN to the lowest version MVS allowed — a
  silent downgrade (e.g. a direct dep pinned at v3.44.0 fell to v3.8.1 via a
  leftover transitive floor). `CHANGELOG.md` documents the downstream
  project's releases, not servicepack's, so it is no longer overwritten.
- `scripts/make/servicepack/_post_update.sh` now merges the framework's
  direct `require` entries and `tool` directives into the downstream's
  existing `go.mod` UPGRADE-ONLY before running `make dep`: it adds what's
  missing, bumps what the framework raised, and never touches a dep the
  project already pins at an equal-or-higher version. Framework dependency
  bumps still land while the project's own deps stay intact.
- `.golangci.yml` is still framework-owned and overwritten on update; a
  project that has customized it opts out via its own
  `.servicepackupdateignore`, as before.

## v1.2.4 — 2026-07-21

Fix a flaky integration test in the service manager.

- `TestIntegration_FullStack` asserted that same-group services (`db`, `cache`)
  record their start before the next dependency group (`migrator`, `api`),
  but same-group services are launched as concurrent goroutines with no
  ordering guarantee between them. The test now waits for both group-0
  services to actually invoke their run callback (via `sync.WaitGroup`)
  before the next group's services record their start, removing the
  scheduler race. No production code changed.

## v1.2.3 — 2026-04-29

Fix a context leak on the app and service-manager run paths.

- `App.Run` and `ServiceManager.Run` now `defer cancel()` on the
  `context.WithCancel` they create, so the cancel func is always called
  even on early-return paths.
- Added a CI workflow restricting certain automation to repo collaborators.

## v1.2.2 — 2026-04-01

Lint fixes only, no functional changes. Retagged the same commit as v1.2.1.

## v1.2.1 — 2026-04-01

Fix `lll` tab-width and line-length lint issues across `app_test.go`,
`service_manager_test.go`, and the `hello-world` example service.

- Adjusted `.golangci.yml` line-length handling.

## v1.2.0 — 2026-03-31

Add lifecycle hooks to `App`.

- `App` gains `OnPreRun` and `OnPostStop` hooks, run immediately before
  `Run` starts and immediately after it returns, respectively.

## v1.1.3 — 2026-03-31

Same content as v1.2.0 (lifecycle hooks) plus build/CI housekeeping.

- Added `internal/app/app_test.go` coverage for the new hooks.
- Dockerfile and CI pipeline touch-ups; `go.mod` bump.
- README updates.

## v1.1.2 — 2026-03-22

Derive service import aliases from directory path instead of package name.

- Fixes an import alias collision when nested service directories share a
  package name.
- Added `example-nested/http` and `example-nested/grpc` — two example
  services with a shared package name (`server`) in different directories,
  demonstrating the fix.

## v1.1.1 — 2026-03-20

Read the module path from `go.mod` in service registration instead of
hardcoding `servicepack`.

- Fixes broken `service-manager` imports in projects generated via
  `make own` (renamed module path).

## v1.1.0 — 2026-03-20

Lazy service initialization via factory-based registration.

- `services.Init()` now registers factories instead of eagerly calling
  `New()` on every service.
- `./app run` instantiates all enabled services (filtered by
  `SERVICES_ENABLED`).
- `./app <service> <subcommand>` instantiates only that one service.
- Standalone CLI commands (`cmd/commands.go`) no longer touch any service.
- Docs updated to match.

## v1.0.7 — 2026-03-20

Remove app-level config, simplify runner config.

- Removed the now-dead `internal/app/config.go` and its `gonfiguration`
  usage.
- Renamed the `APPRUNNER_` env prefix to `RUNNER_`.
- Runner config now uses a struct `default` tag instead of a separate
  `SetDefaults` method / const block.

## v1.0.6 — 2026-03-19

Prevent `go tool modernize -fix` from mutating generated files.

- `lint_fix.sh` now runs `git checkout` on generated files after the
  `modernize -fix` pass, so codegen output isn't silently rewritten by the
  linter.

## v1.0.5 — 2026-03-19

Run Makefile scripts via `bash` explicitly.

- The `find_script` macro now prepends `bash`, so script file permissions
  no longer matter for `make` targets to work.

## v1.0.4 — 2026-03-19

Add `modernize` as a vendored Go tool; exclude generated files from its
output.

- Added `golang.org/x/tools/gopls/internal/analysis/modernize` as a
  `go tool` dependency (vendored) instead of `go run @latest`.
- Lint scripts invoke it via `go tool modernize`.
- `.gen.go` files are filtered out of modernize's suggestions.

## v1.0.3 — 2026-03-14

Add a `cmd/commands.go` extension point for custom CLI commands, plus
service-manager command wiring.

- `cmd/commands.go` is a user-owned file (never overwritten by framework
  updates) for defining project-specific `cobra.Command`s.
- `cmd/main.go` registers both the service manager's generated commands and
  the user's custom commands on the root CLI.
- `scripts/make/servicepack/own.sh` and `.servicepackupdateignore` updated
  to keep `commands.go` out of framework-update overwrites.

## v1.0.2 — 2026-03-14

Add `ReadyNotifier` — optional service readiness signalling.

- A service can implement `Ready() <-chan struct{}` to signal the service
  manager it has finished starting up before the manager launches the next
  dependency group.
- The service manager waits on `Ready()` (via `waitGroupReady`) before
  proceeding.

## v1.0.1 — 2026-03-14

Fix the release pipeline workflow.

## v1.0.0 — 2026-03-14

Initial public release of servicepack: a Go service-manager framework with
dependency-ordered startup, retries, allowed-failure services, and a
generated CLI.
