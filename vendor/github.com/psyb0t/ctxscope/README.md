# ctxscope

[![Go Reference](https://pkg.go.dev/badge/github.com/psyb0t/ctxscope.svg)](https://pkg.go.dev/github.com/psyb0t/ctxscope)
[![CI](https://github.com/psyb0t/ctxscope/actions/workflows/pipeline.yml/badge.svg?branch=master)](https://github.com/psyb0t/ctxscope/actions/workflows/pipeline.yml)
[![coverage](https://raw.githubusercontent.com/psyb0t/ctxscope/badges/coverage.svg)](https://github.com/psyb0t/ctxscope/actions/workflows/pipeline.yml)
[![version](https://raw.githubusercontent.com/psyb0t/ctxscope/badges/version.svg)](https://github.com/psyb0t/ctxscope/tags)
[![license](https://raw.githubusercontent.com/psyb0t/ctxscope/badges/license.svg)](LICENSE)
[![imported by](https://raw.githubusercontent.com/psyb0t/ctxscope/badges/importers.svg)](https://github.com/psyb0t/ctxscope/blob/badges/importers.md)

Stick attributes on a `context.Context`, get them on every log line under that context. No threading `request_id` through nine function signatures to get it onto one log call at the bottom.

Stdlib plus [`ctxerrors`](https://github.com/psyb0t/ctxerrors). That's the whole dependency list, and it stays that way — see [why](#why-it-imports-nothing).

**Status:** active. Extracted from `common-go` and stable — the API has not changed since, only the package name.

## Contents

- [What the fuck does it do?](#what-the-fuck-does-it-do)
- [Two tiers, and the difference matters](#two-tiers-and-the-difference-matters)
- [Getting them onto the line — pick one](#getting-them-onto-the-line--pick-one)
- [Crossing a process boundary](#crossing-a-process-boundary)
- [Wiring it at the edges](#wiring-it-at-the-edges)
- [Why it imports nothing](#why-it-imports-nothing)
- [The full surface](#the-full-surface)
- [Design notes](#design-notes)
- [Was this in common-go?](#was-this-in-common-go)
- [Dev](#dev)
- [License](#license)

## What the fuck does it do?

You set an attribute once, at the boundary:

```go
ctx = ctxscope.Set(ctx, ctxscope.Attr("request_id", requestID))
```

Every log line emitted under that ctx — anywhere, however deep — carries `request_id`. Nothing in between has to know it exists.

```bash
go get github.com/psyb0t/ctxscope
```

## Two tiers, and the difference matters

| call | for | crosses a process hop? |
|---|---|---|
| `SetGlobal` | commit, service, region — facts about the **binary** | never |
| `Set` | request_id, user_id — facts about the **work** | yes, via `ToJSON`/`FromJSON` |

Putting a process fact in `Set`'s tier isn't a style slip, it's a bug: it would ride along to the next service and overwrite that service's own value, and now its logs name the wrong deploy. The tiers are split precisely so that can't happen.

Both get merged when a line is logged. The context tier wins collisions.

## Getting them onto the line — pick one

### The handler (recommended)

Install it once at startup and you're done:

```go
base := slog.NewJSONHandler(os.Stdout, nil)
slog.SetDefault(slog.New(ctxscope.NewHandler(base)))
```

Now plain slog works:

```go
slog.InfoContext(ctx, "order placed", "order_id", id)
// {"level":"INFO","msg":"order placed","order_id":"x","request_id":"abc","service":"api"}
```

This is the one nobody can forget, and the only one that reaches code which has never heard of this package — a library logging through `slog.InfoContext` gets your `request_id` for free.

**Use the `Context`-suffixed calls.** `slog.Info` hands the handler a background context, so the line still gets the global tier — that never came from a context anyway — but none of the per-context tier. Your `service` shows up, your `request_id` doesn't. That's slog's contract, not ours.

### GetLogger

Or skip the handler and pull a logger with the attributes already baked on:

```go
logger := ctxscope.GetLogger(ctx)
logger.Info("order placed", "order_id", id)
```

Call it *where you log*, not once at the top — a logger is a value, so one fetched before a later `Set` doesn't have what you added.

**These two are alternatives, not layers.** `GetLogger` applies the scope itself, so calling it under an installed handler emits every attribute twice. Pick one per project.

## Crossing a process boundary

A `*slog.Logger` can't cross a process boundary. Data can — which is the entire reason the scope is a map:

```go
data, err := ctxscope.ToJSON(ctx)        // outbound; context tier only, never globals
ctx, err := ctxscope.FromJSON(ctx, data) // inbound, far side
```

One call re-seeds the whole map — not a `Set` per key. Works for an HTTP header, a NATS message header, a Temporal `ContextPropagator`, or a subprocess env var.

Two things worth knowing before they surprise you:

- `ToJSON` serializes the **context tier only**. The receiving process keeps its own `commit`/`service`. That's the point of the split.
- JSON has one number type, so an int sent as `42` comes back as `float64(42)`. Fine for log and wire material; a trap only if you type-assert it.

## Wiring it at the edges

Set attributes once where work enters the process. Everything downstream inherits them.

**HTTP server** — stamp the request id, then hand the enriched context onward:

```go
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = newID()
		}

		ctx := ctxscope.Set(r.Context(), ctxscope.Attr("request_id", id))

		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

**Outbound to a queue** — the map goes on the wire, not the logger:

```go
data, err := ctxscope.ToJSON(ctx)
if err != nil {
	return ctxerrors.Wrap(err, "marshal scope")
}

msg.Header.Set("x-scope", string(data))
```

**Receiving side** — re-seed the whole map in one call, and don't drop the message if it's malformed:

```go
ctx, err := ctxscope.FromJSON(context.Background(), []byte(msg.Header.Get("x-scope")))
if err != nil {
	ctx = context.Background() // a bad header is not a reason to lose the work
}
```

**Startup** — process facts go in the global tier, so they never travel:

```go
ctxscope.SetGlobal(
	ctxscope.Attr("service", "api"),
	ctxscope.Attr("commit", commitSHA),
)
```

## Why it imports nothing

Transport adapters — the Temporal propagator, the NATS injector, HTTP middleware — live next to their transport and depend on this package. Never the other way around.

If this package imported the Temporal SDK, every consumer would drag a workflow engine in behind it. Stdlib-only means importing it costs nothing, and that's a constraint, not a coincidence.

## The full surface

| function | does |
|---|---|
| `Set(ctx, ...Attribute) context.Context` | add attributes to the context tier |
| `Remove(ctx, ...string) context.Context` | drop keys from it |
| `Get(ctx) Scope` | read it back, as a copy |
| `SetGlobal(...Attribute)` | add to the process tier |
| `RemoveGlobal(...string)` | drop from it |
| `GetGlobal() Scope` | read it back, as a copy |
| `GetLogger(ctx) *slog.Logger` | a logger with both tiers applied |
| `NewHandler(slog.Handler) *Handler` | the handler alternative to `GetLogger` |
| `ToJSON(ctx) ([]byte, error)` | context tier out to the wire |
| `FromJSON(ctx, []byte) (context.Context, error)` | and back in on the far side |
| `Attr[T Value](key, value) Attribute` | build one attribute |

Four exported types: **`Handler`** (implements `slog.Handler`), **`Scope`** (`map[string]any`), **`Attribute`**, **`Value`** (the type constraint).

That is the entire exported API. If you're reaching for a helper not listed above, it doesn't exist.

Three properties worth knowing:

- **`Attr` is generic over strings, bools, ints and floats.** Anything wider has no sane rendering as either a log attribute or JSON, so it won't compile. The constraint sits on `Attr` rather than on `Attribute`'s field because a Go constraint interface can't be used as a field type, and one variadic call can't mix `Attribute[string]` with `Attribute[int]`.
- **`Get` and `GetGlobal` hand back copies.** Mutating what you get back cannot corrupt the context or the process tier.
- **Concurrency is handled.** The global tier is an atomic pointer to an immutable map — readers never lock, writers copy-and-swap. The context tier needs no locking at all: a `context.Context` is immutable, so `Set` returns a new one rather than mutating.

## Design notes

A few decisions that look arbitrary until they aren't:

- **The map is the only state; the logger is derived when you ask.** Writing a logger onto the context instead would make `Remove` impossible — slog has no way to un-`With` an attribute — and setting a key twice would emit it twice.
- **Scope attributes land at the record's top level, even under `WithGroup`.** A `request_id` nested inside a group is not the `request_id` your log queries match on. `NewHandler` replays `WithAttrs`/`WithGroup` calls *after* applying the scope to make that true.
- **Attributes are sorted by key**, so field order is stable across lines and diffs cleanly.

## Was this in common-go?

Yes — this was `github.com/psyb0t/common-go/scope`. It moved out because it's a foundational primitive that shouldn't share a release cadence with a module that also carries gorm, echo, NATS and the Temporal SDK.

The API is unchanged apart from the package name, plus the new handler:

```go
// before
import "github.com/psyb0t/common-go/scope"
scope.Set(ctx, scope.Attr("request_id", id))

// after
import "github.com/psyb0t/ctxscope"
ctxscope.Set(ctx, ctxscope.Attr("request_id", id))
```

## Dev

```bash
make test           # go test -race ./...
make test-coverage  # + coverage gate
make lint-fix       # go fix + golangci-lint --fix
```

`make help` lists the rest.

## License

MIT. See [LICENSE](LICENSE).

See [CHANGELOG.md](CHANGELOG.md) for release notes.
