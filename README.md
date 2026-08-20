```
___ ___ _____   _____ ___ ___ ___  _   ___ _  __
/ __| __| _ \ \ / /_ _/ __| __| _ \/_\ / __| |/ /
\__ \ _||   /\ V / | | (__| _||  _/ _ \ (__| ' <
|___/___|_|_\ \_/ |___\___|___|_|/_/ \_\___|_|\_\
```

# servicepack

[![Go Reference](https://pkg.go.dev/badge/github.com/psyb0t/servicepack.svg)](https://pkg.go.dev/github.com/psyb0t/servicepack)
[![CI](https://github.com/psyb0t/servicepack/actions/workflows/pipeline.yml/badge.svg?branch=main)](https://github.com/psyb0t/servicepack/actions/workflows/pipeline.yml)
[![coverage](https://raw.githubusercontent.com/psyb0t/servicepack/badges/coverage.svg)](https://github.com/psyb0t/servicepack/actions/workflows/pipeline.yml)
[![version](https://raw.githubusercontent.com/psyb0t/servicepack/badges/version.svg)](https://github.com/psyb0t/servicepack/tags)
[![license](https://raw.githubusercontent.com/psyb0t/servicepack/badges/license.svg)](LICENSE)

Clone-and-own Go service framework: run related services together locally so
you can debug the whole fucking thing in one process, then deploy it as one
binary or split those services into separate microservices when that makes
sense.

This is a template repo, not a `go get` library. It gives a new Go project a
concurrent service supervisor, lifecycle runner, Docker-first build/test
workflow, generated service registration, and sane places to customize it.

## Quick start

Use a fresh clone. `make own` deliberately removes the clone's Git history,
rewrites its module path, removes shipped example services (except
`hello-world`), initializes a new repository, and creates its first commit.

```bash
git clone https://github.com/psyb0t/servicepack.git my-service
cd my-service
make own MODNAME=github.com/yourname/my-service
make service NAME=worker
make build
./build/my-service run
```

The normal Make targets use Docker, so you do not need a matching host Go
toolchain for `make build`, `make test`, or `make lint`. Docker must be
available. See [getting started](docs/getting-started.md) before pointing it
at a clone you care about.

Want to poke the shipped examples first?

```bash
git clone https://github.com/psyb0t/servicepack.git
cd servicepack
make run-dev
```

The examples show retries, dependency ordering, readiness, allowed failures,
and a deliberate crash that stops the process after roughly 36 seconds.

## Why this exists

Local composition is the useful bit: workers, API servers, migrations, and
other process-level services can run together with one signal path and one
structured log stream. That is not a lifetime commitment to a monolith.

- **One binary:** ship the related services together when they genuinely share
  a release cadence and operational boundary.
- **Separate deployables:** move a service behind HTTP, gRPC, a queue, or
  another process boundary when it needs independent scaling, ownership, or
  failure isolation. Servicepack's in-process `Dependent` contract only
  coordinates services present in that binary; external dependencies belong in
  the service's own connection and retry logic.

## Everyday workflow

| Command | What it does |
| --- | --- |
| `make service NAME=worker` | Scaffold a service and regenerate registration. |
| `make test` | Run race-enabled tests in the development container. |
| `make test-integration` | Run uncached, race-enabled tests with Docker available to Testcontainers. |
| `make test-coverage` | Enforce the default 90% framework coverage floor. |
| `make lint` | Run shell and Go linting in Docker. |
| `make build` | Produce a static binary in `build/`, with binary name and source commit in log scope. |
| `make docker-build` | Build the production image. |
| `make help` | List every available target. |

`make servicepack-update` updates the framework copy in a project made from
this template. Read [framework updates](docs/framework-updates.md) first; it
expects a clean repository and creates a backup/update branch.

## Documentation

| Read this | For |
| --- | --- |
| [Getting started](docs/getting-started.md) | Turning a fresh clone into your project, then adding and running a service. |
| [Services and lifecycle](docs/services-and-lifecycle.md) | Interface contracts, dependencies, retries, commands, filtering, logging, and shutdown. |
| [Development](docs/development.md) | Docker-first Make targets, Testcontainers, coverage, builds, and overrides. |
| [Architecture](docs/architecture.md) | The process topology, ownership boundaries, generated registration, and deployment choices. |
| [Framework updates](docs/framework-updates.md) | Updating a clone safely and keeping project customizations out of the blast radius. |
| [Service manager deep dive](internal/pkg/service-manager/README.md) | Exact orchestration semantics and test conventions. |
| [Runner deep dive](pkg/runner/README.md) | Signals, parent contexts, and shutdown deadline behavior. |
| [Framework Make scripts](scripts/make/servicepack/README.md) | The updateable script layer and user overrides. |

## Project layout

```
cmd/                            entry point plus your init/CLI extension points
internal/app/                   application lifecycle wrapper
internal/pkg/service-manager/   concurrent service orchestration
internal/pkg/services/          your services and generated registration
pkg/runner/                     signal-aware lifecycle runner
scripts/make/servicepack/       updateable framework Make scripts
scripts/make/                   project-specific script overrides
docs/                           operational and architectural documentation
```

## Agent integrations

Agent-facing setup and conventions live under
[`.agents/skills/servicepack/`](.agents/skills/servicepack/). The skill is
available through the psyb0t agents marketplace for Claude Code, Codex, and
OpenClaw-compatible setups.

## License

MIT. See [LICENSE](LICENSE). Release history lives in
[CHANGELOG.md](CHANGELOG.md).
