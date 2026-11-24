---
title: "Architecture"
description: "How a servicepack process is laid out: the runner, the service manager, generated registration, logging and the ways to deploy services."
sidebar_position: 4
---

# Architecture

servicepack gives a project one process lifecycle and a place for several
services. It does not tie every later deployment to that one process.

## Runtime shape

```
cmd/main.go
  ├─ set global log scope: binary, commit, version
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

`cmd/main.go` stays small. It sets the process identity, registers the
generated service factories, adds the `run` command and one command namespace
per service, then hands control to the runner. The
[runner README](https://github.com/psyb0t/servicepack/blob/main/pkg/runner/README.md)
covers its signal and deadline rules.

`internal/app` runs the application hooks and hands the services to the
[service manager](https://github.com/psyb0t/servicepack/blob/main/internal/pkg/service-manager/README.md).
Whole-process behavior belongs in `App` hooks; a service should not start or
stop its siblings.

## One process or several deployments

During development, keeping related services in one process makes debugging
concrete: one binary, one cancellation path, one log stream, and every failure
in front of you.

```
my-service process
  ├─ api service
  ├─ worker service
  └─ scheduler service
```

In production you can keep that shape, or run the same image several times
with a different `SERVICES_ENABLED` in each deployment:

```
one deployment                        several deployments of one image
--------------                        --------------------------------
my-service run                        SERVICES_ENABLED=api       my-service run
  ├─ api                              SERVICES_ENABLED=worker    my-service run
  ├─ worker                           SERVICES_ENABLED=scheduler my-service run
  └─ scheduler
```

Each deployment builds and runs only the services it lists. `Dependencies()`
on a service that runs in another deployment is skipped with a warning, so it
no longer orders or waits for anything.

Keep services in one deployment when they share a release cadence and an
operational boundary. Split them when they need separate scaling, ownership,
security boundaries or failure isolation. Splitting changes how the services
talk to each other: in-process calls and dependency declarations become
explicit APIs or messages, with their own authentication, retries, readiness
checks and monitoring. servicepack does not provide that layer.

## Registration and lazy construction

`make service-registration` finds every `Service` implementation under
`internal/pkg/services/` and writes `internal/pkg/services/services.gen.go`.
The generated code registers a factory, the package's `New`, for each service,
not a built service.

That matters because:

- `run` builds only the services `SERVICES_ENABLED` selects, or all of them;
- a service's command builds only that service;
- config parsing and connections happen in `New`, not when the package is
  imported.

So running one service's command does not open every database and client in
the project. The [reference](reference.md#service-package-rules) has the rules a
service package must follow for the generator.

## Logging and configuration

Logging is configured by the `slogging` package from the `LOG_*` variables and
flows through `ctxscope`. The binary name, build commit and version are in the
global scope, and the service manager adds a `service` field while it runs or
stops a service. Pass the context you receive in `Run` on to everything the
service calls, so cancellation and those fields travel together. To send logs
somewhere else as well, add a handler with `slogconf.AddSink` in `cmd/init.go`.

Each service parses its own typed config with `gonfiguration` in `New`. The
framework's variables are in the [reference](reference.md#framework-environment-variables);
your application's variables belong in your own documentation.

## Framework and project code

`make servicepack-update` replaces framework code and leaves project code
alone. [Framework updates](framework-updates.md#what-an-update-changes) has the
exact boundary. In short, your code lives in `internal/pkg/services/`,
`cmd/init.go`, `cmd/commands.go`, `Makefile`, `scripts/make/`, the Dockerfiles
and `tests/`.
