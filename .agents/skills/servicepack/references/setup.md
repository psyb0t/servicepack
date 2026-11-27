# Setup

## This is a template, not a `go get` dependency

There is no `go get github.com/psyb0t/servicepack` step. `internal/`, `cmd/main.go`, and `pkg/runner/` are not designed to be imported into an existing module. They're the skeleton of YOUR module. You clone the repo, rewrite the module path, and build on top.

```bash
git clone https://github.com/psyb0t/servicepack
cd servicepack
make own MODNAME=github.com/yourname/yourproject
```

### What make own changes

`make own` cannot be undone on the clone. Run it only when the user asked to start a new project, and only in a directory you just cloned. Read `scripts/make/servicepack/own.sh` first to see the exact steps. It:

- removes the example services (it keeps `hello-world`)
- deletes servicepack's own `docs/`, `website/`, `CHANGELOG.md`, agent files and publishing workflows
- deletes `.git`, `go.sum`, `go.mod` and `vendor/`, then rebuilds `go.mod` with your module name and the framework's pinned dependencies
- rewrites the old module path to yours in every `.go`, `.mod` and `.md` file
- replaces `README.md` with a stub for your project name
- runs the Docker-backed registration and dependency targets, which build the dev image and download Go modules
- runs `git init` on `main` and creates the initial commit

Your binary name is derived from `go.mod`'s module name (last path segment) at build time, no separate app-name config.

## Go and Docker requirements

`go.mod` declares Go `1.26.11`. The supported Make targets run that project
toolchain in the development/build Docker images, so `make own` does not reject
an older host Go installation. Docker must be available for normal dependency,
generation, build, lint, and test work. Keep the `go.mod` declaration aligned
with the framework version you are updating to; do not treat an arbitrary host
Go version as the project contract.

## Module path / import path

Framework code imports as `github.com/psyb0t/servicepack/...` before you run `make own`; afterwards every import is rewritten to your module path, e.g. `github.com/yourname/yourproject/internal/app`.

Core framework packages you'll reference directly:

| Package | Purpose |
|---|---|
| `<your-module>/internal/app` | `App` singleton: `GetInstance()`, `OnPreRun`, `OnPostStop` |
| `<your-module>/internal/pkg/service-manager` (import alias `servicemanager`) | `Service`/`Retryable`/`AllowedFailure`/`Dependent`/`ReadyNotifier`/`Commander` interfaces, `GetInstance()` |
| `<your-module>/pkg/runner` | `runner.RunContext(ctx, runnable)`: signal handling + graceful shutdown with a caller-supplied parent context; `runner.Run(runnable)` remains the background-context compatibility helper |
| `<your-module>/internal/pkg/services` | generated `services.Init()` (via `services.gen.go`) |

Third-party deps pulled in by the framework itself (already in `go.mod`, vendored):

- `github.com/psyb0t/ctxerrors`: error wrapping with file/line/function capture
- `github.com/psyb0t/ctxscope`: contextual structured logging
- `github.com/psyb0t/goenv`: `dev`/`prod` environment detection
- `github.com/psyb0t/gonfiguration`: env-var config parsing via struct tags
- `github.com/psyb0t/slogging`: `log/slog` handler wiring (`slogging/slogconf`)
- `github.com/spf13/cobra`: CLI command tree

## Skip the clone entirely: try it in Docker first

```bash
git clone https://github.com/psyb0t/servicepack
cd servicepack
make run-dev
```

Builds a dev image and runs the shipped example services (`hello-world`, `example-database`, `example-api`, `example-migrator`, `example-optional`, `example-flaky`, `example-crasher`, `example-nested/http`, `example-nested/grpc`) so you can see retries, dependencies, allowed failures, readiness gating, and a crash-everything failure in action before committing to `make own`.

## Make targets that need Docker access

`make test`, `make test-integration` and `make test-coverage` run the tests in the dev container with the host Docker socket mounted and host networking, because the integration tests start sibling containers through testcontainers. That gives the test process control of the host Docker daemon. Run them only when the user asked for tests, and only on a machine where that is acceptable. The other targets the skill uses run in the dev container without the socket.

