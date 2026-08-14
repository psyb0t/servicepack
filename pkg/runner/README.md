# Runner

`pkg/runner` turns an application's `Run(context.Context) error` and
`Stop(context.Context) error` pair into a signal-aware process lifecycle. The
application entry point calls `runner.RunContext(cmd.Context(), app)`; use this
package rather than adding a second signal handler in a service.

## Contract

```go
type Runnable interface {
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}
```

`Run(runnable)` is the compatibility helper for a background parent context.
`RunContext(parent, runnable)` is the normal entry point when the caller has a
meaningful parent context (as Cobra does in `cmd/main.go`). The runner derives
an application context from that parent and starts `Runnable.Run` once.

## What starts shutdown

The runner waits for the first of:

- cancellation of the supplied parent context;
- `SIGINT`, `SIGTERM`, or `os.Interrupt`;
- a return from `Runnable.Run`.

A non-nil `Run` error is logged and returned with context after shutdown. A
nil return also initiates cleanup—it means the application is done, not that
the runner should hang forever.

## Deadline semantics

`RUNNER_SHUTDOWNTIMEOUT` is parsed through `gonfiguration` and defaults to
`10s`. Once shutdown begins, the runner calls `Stop` with a fresh timeout
context based on `context.WithoutCancel(applicationContext)`. That lets cleanup
run even though the application context was cancelled to tell `Run` methods to
exit.

If `Stop` does not complete before the deadline, `RunContext` returns
`ErrShutdownTimeout`. The caller can treat that as an unhealthy shutdown and
the process is free to exit rather than wait indefinitely.

The app's service manager has its own per-service stopping mechanics, but the
runner deadline is the process-level budget in the standard binary. Configure
one value that genuinely fits resource draining and keep service `Stop`
methods cooperative with their context.

## Logging and errors

The runner emits structured start, signal, cancellation, shutdown, and timeout
logs through `ctxscope`. It wraps configuration, run, and stop errors with
`ctxerrors` so callers retain the useful operation context.

`Stop` errors are joined with the original `Run` error where both exist. A
timeout returns the explicit sentinel because callers need to distinguish an
incomplete shutdown from the application error that started it.

## Testing

`runner_test.go` drives the runner with fake `Runnable` implementations and
controlled contexts. Test parent cancellation, run errors, stop errors, and
the timeout branch without sending host signals. Run it through `make test`,
which uses the Docker development environment and the race detector.

For the surrounding application/service orchestration, see the
[service manager README](../../internal/pkg/service-manager/README.md) and
[lifecycle overview](../../docs/services-and-lifecycle.md).
