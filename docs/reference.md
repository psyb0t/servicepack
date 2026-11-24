---
title: "Reference"
description: "servicepack reference: framework environment variables, service interfaces with signatures, service package rules and the Make targets."
sidebar_position: 7
---

# Reference

The exact names and signatures, in one place. The other pages explain when to
use them.

## Framework environment variables

These belong to the framework. Your services define their own variables in
their `Config` structs.

| Variable | Meaning | Default |
| --- | --- | --- |
| `LOG_LEVEL` | `debug`, `info`, `warn`, or `error` logging threshold. | `info` |
| `LOG_FORMAT` | `json` or `text` output. | `text` |
| `LOG_ADD_SOURCE` | Include the source location in log records. | `false` |
| `ENV` | Read by `goenv`. Only `dev` is recognized; any other value, or none, means `prod`. The framework logs it at startup and does not change behavior on it. | `prod` |
| `RUNNER_SHUTDOWNTIMEOUT` | Deadline for the whole graceful shutdown, as a Go duration. | `10s` |
| `SERVICES_ENABLED` | Comma-separated list of the services to run in this process, by `ServiceName`. Empty or unset runs every registered service. | all |

```bash
SERVICES_ENABLED=price-worker,api LOG_LEVEL=debug ./build/my-service run
```

## Service interfaces

Defined in
[`internal/pkg/service-manager/service_manager.go`](https://github.com/psyb0t/servicepack/blob/main/internal/pkg/service-manager/service_manager.go).
Every service implements `Service`. The others are optional, and one service
can implement any combination of them.

```go
type Service interface {
	Name() string
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Retry a failed Run up to MaxRetries() times, RetryDelay() apart.
// A negative MaxRetries counts as 0.
type Retryable interface {
	MaxRetries() int
	RetryDelay() time.Duration
}

// Return true to keep the application running after this service
// fails for good.
type AllowedFailure interface {
	IsAllowedFailure() bool
}

// Start after the named services.
type Dependent interface {
	Dependencies() []string
}

// Close the channel once the service can serve. Later dependency
// groups wait for it.
type ReadyNotifier interface {
	Ready() <-chan struct{}
}

// Add subcommands under ./build/<app> <service> <subcommand>.
type Commander interface {
	Commands() []*cobra.Command
}
```

[Services and lifecycle](services-and-lifecycle.md) explains how the manager
applies each one.

## Service package rules

`make service-registration` scans `internal/pkg/services/` for types that
implement `Service` and writes `internal/pkg/services/services.gen.go`. For each
implementation, its package must export:

- `const ServiceName`, the name used to register the service, to select it in
  `SERVICES_ENABLED`, and as its command namespace;
- `func New() (*T, error)`, where `*T` is the type that implements `Service`.

`make service` writes both. If you write a service by hand, add them, or the
generated file does not compile. Keep one `Service` implementation per package.

Return `ServiceName` from `Name()`. The manager uses `Name()` to resolve
`Dependencies()` and to name the service in logs, so a different value makes
dependency declarations and `SERVICES_ENABLED` refer to different names.

Packages can be nested, for example `internal/pkg/services/http/api`.
`make service` does not accept a `/` in `NAME`, so create nested packages by
hand and run `make service-registration`. The
generator builds each import alias from the path under `services/` with `/` and
`-` removed (`http/api` becomes `httpapi`), so packages with the same name in
different directories do not collide. With no services at all, the generated
`Init` panics with `No services`.

## Make targets

Run `make help` for the list your copy has. `servicepack-reformat` also
appears there; the update uses it internally. The **Script** column names the
script a target runs. Copy that script from `scripts/make/servicepack/` to
`scripts/make/` to override it (see
[development](development.md#override-a-target)). Targets without a script are
defined directly in `Makefile.servicepack`; override those by defining the
target in your `Makefile`.

| Target | Script | What it does |
| --- | --- | --- |
| `make own MODNAME=<module>` | `own.sh` | Turn a fresh clone into your project. Run it once. |
| `make service NAME=<name>` | `service.sh` | Create a service and regenerate registration. |
| `make service-remove NAME=<name>` | `service_remove.sh` | Delete a service and regenerate registration. |
| `make service-registration` | `service_registration.sh` | Regenerate `services.gen.go`. |
| `make build` | `build.sh` | Build the static binary into `build/<app>`. |
| `make run-dev` | `run_dev.sh` | Build the development image and run the app in it with the race detector. |
| `make docker-build` | `docker_build.sh` | Build the production image. |
| `make dev-image`, `make docker-build-dev` | `docker_build_dev.sh` | Build the development image the other targets run in. |
| `make shell` | | Open a shell in the development image. |
| `make test` | `test.sh` | Race-enabled `go test ./...` with the Docker socket. |
| `make test-unit` | | Race-enabled `go test ./...` without the Docker socket. |
| `make test-integration` | | Uncached race-enabled `go test -tags=integration ./...` with the Docker socket and a ten-minute timeout. |
| `make test-coverage` | `test_coverage.sh` | Coverage run with `-tags=integration`; fails under `MIN_TEST_COVERAGE` (default `90`). |
| `make lint` | `lint.sh` | shfmt, ShellCheck, a `go fix` diff check and golangci-lint. |
| `make lint-fix` | `lint_fix.sh` | Apply the supported fixes. Review the diff. |
| `make format` | | gofumpt and shfmt. |
| `make sec` | `sec.sh` | govulncheck and semgrep, merged into `sec.sarif`. Fails on any finding. Needs network access for the semgrep rules. |
| `make generate` | | `go generate ./...`. |
| `make clean` | `clean.sh` | Remove build output and coverage files. |
| `make all` | | `dep`, `lint-fix`, `test-coverage`, then `build`. |
| `make dep`, `make pkg-lock` | | `go mod tidy` and `go mod vendor` without changing versions. |
| `make pkg-add PKG=<module>@<version>` | | Add a module. |
| `make pkg-update PKG=<module>` | | Update one module to its latest version. |
| `make pkg-upgrade` | | Upgrade every direct module to its latest version. |
| `make pkg-remove PKG=<module>` | | Remove a module. |
| `make pkg-add-tool MODULE=<module>@<version> TOOL=<package>` | | Add a Go tool directive. |
| `make pkg-remove-tool TOOL=<package>` | | Remove a Go tool directive. |
| `make servicepack-update` | `servicepack_update.sh` | Bring in the latest framework on a review branch. |
| `make servicepack-update-review` | `servicepack_update_review.sh` | Show the pending update. |
| `make servicepack-update-merge` | `servicepack_update_merge.sh` | Merge the pending update. |
| `make servicepack-update-revert` | `servicepack_update_revert.sh` | Drop the pending update. |
| `make backup` | `backup.sh` | Archive the project before risky work. |
| `make backup-restore [BACKUP=<file>]` | `backup_restore.sh` | Restore a backup, the latest by default. |
| `make backup-clear` | `backup_clear.sh` | Delete every backup. |

### Module age gate

`pkg-add`, `pkg-update`, `pkg-upgrade` and `pkg-add-tool` refuse a module
version published less than seven days ago. Modules under `github.com/psyb0t/`
are exempt.

`pkg-update` and `pkg-upgrade` check the latest version and stop when it is too
new; they do not fall back to an older one. `pkg-upgrade` checks every direct
module before it changes any, so one refusal leaves `go.mod` and `vendor/` as
they were. The check is
[`scripts/check-go-age.sh`](https://github.com/psyb0t/servicepack/blob/main/scripts/check-go-age.sh).
