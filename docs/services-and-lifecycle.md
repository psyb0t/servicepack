# Services and lifecycle

Servicepack supervises services that live in the same process. It provides
startup ordering, readiness gates, failure policy, service-scoped logs, and
one coordinated shutdown path. It does not turn in-process calls into a
network protocol or solve distributed-systems concerns for services you split
out later.

For the implementation-level rules and test setup, see the
[service-manager deep dive](../internal/pkg/service-manager/README.md) and
[runner deep dive](../pkg/runner/README.md).

## The required contract

Every service implements:

```go
type Service interface {
	Name() string
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}
```

`Name` must be the stable name used by `SERVICES_ENABLED` and dependency
declarations. `Run` must return when its context is cancelled. `Stop` is where
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
builds, `commit` attributes. Do not replace those with ad-hoc global logging.

## Optional contracts

Implement any combination of these interfaces on the same service.

| Interface | Effect |
| --- | --- |
| `Retryable` | Retry a failed `Run` up to `MaxRetries()` times, waiting `RetryDelay()` between attempts. `0` means no retry. |
| `AllowedFailure` | After retries are exhausted, log the failure and leave the rest of the application running. |
| `Dependent` | Start after named services in this binary. Cycles fail startup. |
| `ReadyNotifier` | Block later dependency groups until the service closes `Ready()`. |
| `Commander` | Add `./build/<app> <service> <subcommand>` commands, instantiating only that service. |

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

Dependency names that are not registered in the current process are logged and
ignored. That makes it possible to use the same business design in a composed
local binary and in a separately deployed setup, but it also means an external
database, queue, or microservice still needs its own connection/retry/readiness
handling.

### Failure and retry policy

The first non-allowed terminal failure ends the application. Later concurrent
failures are logged but do not block shutdown. A panic in `Run` is converted to
a service error and follows that same policy.

`AllowedFailure` is not a retry setting: a service can be both `Retryable` and
`AllowedFailure`, in which case it retries first and becomes non-fatal only
after the retry budget is exhausted. Use this only for work whose disappearance
does not make the process lie about its health.

## Selectively run services

`SERVICES_ENABLED` filters service factories before instantiation:

```bash
SERVICES_ENABLED=api,price-worker ./build/my-service run
```

An empty or unset value runs every registered service. A selected service does
not automatically pull in its dependencies; include the services needed for
that local run. If filtering yields no services, startup returns an error.

## Commands and lifecycle hooks

`Commander` is for commands owned by one service. It lets an operator invoke,
for example, `my-service database migrate` without building the rest of the
application's service graph.

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
in reverse order; services within each group stop concurrently. The manager's
own per-service stop default is `30s`, but the runner's parent deadline is
normally shorter, so the whole process is usually capped by the runner's 10
seconds. Set `RUNNER_SHUTDOWNTIMEOUT` high enough for legitimate cleanup, but
do not make shutdown unbounded.

## Local composition versus microservices

Keep services together when shared release cadence, shared local debugging,
and in-process coordination are the simplest honest design. Split them when
they need independent scaling, ownership, security boundaries, or failure
isolation. The split point means replacing in-process contracts with explicit
network/message contracts, not pretending a `Dependent` declaration will order
another deployment.

Read [architecture](architecture.md) for the process-level picture and
[development](development.md) for testing this in Docker.
