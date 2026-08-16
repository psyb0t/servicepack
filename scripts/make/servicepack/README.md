# Framework Make scripts

`scripts/make/servicepack/` is the updateable implementation layer behind
`Makefile.servicepack`. It is deliberately separate from `scripts/make/`,
where a project can replace an individual step without forking the framework.

The top-level [development guide](../../../docs/development.md) documents the
developer-facing targets; this file documents how those targets are wired.

## Lookup and override rule

`Makefile.servicepack` resolves a script name by checking:

1. `scripts/make/<name>.sh` — project override; then
2. `scripts/make/servicepack/<name>.sh` — framework default.

For example:

```bash
cp scripts/make/servicepack/test.sh scripts/make/test.sh
# make test now uses scripts/make/test.sh
```

Make the smallest override that owns your project-specific behavior. The
framework directory is updated by `make servicepack-update`; edit it only when
you are intentionally changing Servicepack itself.

## Script map

| Script | Target(s) | Responsibility |
| --- | --- | --- |
| `build.sh` | `make build` | Build a static binary in a pinned Go container and inject binary/commit identity. |
| `docker_build*.sh` | `make docker-build*` | Build production or development images. |
| `run_dev.sh` | `make run-dev` | Build/run the development image with the race detector. |
| `dep.sh` | `make dep` | Tidy and vendor the Go module. |
| `lint*.sh` | `make lint*` | Run or fix shell and Go lint findings. |
| `test*.sh` | `make test`, `make test-coverage` | Race-enabled test and coverage workflow. |
| `service.sh` / `service_remove.sh` | `make service*` | Create/remove a service directory. |
| `service_registration.sh` | `make service-registration` | Discover services and write `services.gen.go`. |
| `servicepack_update*.sh`, `do_update.sh` | `make servicepack-update*` | Bootstrap, apply, review, merge, or revert framework sync. |
| `backup*.sh` | `make backup*` | Create, restore, or clear update backups. |

`common.sh` supplies the shared shell helpers and output formatting. Keep shell
scripts portable to the development environment and fail on errors.

## Docker execution model

Most normal targets run in the development image through `DEV_RUN`, with the
repository mounted at `/work` and container UID/GID mapped back to the host.
Test targets use `DEV_RUN_DIND`: they also mount Docker access because Go tests
may launch Testcontainers infrastructure.

`build.sh`, Docker image targets, lifecycle/update tooling, and backup tooling
are host-side shell entry points that invoke Docker or Git as their job
requires. The generated binary itself still comes from the pinned build image.

Do not add a host-Go fallback to a standard quality target. That creates an
untested second toolchain and breaks the template's Docker-first promise.

## Generation contract

`service.sh` validates lowercase, hyphenated service names, creates the
service scaffold, and invokes `service_registration.sh`. The registration
script discovers implementations and writes
`internal/pkg/services/services.gen.go`. Regenerate; do not hand-edit output.

The scaffold uses `gonfiguration`, `ctxerrors`, and `ctxscope` so a new service
starts with typed config, contextual errors, service-scoped logs, and context
cancellation. Project-specific skeleton changes belong in an override or an
intentional framework release.

## Build contract

`build.sh` derives the app name from the module path, resolves `HEAD` and its
exact tag when available, runs the digest-pinned Go build image, and injects
the identity into `main.appName`, `main.buildCommit`, and `main.buildVersion`.
It writes a static binary to
`build/<app-name>` and changes ownership back to the invoking host user.

The runtime entry point uses those values as global log-scope fields. If a
source tree has no Git `HEAD`, the binary still builds with `dev` as its
version and omits the commit field.

## Updating this layer

The framework updater deliberately runs its apply policy from a freshly
downloaded Servicepack copy. This lets a release update the updater itself on
the first run. See [framework updates](../../../docs/framework-updates.md) for
the downstream safety model and exclusion policy.
