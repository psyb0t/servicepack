# Service manager

`service-manager` is the framework's in-process supervisor. It owns factory
registration, service construction, dependency ordering, readiness gates,
retry/failure policy, service-scoped logging, and ordered shutdown. `App` owns
the process-level lifecycle around it; `pkg/runner` owns signals and the outer
deadline. See [the architecture overview](../../../docs/architecture.md) for
how those pieces connect.

## Public contract

The required `Service` contract is deliberately small:

```go
type Service interface {
	Name() string
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}
```

`Name` is the map key for registration, filtering, dependency resolution, and
service log scope. Names must be stable enough that a deployment configuration
can refer to them through `SERVICES_ENABLED`.

`Run` is called in a managed goroutine. It must return when `ctx.Done()` is
closed. A clean return is logged as a clean service exit; a non-nil error goes
through retry/failure handling. A panic is recovered and represented as
`ErrServicePanic` so it cannot tear down the process without context.

`Stop` is called after the manager cancels its run context. It receives a
bounded shutdown context and should close resources, not start unbounded work.
The manager logs a `Stop` error but has no return path for it; report only
useful errors there and make cleanup idempotent.

## Registration is factories, not instances

`Register(name, factory)` stores a `ServiceFactory`. It does not construct a
service. `Run` constructs all selected factories; `Commands` constructs only
the service named by the invoked command. That is why constructors should own
configuration parsing and client setup, while package initialization should
remain side-effect free.

Generated `internal/pkg/services/services.gen.go` calls `Register` for each
discovered service. It is generation output, not a hand-maintained registry.
Use `make service-registration` after modifying services.

## Startup graph

`resolveOrder` builds a directed graph from `Dependent.Dependencies()` and
topologically sorts it into groups. Every service in a group starts
concurrently; group N+1 starts only after group N has been launched and its
ready-notifying services have become ready.

```
database ─┐
cache ────┴─ group 0 (concurrent)
    │
api ──────── group 1
    │
worker ───── group 2
```

The key distinction:

- a `Dependent` relationship orders *launch*;
- `ReadyNotifier` controls when dependents may advance beyond that group.

Without `ReadyNotifier`, launch is treated as readiness. If `api` needs a
database that is accepting connections—not merely a goroutine that has been
scheduled—have `database.Ready()` return a channel and close it after the
connection is established.

Missing dependency names are treated as external to this process: the manager
logs a warning and skips that edge. Cycles among registered services return
`ErrCyclicDependency`; zero selected services return `ErrNoEnabledServices`.

## Failure policy

`Retryable` supplies an attempt budget and delay. The manager calls `Run` once
plus `MaxRetries()` additional times; it exits a delay early when the context
is cancelled. A non-positive delay retries immediately.

After the retry budget is exhausted:

- an `AllowedFailure` whose `IsAllowedFailure()` returns true is logged and
  allowed to disappear;
- every other failure is sent as the terminal error for the process.

Only the first terminal failure is delivered. Concurrent later failures are
logged rather than blocking on the already-full error channel, because the
first failure is already causing application shutdown. This is an intentional
anti-deadlock semantic, not lost success information.

## Context and logging conventions

The manager scopes every service run, command, registration log, and stop path
with `ctxscope.Attr("service", service.Name())`. Services should keep deriving
from the passed context rather than replacing it with `context.Background()`.
That preserves cancellation plus the process-wide `binary`/`commit` fields set
by `cmd/main.go`.

Use `Add` for straightforward registration in tests and `AddContext` when the
registration log needs a caller-provided scope. Both overwrite a prior service
with the same name in the in-memory service map; registering duplicate names is
therefore a bug in the caller, not a supported way to run two instances.

## Stop behavior

`Stop` cancels the run context once, then walks the resolved start groups in
reverse order. It stops each group concurrently and waits for that group before
moving to the preceding one. The local per-service timeout is 30 seconds.

In the normal application path, the runner supplies a shorter whole-process
deadline (10 seconds by default), so services must respect the passed context
promptly. The runner can return on its outer deadline even if a broken `Stop`
implementation ignores cancellation; do not rely on the manager's timer as a
way to make non-cooperative cleanup safe.

## Testing this package

Tests need singleton isolation. Start each independent scenario by resetting
the manager and avoid the generated global registration unless that behavior is
the subject under test:

```go
servicemanager.ResetInstance()
manager := servicemanager.GetInstance()
manager.ClearServices()
manager.Add(fakeService)
```

Use controlled channels for readiness, attempts, and stop observation. Cover
both ordering and the absence of readiness: they are different guarantees.
Run the suite through `make test`; the framework test workflow uses Docker and
the race detector. See [development](../../../docs/development.md) for the
full test/coverage contract.
