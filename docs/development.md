# Development

The supported everyday workflow runs Go tooling inside the project's Docker
development image. That removes host-version drift and means a clean machine
needs Docker rather than a separately curated Go toolchain.

The Makefile has two execution shapes:

- ordinary build, lint, formatting, dependency, and vulnerability work runs
  in the development container with the repository mounted at `/work`;
- `make test-unit` uses the ordinary socketless container; full, integration,
  and coverage targets mount the Docker socket for Testcontainers.

The exact scripts and override lookup live in the
[framework Make-script deep dive](../scripts/make/servicepack/README.md).

## Common targets

| Target | Runs where | Purpose |
| --- | --- | --- |
| `make dev-image` | Docker build | Build the development image used by normal tooling. |
| `make test` | Docker + socket | Race-enabled `go test ./...`. |
| `make test-unit` | Docker | Race-enabled `go test ./...` without Docker-socket access. |
| `make test-integration` | Docker + socket | Uncached, race-enabled test run with a ten-minute timeout. |
| `make test-coverage` | Docker + socket | Race-enabled coverage check; default floor is `MIN_TEST_COVERAGE=90`. |
| `make lint` | Docker | `shfmt`, ShellCheck, `go fix` diff check, and golangci-lint. |
| `make lint-fix` | Docker | Apply supported formatting and lint fixes. Review its diff. |
| `make format` | Docker | Run gofumpt and shfmt. |
| `make audit` | Docker | Run `govulncheck` over reachable Go code. |
| `make generate` | Docker | Execute package-local `go:generate` directives. |
| `make build` | Docker | Build a static executable under `build/`. |
| `make docker-build` | Docker | Build the production image. |
| `make run-dev` | Docker | Build/run the development image with the race detector. |

Use `make help` as the current target list. Do not replace these calls with a
host `go test` or host linter in project automation: that gives two different
toolchains two chances to disagree.

## Tests, integration tests, and coverage

`make test` is the standard full suite. It enables Go's race detector and has
Docker available for Testcontainers-backed tests. `make test-integration` uses
the same package pattern but disables the Go test cache and applies a
ten-minute timeout—use it when checking changes to real-infrastructure tests.
Use `make test-unit` for the same race-enabled package walk without exposing
the Docker socket; Testcontainers-backed cases cannot start infrastructure in
that target.

`make test-coverage` also runs the race detector. It calculates framework
coverage after excluding `cmd/`, the complete `internal/pkg/services/` tree,
and generated service-manager mocks. That keeps the template's quality gate on
the reusable framework while leaving a downstream project free to establish
coverage policy for its own services. Override the threshold deliberately:

```bash
make test-coverage MIN_TEST_COVERAGE=95
```

The target writes `coverage-percent.txt` for the badge workflow and removes
the temporary coverage profiles. Testcontainers needs access to the Docker
daemon; if Docker cannot create containers, fix Docker access rather than
working around the integration tests.

## Build outputs and runtime identity

`make build` uses the pinned Go build image, installs the static-build
requirements inside that temporary container, and writes `build/<module-tail>`
back with your host UID/GID. It injects two values with linker flags:

- `main.appName` — the final segment of the module path; it sets the binary
  name and root Cobra command name;
- `main.buildCommit` — the checked-out `HEAD` commit when available.

`cmd/main.go` puts those into the global log scope as `binary` and `commit`.
Build output is therefore traceable without hardcoding identity in service
code. A source tree without a resolvable Git `HEAD` builds, but has no commit
field to inject.

`make docker-build` is a separate production-image path. Do not assume a
Dockerfile build and `make build` have the same customization hooks; inspect
the relevant script before changing either.

## Dependencies and code generation

Use the Make targets so dependency metadata and `vendor/` stay synchronized:

```bash
make dep
make pkg-add PKG=example.com/module@v1.2.3
make pkg-update PKG=example.com/module
make pkg-upgrade
make pkg-remove PKG=example.com/module
```

The framework commits Go's `vendor/` tree. Do not edit vendored sources by
hand. `make service` already regenerates service registration; use
`make service-registration` after manual service changes. Treat
`internal/pkg/services/services.gen.go` as generated output.

## Override, do not fork framework plumbing

`Makefile` includes `Makefile.servicepack`. Define the same target in your
project Makefile to replace a framework target, or add your own targets next
to it. Framework script lookup prefers `scripts/make/<script>.sh` over
`scripts/make/servicepack/<script>.sh`.

```bash
cp scripts/make/servicepack/test.sh scripts/make/test.sh
# edit scripts/make/test.sh for project-specific behavior
```

Likewise, project `Dockerfile`, `Dockerfile.dev`, `cmd/init.go`, and
`cmd/commands.go` are yours. The framework-owned versions are updated by
`make servicepack-update`; see [framework updates](framework-updates.md).

## Before you hand off a change

For a normal code change, run the narrowest useful target first, then the
relevant full check—for example `make test`, `make lint`, and
`make test-coverage` when a framework behavior changes. The repository's
pre-commit hook calls `make lint && make test-coverage`, so it will repeat
those checks during the project's normal commit flow.

Read [getting started](getting-started.md) for ownership boundaries and
[architecture](architecture.md) for where a change should live.
