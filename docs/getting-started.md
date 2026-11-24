---
title: "Getting started"
description: "Turn a fresh servicepack clone into your own Go project: make it yours, add a service, configure it and run it locally or as one binary."
sidebar_position: 2
---

# Getting started

servicepack is the starting point for a Go application with several
long-running parts: an API, workers, feeds, schedulers. Clone it, make the
clone yours, and put your services in `internal/pkg/services/`.

It is a template you copy, not a library you add to an existing project with
`go get`.

You need Docker and Git. Every build, test and lint target runs Go inside a
container, so the host Go version does not matter for those.

## 1. Make a fresh clone yours

```bash
git clone https://github.com/psyb0t/servicepack.git my-service
cd my-service
make own MODNAME=github.com/yourname/my-service
```

Run `make own` once, in a fresh clone. It:

1. removes the `example-*` services and keeps `hello-world`;
2. removes what belongs to servicepack itself: its agent instructions
   (`AGENTS.md`, `CLAUDE.md`, `.claude`, `.agents`), its documentation
   (`docs/`, `website/` and the docs workflow), its `CHANGELOG.md`, its
   `.github/FUNDING.yml`, its mirror-and-archive, issue-pull and
   collaborators-only workflows, and the `Makefile` lines that include the
   documentation site's targets;
3. removes `.git`, `go.mod`, `go.sum` and `vendor/`;
4. writes a new `go.mod` for `MODNAME` that keeps the framework's pinned
   dependencies and tool directives;
5. replaces `github.com/psyb0t/servicepack` with your module path in every
   remaining `.go`, `.mod` and `.md` file;
6. replaces `README.md` with a stub;
7. writes `servicepack.version`, if it does not exist, with the servicepack
   tag you cloned, or its commit when the clone is not on a tag;
8. regenerates service registration, then runs `make dep`; and
9. runs `git init` on `main` and makes an initial commit, so Git needs a user
   name and email.

Your binary is named after the last segment of `MODNAME`: for
`github.com/yourname/my-service`, it is `my-service`.

### Then finish making it yours

Two files still need your attention before your first push:

- `.github/workflows/pipeline.yml` is servicepack's CI. Among other jobs, it
  publishes the agent skill that `make own` just removed and writes coverage
  badges. Rewrite it for your project. Updates never touch it.
- `Dockerfile.dev` is a copy of the framework's development image. Delete it
  unless you change it; the Make targets then use `Dockerfile.servicepack.dev`,
  which updates bring forward. Keep `Dockerfile`, because the test harness
  builds it, and see [framework updates](framework-updates.md#the-dockerfiles)
  for keeping it current.

## 2. Add a service

```bash
make service NAME=price-worker
```

That creates `internal/pkg/services/price-worker/priceworker.go` and regenerates
`internal/pkg/services/services.gen.go`, which registers every service package
with the manager. Never edit the generated file. The package rules it relies on
are in the [reference](reference.md#service-package-rules).

The generated service waits for cancellation and does nothing else:

```go
func (s *PriceWorker) Run(ctx context.Context) error {
	ctx = ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))
	logger := ctxscope.GetLogger(ctx)
	logger.Info("starting service")

	<-ctx.Done()
	logger.Info("service context cancelled")

	return nil
}
```

Replace the wait with the real work, and keep returning when `ctx` is
cancelled. A non-nil error from `Run` normally shuts the application down.
Returning `nil` ends only that service; the process keeps running until it gets
a signal, so a one-off job belongs in a [command](services-and-lifecycle.md#commands-and-lifecycle-hooks),
not a service. [Services and lifecycle](services-and-lifecycle.md) has the
retry and failure rules.

After you add, remove or rename a service by hand, regenerate registration:

```bash
make service-registration
```

## 3. Configure and run it

Each service loads its own typed config with `gonfiguration`:

```go
type Config struct {
	Endpoint string        `env:"PRICEWORKER_ENDPOINT"`
	Interval time.Duration `env:"PRICEWORKER_INTERVAL" default:"15s"`
}
```

Parse it in `New` and return a wrapped error if it fails, instead of calling
`os.Getenv`:

```go
func New() (*PriceWorker, error) {
	cfg := Config{}

	if err := gonfiguration.Parse(&cfg); err != nil {
		return nil, ctxerrors.Wrap(err, "parse price-worker config")
	}

	return &PriceWorker{config: cfg}, nil
}
```

Build and run the binary:

```bash
make build
./build/my-service run
```

`run` starts every registered service. Without a subcommand the binary prints
its help. The framework's own settings, such as `LOG_LEVEL` and
`SERVICES_ENABLED`, are listed in the [reference](reference.md#framework-environment-variables):

```bash
SERVICES_ENABLED=price-worker LOG_LEVEL=debug ./build/my-service run
```

For a container, build the production image and pass `run` to it, since the
image's default command is `--help`:

```bash
make docker-build
docker run --rm my-service run
```

`make run-dev` builds the development image and runs the application inside it
with the race detector.

## Where your changes go

| You are changing | Put it here |
| --- | --- |
| A business service | `internal/pkg/services/<name>/` |
| Service config | That service's `Config` struct |
| Startup and shutdown hooks | `cmd/init.go` |
| Application-wide commands | `cmd/commands.go` |
| Build behavior | `Makefile`, or a script override in `scripts/make/` |
| Container images | `Dockerfile`, `Dockerfile.dev` |
| Tests that need real infrastructure | `tests/` |
| Your documentation and changelog | `docs/`, `CHANGELOG.md` |

Directories you add yourself are yours too. The framework's own files outside
these paths are replaced on every update; [framework updates](framework-updates.md#what-an-update-changes)
lists exactly what an update changes and what it leaves alone.

Next: the [service lifecycle](services-and-lifecycle.md), then the
[development workflow](development.md).
