# Getting started

Servicepack is the starting point for a Go application with multiple
long-running things to run: an API, workers, feeds, schedulers, migration
commands, whatever. Clone it, make the clone yours, and put your services in
`internal/pkg/services/`.

It is not intended to be imported wholesale into an existing project with
`go get`.

## 1. Make a fresh clone yours

```bash
git clone https://github.com/psyb0t/servicepack.git my-service
cd my-service
make own MODNAME=github.com/yourname/my-service
```

Run that once, only in a disposable fresh clone. `make own`:

1. removes shipped `example-*` services but leaves `hello-world`;
2. removes the clone's `.git`, module manifests, and `vendor/` directory;
3. recreates the module using the supplied `MODNAME` while retaining the
   framework's pinned dependency and tool declarations;
4. rewrites framework imports to your module path;
5. replaces this README with a project stub;
6. records the framework revision in `servicepack.version` if it does not
   already exist;
7. runs dependency and service-registration targets; and
8. initializes a new Git repository on `main` and creates an initial commit.

It needs Git identity configured well enough for that initial commit. Its
normal dependency and generation targets run in the Docker development image;
it does not reject an older host Go toolchain.

Your binary name is the final segment of `MODNAME`: for
`github.com/yourname/my-service`, it is `my-service`.

## 2. Add a service

```bash
make service NAME=price-worker
```

That creates `internal/pkg/services/price-worker/priceworker.go` and then
regenerates `internal/pkg/services/services.gen.go`. The generated file finds
the service factories at runtime; do not hand-edit it.

The scaffold is intentionally boring:

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

Replace the wait with the real work, but keep cancellation part of the
contract. `Run` returning a non-nil error normally starts application
shutdown; see [services and lifecycle](services-and-lifecycle.md) for the
exceptions and retry rules.

When you add, remove, rename, or materially change a service implementation,
regenerate registration:

```bash
make service-registration
```

## 3. Configure and run it

Generated services load their own typed config with `gonfiguration`:

```go
type Config struct {
	Endpoint string        `env:"PRICEWORKER_ENDPOINT"`
	Interval time.Duration `env:"PRICEWORKER_INTERVAL" default:"15s"`
}
```

Parse configuration in `New`, fail with context, and do not reach straight for
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

Build and run:

```bash
make build
./build/my-service run
```

For the everyday development loop, `make run-dev` builds the development image
and runs the application with the race detector. It is useful for the shipped
examples; a made-own project usually needs its own config and services first.

## Framework configuration

These names belong to the framework itself. Your services own their own
configuration names.

| Variable | Meaning | Default |
| --- | --- | --- |
| `LOG_LEVEL` | `debug`, `info`, `warn`, or `error` logging threshold. | Handler default |
| `LOG_FORMAT` | `json` or `text` output. | Handler default |
| `LOG_ADD_SOURCE` | Include source location in log records. | Handler default |
| `ENV` | Environment selected by `goenv`. | `prod` |
| `RUNNER_SHUTDOWNTIMEOUT` | Whole-application graceful shutdown deadline. | `10s` |
| `SERVICES_ENABLED` | Comma-separated in-process service allowlist. Empty/unset means all registered services. | all |

Example:

```bash
SERVICES_ENABLED=price-worker,api LOG_LEVEL=debug ./build/my-service run
```

## Where your changes go

| You are changing | Put it here |
| --- | --- |
| A business service | `internal/pkg/services/<name>/` |
| Service-specific config | That service's `Config` struct and project env documentation |
| Startup/shutdown extensions | `cmd/init.go` hooks |
| Standalone application commands | `cmd/commands.go` |
| Project build behavior | `Makefile` or `scripts/make/` overrides |
| Project Docker behavior | `Dockerfile`, `Dockerfile.dev` |

The framework-owned directories are deliberately replaceable by
`make servicepack-update`: `internal/app/`,
`internal/pkg/service-manager/`, `pkg/runner/`, `cmd/main.go`,
`Makefile.servicepack`, `scripts/make/servicepack/`, and
`Dockerfile.servicepack*`. Do not put application-specific changes there.

Next: learn the [service lifecycle](services-and-lifecycle.md), then the
[Docker-first development workflow](development.md).