## Environment variables the framework itself reads

```bash
LOG_LEVEL=debug                 # debug, info, warn, error
LOG_FORMAT=json                 # json, text
LOG_ADD_SOURCE=true              # include file:line in log records
ENV=dev                          # dev, prod (default: prod), via goenv
RUNNER_SHUTDOWNTIMEOUT=10s        # graceful shutdown deadline (default: 10s)
SERVICES_ENABLED=svc1,svc2        # comma-separated allowlist; empty/unset = run all
```

Your own services define their own env vars via `gonfiguration` struct tags, see the worked example below.

## Fuller worked example: retryable, dependent, ready-notifying service

Scaffolded with `make service NAME=example-db`, then hand-extended past the generated skeleton to show the three most commonly combined optional interfaces:

```go
// internal/pkg/services/example-db/example_db.go
package exampledb

import (
	"context"
	"time"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxscope"
	"github.com/psyb0t/gonfiguration"
)

const ServiceName = "example-db"

type Config struct {
	DSN string `env:"EXAMPLEDB_DSN"`
}

type ExampleDB struct {
	config  Config
	readyCh chan struct{}
}

func New() (*ExampleDB, error) {
	cfg := Config{}

	gonfiguration.SetDefaults(map[string]any{
		"EXAMPLEDB_DSN": "postgres://localhost/example",
	})

	if err := gonfiguration.Parse(&cfg); err != nil {
		return nil, ctxerrors.Wrap(err, "failed to parse example-db config")
	}

	return &ExampleDB{
		config:  cfg,
		readyCh: make(chan struct{}),
	}, nil
}

func (s *ExampleDB) Name() string {
	return ServiceName
}

// MaxRetries + RetryDelay satisfy Retryable: restart up to 2 times,
// waiting 2s between attempts, before giving up.
func (s *ExampleDB) MaxRetries() int {
	return 2
}

func (s *ExampleDB) RetryDelay() time.Duration {
	return 2 * time.Second
}

// Ready satisfies ReadyNotifier: dependents wait on this channel
// before their own Run() is started.
func (s *ExampleDB) Ready() <-chan struct{} {
	return s.readyCh
}

func (s *ExampleDB) Run(ctx context.Context) error {
	ctx = ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))
	ctxscope.GetLogger(ctx).Info("starting service")

	// connect, migrate, whatever "actually ready" means for you
	close(s.readyCh)

	<-ctx.Done()

	return nil
}

func (s *ExampleDB) Stop(ctx context.Context) error {
	serviceCtx := ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))
	ctxscope.GetLogger(serviceCtx).Info("stopping service")

	return nil
}
```

```go
// internal/pkg/services/example-api/example_api.go
package exampleapi

import (
	"context"

	"github.com/psyb0t/ctxscope"
)

const ServiceName = "example-api"

type ExampleAPI struct{}

func New() (*ExampleAPI, error) {
	return &ExampleAPI{}, nil
}

func (s *ExampleAPI) Name() string {
	return ServiceName
}

// Dependencies satisfies Dependent: example-api waits for
// example-db to signal ready before its Run() starts.
func (s *ExampleAPI) Dependencies() []string {
	return []string{"example-db"}
}

func (s *ExampleAPI) Run(ctx context.Context) error {
	ctx = ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))
	ctxscope.GetLogger(ctx).Info("starting service")
	<-ctx.Done()

	return nil
}

func (s *ExampleAPI) Stop(ctx context.Context) error {
	serviceCtx := ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))
	ctxscope.GetLogger(serviceCtx).Info("stopping service")

	return nil
}
```

After adding files, regenerate discovery and build:

```bash
make service-registration
make build
./build/yourproject run
```

`example-db` starts first (no deps), signals ready, then `example-api` starts. If `example-db.Run()` returns an error, it retries twice with a 2s delay before propagating and stopping the whole process (it does not implement `AllowedFailure`).
