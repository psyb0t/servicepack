```
___ ___ _____   _____ ___ ___ ___  _   ___ _  __
/ __| __| _ \ \ / /_ _/ __| __| _ \/_\ / __| |/ /
\__ \ _||   /\ V / | | (__| _||  _/ _ \ (__| ' <
|___/___|_|_\ \_/ |___\___|___|_|/_/ \_\___|_|\_\
```

# servicepack

A Go service framework that runs your shit concurrently without fucking around.

## Table of Contents

**Getting Started**

- [What is this?](#what-is-this)
- [Quick Start (Make It Your Own in 30 Seconds)](#quick-start-make-it-your-own-in-30-seconds)
- [Just Want to Try It First?](#just-want-to-try-it-first)

**Core Concepts**

- [Creating Services](#creating-services)
- [Service Interface](#service-interface)
  - [Optional Interfaces](#optional-interfaces)
  - [Custom Initialization](#custom-initialization)
- [How Services Actually Work](#how-services-actually-work)
  - [Service Filtering](#service-filtering)

**Essential Tools**

- [The Makefile (Your New Best Friend)](#the-makefile-your-new-best-friend)
  - [Basic Commands](#basic-commands)
  - [Service Management](#service-management)
  - [Development](#development)
  - [Framework Management](#framework-management)
  - [Backup Management](#backup-management)
  - [Script Customization](#script-customization)

**System Details**

- [Architecture](#architecture)
  - [Key Components](#key-components)
- [Environment Variables](#environment-variables)
- [Build System Details](#build-system-details)
  - [Build Process](#build-process)

**Framework Management**

- [Framework Updates](#framework-updates)
  - [Review and Apply Updates](#review-and-apply-updates)
  - [Customizing Updates with .servicepackupdateignore](#customizing-updates-with-servicepackupdateignore)

**Advanced Topics**

- [Pre-commit Hook](#pre-commit-hook)
- [Testing](#testing)
  - [Test Isolation](#test-isolation)
- [Concurrency Model](#concurrency-model)
- [Error Handling](#error-handling)

**Reference**

- [Dependencies](#dependencies)
- [Directory Structure](#directory-structure)
- [Example Services](#example-services)
- [License](#license)

## What is this?

You write services, this thing runs them. All your services go into one binary so you can debug the fuck out of service-to-service calls without dealing with distributed bullshit. Run everything locally, then deploy individual services as microservices when you're ready. Or just fuckin' deploy everything together, y not.

## Quick Start (Make It Your Own in 30 Seconds)

```bash
# Clone this shit
git clone https://github.com/psyb0t/servicepack
cd servicepack

# Make it yours
make own MODNAME=github.com/yourname/yourproject

# Build and run
make build
./build/yourproject run
```

This will:

- Nuke the .git directory
- Replace the module name everywhere
- Set you up with a fresh go.mod
- Replace README with just your project name
- Run `git init` to start fresh
- Setup dependencies
- Create initial commit on main branch

You'll see the hello-world service spamming "Hello, World!" every 5 seconds. Hit Ctrl+C to stop it cleanly.

## Just Want to Try It First?

```bash
git clone https://github.com/psyb0t/servicepack
cd servicepack
make run-dev
```

This builds a dev Docker image and runs it with debug logging. You'll see all the example services in action - retries, dependencies, allowed failures, one-shot jobs, and a crasher that takes everything down after ~30 seconds.

## Creating Services

Create a new service:

```bash
make service NAME=my-cool-service
```

This shits out a skeleton service at `internal/pkg/services/my-cool-service/`. Edit the generated file, put your logic in the `Run()` method. Done - your service starts automatically.

Remove a service:

```bash
make service-remove NAME=my-cool-service
```

## Service Interface

Every service implements this interface:

```go
type Service interface {
    Name() string                        // Return service name
    Run(ctx context.Context) error      // Your service logic goes here
    Stop(ctx context.Context) error     // Cleanup logic (optional)
}
```

The `Run()` method should:

- Listen for `ctx.Done()` and return cleanly when cancelled
- Return an error if something goes wrong (this will stop all services)
- Do whatever the fuck your service is supposed to do

The `Stop()` method is for cleanup - it runs when the app is shutting down.

### Optional Interfaces

Services can opt into advanced behavior by implementing these:

```go
// Retryable - service gets restarted on failure
type Retryable interface {
    MaxRetries() int
    RetryDelay() time.Duration  // e.g. 5*time.Second
}

// AllowedFailure - service can die without killing everything
type AllowedFailure interface {
    IsAllowedFailure() bool
}

// Dependent - service waits for other services to start first
type Dependent interface {
    Dependencies() []string  // service names
}
```

**Retry**: When `Run()` returns an error, the service manager retries up to `MaxRetries()` times with `RetryDelay()` between attempts. If context is cancelled during the delay, it bails cleanly.

**Allowed Failure**: When a service fails (even after retries), its error gets logged but doesn't propagate - other services keep running. Perfect for non-critical shit like cache warmers or metrics exporters.

**Dependencies**: The service manager resolves a dependency graph using topological sort. Services with no deps start first, then their dependents, etc. Cyclic dependencies are detected and rejected. Dependencies on services not in the current process (external databases, services on other servers) are skipped with a debug log.

You can combine them - a service can be retryable AND an allowed failure AND have dependencies.

### Custom Initialization

`cmd/init.go` is your hook to run shit before the app starts. It's never touched by framework updates. Use it to add custom slog handlers, set up global config, or anything else:

```go
// cmd/init.go
package main

import slogconfigurator "github.com/psyb0t/slog-configurator"

func init() {
    slogconfigurator.AddHandler(myLokiHandler)
}
```

Every `slog.Info/Error/etc` call across the entire app - framework, services, everything - goes to all registered handlers. Want Loki? Datadog? Elasticsearch? Just write a `slog.Handler` and plug it in here.

## How Services Actually Work

1. Services are auto-discovered using the [`gofindimpl`](https://github.com/psyb0t/gofindimpl) tool
2. The `scripts/make/service_registration.sh` script finds all Service implementations
3. It generates `internal/pkg/services/services.gen.go` with a `services.Init()` function
4. The `services.Init()` function is called when the app starts to register all services
5. Services get filtered based on the `SERVICES_ENABLED` environment variable

### Service Filtering

By default, all services run. To run specific services:

```bash
export SERVICES_ENABLED="hello-world,my-cool-service"
./build/servicepack run
```

Leave `SERVICES_ENABLED` empty or unset to run all services.

## The Makefile (Your New Best Friend)

### Basic Commands

- `make all` - Full pipeline: dep → lint-fix → test-coverage → build
- `make build` - Build the binary using Docker (static linking)
- `make dep` - Get dependencies with `go mod tidy` and `go mod vendor`
- `make test` - Run all tests with race detection
- `make test-coverage` - Run tests with 90% coverage requirement (excludes example services and cmd packages)
- `make lint` - Lint your code with comprehensive golangci-lint rules (80+ linters enabled)
- `make lint-fix` - Lint and auto-fix issues
- `make clean` - Clean build artifacts and coverage files

### Service Management

- `make service NAME=foo` - Create new service
- `make service-remove NAME=foo` - Remove service
- `make service-registration` - Regenerate service discovery

### Development

- `make run-dev` - Run in development Docker container
- `make docker-build-dev` - Build dev image

### Docker

- `make docker-build` - Build production Docker image
- `make docker-build-dev` - Build development Docker image

### Framework Management

- `make servicepack-update` - Update to latest servicepack framework (creates backup first)
- `make servicepack-update-review` - Review pending framework update changes
- `make servicepack-update-merge` - Merge pending framework update
- `make servicepack-update-revert` - Revert pending framework update
- `make own MODNAME=github.com/you/project` - Make this framework your own

### Backup Management

- `make backup` - Create timestamped backup in `/tmp` and `.backup/`
- `make backup-restore [BACKUP=filename.tar.gz]` - Restore from backup (defaults to latest, nukes everything first)
- `make backup-clear` - Delete all backup files

**Note**: Framework updates (`make servicepack-update`) automatically create backups before making changes.

### Script Customization

You can override any framework script by creating a user version:

```bash
# Create custom script (will override framework version)
cp scripts/make/servicepack/test.sh scripts/make/test.sh
# Edit your custom version
vim scripts/make/test.sh
```

The Makefile checks for user scripts first (`scripts/make/`), then falls back to framework scripts (`scripts/make/servicepack/`). This lets you customize any build step while preserving the ability to update the framework without conflicts.

**Framework scripts** (in `scripts/make/servicepack/`):

- Get updated when you run `make servicepack-update`
- Always preserved - your customizations won't get overwritten

**User scripts** (in `scripts/make/`):

- Take priority over framework scripts
- Never touched by framework updates
- Perfect for project-specific build customizations

### Makefile Customization

The build system uses a split Makefile approach:

```bash
# Override any framework command by defining it in your Makefile
build: ## Custom build command
	@echo "Running my custom build..."
	@docker build -t myapp .

# Add your own custom commands
deploy: ## Deploy to production
	@./deploy.sh
```

**How it works:**

- `Makefile.servicepack` - Contains all framework commands (updated by framework)
- `Makefile` - Your file that includes servicepack + allows custom commands (never touched)
- User commands override framework commands automatically
- `make help` shows both user and framework commands

### Dockerfile Customization

Both development and production Docker environments use the override pattern:

```bash
# Customize development environment
cp Dockerfile.servicepack.dev Dockerfile.dev
vim Dockerfile.dev

# Customize production environment
cp Dockerfile.servicepack Dockerfile
vim Dockerfile
```

**How it works:**

- `Dockerfile.servicepack.dev` - Framework development image (updated by framework)
- `Dockerfile.dev` - Your custom development image (never touched)
- `Dockerfile.servicepack` - Framework production image (updated by framework)
- `Dockerfile` - Your custom production image (never touched)
- `make docker-build-dev` automatically uses your custom development version if it exists

## Architecture

```
cmd/main.go                          # Entry point, CLI setup
internal/app/                        # Application layer
├── app.go                          # Main app orchestration
├── config.go                       # Configuration parsing
internal/pkg/
├── service-manager/                 # Framework service orchestration
│   ├── service_manager.go          # Concurrent service runner
│   ├── errors.go                   # Framework error definitions
│   └── *_test.go                   # Framework tests
└── services/                       # User service space
    ├── services.gen.go             # Auto-generated services.Init() function
    ├── hello-world/                # Example: basic long-running service
    ├── example-database/           # Example: retryable service
    ├── example-api/                # Example: service with dependencies
    ├── example-migrator/           # Example: one-shot with allowed failure
    ├── example-optional/           # Example: allowed failure
    ├── example-flaky/              # Example: fails then recovers
    ├── example-crasher/            # Example: crashes and kills everything
    └── your-service/               # Your services go here
scripts/make/                        # Build script system
├── servicepack/                    # Framework scripts (updated by framework)
│   ├── build.sh                   # Docker build script
│   ├── dep.sh                     # Dependency management
│   ├── test.sh                    # Test runner
│   └── *.sh                       # Other framework scripts
└── [custom scripts]               # User overrides (take priority)
```

### Key Components

**ServiceManager**: Runs your services concurrently with dependency ordering, automatic retries, and allowed failures. Handles shutdown and error propagation. It's a singleton because globals are fine when you know what you're doing.

**Service Registration**: Auto-discovery using [`gofindimpl`](https://github.com/psyb0t/gofindimpl) finds all your Service implementations and generates a `services.Init()` function. No manual registration bullshit.

**App**: Wrapper that runs the ServiceManager and handles the lifecycle shit.

## Environment Variables

The framework uses these:

```bash
# Logging (via slog-configurator)
LOG_LEVEL=debug          # debug, info, warn, error
LOG_FORMAT=json          # json, text
LOG_ADD_SOURCE=true      # show file:line in logs

# Environment (via goenv)
ENV=dev                  # dev, prod (default: prod)

# Service filtering
SERVICES_ENABLED=service1,service2   # comma-separated, empty = all

# Your services can define their own env vars
```

## Build System Details

The build system is dynamic as fuck:

1. App name is extracted from `go.mod` automatically
2. Binary gets built with static linking (no external deps)
3. App name is injected at build time via ldflags
4. Docker builds ensure consistent environment

### Build Process

```makefile
APP_NAME := $(shell head -n 1 go.mod | awk '{print $2}' | awk -F'/' '{print $NF}')

build:
    docker run --rm -v $(PWD):/app -w /app golang:1.25-alpine \
        sh -c "apk add --no-cache gcc musl-dev && \
               CGO_ENABLED=0 go build -a \
               -ldflags '-extldflags \"-static\" -X main.appName=$(APP_NAME)' \
               -o ./build/$(APP_NAME) ./cmd/..."
```

This means your binary name matches your module name automatically.

## Framework Updates

Keep your servicepack framework up to date:

```bash
make servicepack-update
```

This script:

1. Checks for uncommitted changes (fails if found)
2. Compares current version with latest
3. Creates backup if update is needed
4. Creates update branch `servicepack_update_to_VERSION`
5. Downloads latest framework and applies changes
6. Commits changes to update branch for review
7. Leaves you on update branch to review and test

### Review and Apply Updates

After running `make servicepack-update`:

```bash
# Review what changed
make servicepack-update-review

# Test the update
make dep && make service-registration && make test

# If satisfied, merge the update
make servicepack-update-merge

# If not satisfied, discard the update
make servicepack-update-revert
```

### Customizing Updates with .servicepackupdateignore

Create a `.servicepackupdateignore` file to exclude files from framework updates:

```
# Custom framework modifications (these are already user files)
# Note: Dockerfile, Dockerfile.dev, Makefile, and scripts/make/ are automatically excluded

# Local configuration files
*.local
.env*
```

**Framework vs User Files**:

```
cmd/                           # Framework files
internal/app/                  # Framework files
internal/pkg/service-manager/  # Framework files
scripts/make/servicepack/      # Framework scripts (updated by servicepack-update)
scripts/make/                  # User scripts (override framework, never touched)
Makefile.servicepack           # Framework Makefile (updated by servicepack-update)
Makefile                       # User Makefile (includes servicepack, never touched)
Dockerfile.servicepack.dev     # Framework development image (updated by servicepack-update)
Dockerfile.dev                 # User development image (overrides framework, never touched)
Dockerfile.servicepack         # Framework production image (updated by servicepack-update)
Dockerfile                     # User production image (never touched)
.github/                       # Framework files (CI/CD workflows)
LICENSE                        # Your project license
.golangci.yml                  # Framework files
go.mod                         # Your module name preserved
go.sum                         # Gets regenerated
README.md                      # Your project docs
internal/pkg/services/         # Your services - never touched
```

Use `.servicepackupdateignore` to exclude any framework files you've customized.

## Pre-commit Hook

There's a `pre-commit.sh` script that runs `make lint && make test-coverage`. You can:

- Use your favorite pre-commit tool to manage hooks
- Use [`ez-pre-commit`](https://github.com/psyb0t/ez-pre-commit) to auto-setup Git hooks that run this script
- Just use the simple script as-is (it runs lint and coverage checks)

## Testing

Tests are structured per component:

- `internal/app/app_test.go` - Application tests with mock services
- `internal/pkg/service-manager/service_manager_test.go` - Unit tests for retry, allowed failure, dependencies, concurrency
- `internal/pkg/service-manager/service_manager_integration_test.go` - Integration tests combining all features end-to-end
- `internal/pkg/service-manager/errors_test.go` - Error definition and matching tests
- Each service should have its own `*_test.go` files

90% test coverage is required by default (excludes example services and cmd packages). The coverage check runs with race detection and fails if below threshold.

### Test Isolation

- `ResetInstance()` resets the singleton for clean test state
- `ClearServices()` clears all registered services
- Mock services implement the Service interface for testing
- Tests should avoid calling `services.Init()` and manually add mock services instead

## Concurrency Model

- Each service runs in its own goroutine
- ServiceManager uses sync.WaitGroup for coordination
- Context cancellation for clean shutdown
- Services are started in dependency order (topological sort)
- Services in the same dependency group start concurrently
- Retryable services get restarted automatically on failure
- Allowed-failure services can die without killing everything
- Graceful shutdown cancels context and calls Stop() in reverse dependency order
- Per-service stop timeout (30s default) prevents hung shutdowns
- Panics in services are recovered and treated as errors

## Error Handling

- Service errors bubble up through the ServiceManager
- First non-allowed-failure error stops all services
- Allowed failures are logged but don't propagate
- Retryable services exhaust retries before propagating
- Context errors (cancellation) are treated as clean shutdown
- All errors use [`ctxerrors`](https://github.com/psyb0t/ctxerrors) for context preservation
- `ErrNoEnabledServices` - no services registered
- `ErrCyclicDependency` - circular dependency detected
- `ErrServicePanic` - service panicked (recovered, treated as error)
- `ErrStopTimeout` - service didn't stop within timeout

## Dependencies

Core dependencies:

- `log/slog` with [`slog-configurator`](https://github.com/psyb0t/slog-configurator) - Logging
- [`github.com/spf13/cobra`](https://github.com/spf13/cobra) - CLI
- [`github.com/psyb0t/gonfiguration`](https://github.com/psyb0t/gonfiguration) - Config parsing
- [`github.com/psyb0t/ctxerrors`](https://github.com/psyb0t/ctxerrors) - Error handling
- [`github.com/psyb0t/goenv`](https://github.com/psyb0t/goenv) - Environment detection (prod/dev)

Development dependencies:

- [`golangci-lint`](https://github.com/golangci/golangci-lint) - Comprehensive linting (80+ linters: errcheck, govet, staticcheck, gosec, etc.)
- [`testify`](https://github.com/stretchr/testify) - Testing assertions and mocks
- [`gofindimpl`](https://github.com/psyb0t/gofindimpl) - Service auto-discovery tool

## Directory Structure

```
.
├── cmd/
│   ├── main.go                    # Entry point
│   └── init.go                    # Your custom init hooks
├── pkg/runner/                     # App lifecycle runner
├── internal/
│   ├── app/                        # Application layer
│   └── pkg/services/               # Services
├── scripts/make/                   # Build script system
│   ├── servicepack/               # Framework scripts (auto-updated)
│   └── [user scripts]             # User overrides (take priority)
├── build/                          # Build output
├── vendor/                         # Vendored dependencies
├── Makefile                        # User Makefile (includes servicepack framework)
├── Makefile.servicepack            # Framework Makefile (auto-updated)
├── Dockerfile                      # User production image (optional override)
├── Dockerfile.dev                  # User development image (optional override)
├── Dockerfile.servicepack          # Framework production image (auto-updated)
├── Dockerfile.servicepack.dev      # Framework development image (auto-updated)
└── servicepack.version             # Framework version tracking
```

## Example Services

The framework ships with example services that demonstrate every lifecycle pattern:

| Service | Pattern |
|---|---|
| `hello-world` | Long-running, no deps, basic service |
| `example-database` | Long-running, retryable (2 retries, 2s delay) |
| `example-api` | Long-running, depends on database + flaky |
| `example-migrator` | One-shot, depends on database, allowed failure |
| `example-optional` | Allowed failure, fails immediately but app keeps running |
| `example-flaky` | Retryable (2 retries, 1s delay), fails twice then recovers |
| `example-crasher` | Retryable (2 retries, 3s delay), fails all retries and kills everything |

Run them all: `go run ./cmd run` or `make run-dev`

These get removed when you run `make own`.

## License

MIT
