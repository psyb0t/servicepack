---
title: "Services and lifecycle"
description: "The servicepack service contract: Run and Stop, dependencies, readiness, retries, allowed failures, commands, service filtering, logging and shutdown."
sidebar_position: 3
---

# Services and lifecycle

servicepack supervises services that live in the same process. It provides
startup ordering, readiness gates, failure policy, service-scoped logs, and
one coordinated shutdown path. It does not turn in-process calls into a
network protocol or solve distributed-systems concerns for services you split
out later.

For the implementation-level rules and test setup, see the
[service-manager deep dive](https://github.com/psyb0t/servicepack/blob/main/internal/pkg/service-manager/README.md) and
[runner deep dive](https://github.com/psyb0t/servicepack/blob/main/pkg/runner/README.md).

## The required contract

Every service implements:

```go
type Service interface {
	Name() string
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}
```

`Name` must return the package's `ServiceName` constant: other services name it in
`Dependencies()`, `SERVICES_ENABLED` selects it, and logs show it under that
name. `Run` must return when its context is cancelled. `Stop` is where
you close listeners, drain work, flush a client, or do other bounded cleanup.

Do not use `Stop` as the first mechanism that tells a service to stop. The
manager cancels its run context before calling `Stop`; `Run` should be written
to observe that cancellation on its own.

Use `ctxscope` at service boundaries so all logs include the service name:

```go
func (s *Worker) Run(ctx context.Context) error {
	ctx = ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))
	logger := ctxscope.GetLogger(ctx)
	logger.Info("worker started")

	for {
		select {
		case <-ctx.Done():
			return nil
		case item := <-s.items:
			if err := s.handle(ctx, item); err != nil {
				return ctxerrors.Wrap(err, "handle item")
			}
		}
	}
}
```

The application entry point also sets process-wide `binary` and, for normal
builds, `commit` and `version` attributes. Do not replace those with ad-hoc global logging.

## Optional contracts

Implement any combination of these interfaces on the same service.

| Interface | Methods | Effect |
| --- | --- | --- |
| `Retryable` | `MaxRetries() int`, `RetryDelay() time.Duration` | Retry a failed `Run` up to `MaxRetries()` times, `RetryDelay()` apart. `0` means no retry, and a negative value counts as `0`. |
| `AllowedFailure` | `IsAllowedFailure() bool` | When it returns `true`: after the retries run out, log the failure and keep the rest of the application running. |
| `Dependent` | `Dependencies() []string` | Start after the named services in this process. A cycle fails startup. |
| `ReadyNotifier` | `Ready() <-chan struct{}` | Hold back later dependency groups until the service closes the channel. |
| `Commander` | `Commands() []*cobra.Command` | Add `./build/<app> <service> <subcommand>` commands that build only that service. |

The full interface definitions are in the [reference](reference.md#service-interfaces).

### Ordering is not readiness

`Dependent` only controls launch order. A dependency that does not implement
`ReadyNotifier` is considered ready as soon as its goroutine has been
launched, not once it has opened a socket or established a database connection.

If `api` cannot operate before `database` is usable, `database` should expose a
channel and close it only after connecting, while `api` declares the name as a
dependency:

```go
func (s *Database) Ready() <-chan struct{} { return s.readyCh }

func (s *API) Dependencies() []string { return []string{"database"} }
```

Services in the same dependency group start concurrently. The manager waits
for every `ReadyNotifier` in that group before starting the next group.

The wait ends early in two cases. If a service fails with a non-allowed error before it is ready, `Run` returns that error and later groups are never started. If a service's `Run` returns for good (an allowed failure, or a clean exit) before it signals ready, the manager logs a warning and stops waiting for it, and later groups still start.

A dependency that is not running in this process, because it is not registered
or `SERVICES_ENABLED` left it out, is logged as a warning and skipped. That
lets the same code run with every service in one process or with services
split across deployments. A dependency that runs elsewhere still needs its own
connection, retry and readiness handling in the service that calls it.

### Failure and retry policy

The first service that fails for good, without `AllowedFailure`, shuts the
application down. Failures that arrive during that shutdown are logged. A panic
in `Run` becomes an error and follows the same rules.

A `Run` that returns `nil` ends that service only. The manager logs it and the
other services keep running; when every service has returned, the process still
waits for a signal. Put a one-off job in a `Commander` command or in
`cmd/commands.go` instead.

`AllowedFailure` is not a retry setting: a service can be both `Retryable` and
`AllowedFailure`, in which case it retries first and becomes non-fatal only
after the retry budget is exhausted. Use this only for work whose disappearance
does not make the process lie about its health.

## Selectively run services

`SERVICES_ENABLED` picks services by `ServiceName` before any of them is
built:

```bash
SERVICES_ENABLED=api,price-worker ./build/my-service run
```

An empty or unset value runs every registered service. A selected service does
not automatically pull in its dependencies; include the services needed for
that local run. If filtering yields no services, startup returns an error.

## Commands and lifecycle hooks

`Commander` is for commands owned by one service. It lets an operator invoke,
for example, `my-service database migrate` without building the rest of the
application's service graph. Every registered service appears as a command in
`--help`; running one that does not implement `Commander` fails with
`service has no commands`.

`cmd/commands.go` is for app-level Cobra commands that do not belong to a
specific service. `cmd/init.go` is the extension point for application hooks:

```go
func init() {
	app.GetInstance().OnPreRun(func(ctx context.Context) {
		// start bounded application-level work here
	})

	app.GetInstance().OnPostStop(func(ctx context.Context) {
		// flush/close after services have stopped
	})
}
```

Hooks run sequentially in registration order. Keep them short and make any
goroutine they start respect the supplied context.

## Shutdown path

`pkg/runner` listens for `SIGINT` and `SIGTERM`, as well as the supplied parent
context and application errors. It then creates a shutdown context bounded by
`RUNNER_SHUTDOWNTIMEOUT` (default `10s`) and calls application stop.

The service manager cancels every running service, then stops dependency groups
in reverse order; services within each group stop concurrently. Each `Stop`
gets a context that is not cancelled, limited to the manager's 30 seconds per
service, so a drain such as `http.Server.Shutdown(ctx)` has time to finish.

On a signal or a cancelled parent context, the runner's deadline also caps the
whole process, and it is normally shorter: 10 seconds by default. Set
`RUNNER_SHUTDOWNTIMEOUT` high enough for legitimate cleanup, but do not make
shutdown unbounded. When a service fails, the manager stops the other services
before the runner learns of the failure, so only the 30-second limit for each
`Stop` applies.

## Next

Read [architecture](architecture.md) for running services in one process or
splitting them across deployments, and [development](development.md) for
testing them.
