# Architecture

Servicepack is a clone-and-own application skeleton. It gives a project a
well-defined process lifecycle and a place for multiple services; it does not
require every future deployment to stay inside one binary.

## Runtime shape

```
cmd/main.go
  ├─ set global log scope: binary, commit
  ├─ services.Init()                         generated factory registration
  └─ Cobra "run"
       └─ pkg/runner
            ├─ signal / parent-context handling
            └─ internal/app.App
                 ├─ pre-run hooks
                 ├─ internal/pkg/service-manager
                 │    └─ service factories → services running concurrently
                 └─ post-stop hooks
```

`cmd/main.go` is intentionally small: establish process identity, register
generated service factories, expose `run` and command namespaces, then pass
control to the runner. The [runner README](../pkg/runner/README.md) explains
the signal and deadline rules.

`internal/app` owns application-level hooks and delegates service execution to
the [service manager](../internal/pkg/service-manager/README.md). `App` is the
place for whole-process behavior; individual services should not reach across
into siblings to create their own lifecycle graph.

## Local composition and deployment choices

During local development, keeping related services in one process makes
debugging concrete: one binary, one cancellation path, one structured log
stream, and direct visibility into failures. This is the default development
shape:

```
my-service process
  ├─ api service
  ├─ worker service
  ├─ scheduler service
  └─ migration command namespace
```

Production has two valid shapes:

```
one release unit                      independently deployed units
----------------                      ----------------------------
my-service binary                     api binary ──────┐
  ├─ api                               worker binary ───┼─ explicit HTTP/gRPC/queue contracts
  ├─ worker                            scheduler binary ┘
  └─ scheduler
```

Choose one binary when the services have the same release cadence and
operational boundary. Split when they need separate scaling, ownership,
security boundaries, or failure isolation. Splitting is an architecture change:
replace in-process calls and service-manager dependency declarations with
explicit APIs, messages, authentication, retries, observability, and deploy
configuration.

`SERVICES_ENABLED` is useful for a partial local run within one binary; it is
not a microservice deployment system.

## Source ownership

| Path | Role | Ownership after `make own` |
| --- | --- | --- |
| `cmd/main.go` | Process entry point and root CLI. | Framework |
| `cmd/init.go` | Extra handlers and application hooks. | Project |
| `cmd/commands.go` | App-level CLI commands. | Project |
| `internal/app/` | App lifecycle wrapper. | Framework |
| `internal/pkg/service-manager/` | Concurrency, dependency, retry, and stop semantics. | Framework |
| `internal/pkg/services/` | Business services and `services.gen.go`. | Project; generated registration is not hand-edited. |
| `pkg/runner/` | Signal-aware runner. | Framework |
| `scripts/make/servicepack/` | Updateable Make implementations. | Framework |
| `scripts/make/` | Project-specific target overrides. | Project |
| `Makefile.servicepack` | Framework Make target definitions. | Framework |
| `Makefile` | Project targets and overrides. | Project |

Framework-owned means an update may replace it. Project-owned means the
update's normal exclusion policy preserves it. See
[framework updates](framework-updates.md) before changing that boundary.

## Registration and lazy construction

`make service-registration` runs the generator that discovers `Service`
implementations and writes `internal/pkg/services/services.gen.go`. The
generated code registers factories rather than fully constructed services.

That distinction matters:

- `run` instantiates all enabled factories;
- a per-service Cobra command instantiates only that one service;
- connections and config parsing happen in a service's `New`, not at package
  import time.

This keeps command execution from accidentally opening every database/client
in the project. Details are in the
[service-manager README](../internal/pkg/service-manager/README.md).

## Observability and configuration

Logging starts with the `slogging` handler setup and flows through `ctxscope`.
The binary and build commit are global scope; the service manager adds a
`service` field while it runs or stops a service. Preserve and extend the
context passed into `Run`; it carries cancellation and those fields together.

Configuration is typed and parsed at each service boundary with
`gonfiguration`. Framework settings are documented in
[getting started](getting-started.md); application settings belong in the
project's own docs and config examples.

## Build and test topology

The Makefile runs tooling in Docker. Test targets additionally receive Docker
access for Testcontainers. The development model, coverage boundary, and
override mechanism are documented in [development](development.md), with the
exact script behavior in the [Make-script README](../scripts/make/servicepack/README.md).
