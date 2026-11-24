---
title: "Development"
description: "The Docker-first servicepack Make targets: builds, lint, Testcontainers tests, the coverage gate, security checks, dependencies and overrides."
sidebar_position: 5
---

# Development

Go tooling runs inside Docker through Make targets. A clean machine needs
Docker, not a matching Go toolchain, and every developer and CI run uses the
same Go version.

Targets run in one of these ways:

- most targets, such as lint, format, dependencies, security checks and
  `make test-unit`, run in the project's development image with the repository
  mounted at `/work`;
- `make test`, `make test-integration` and `make test-coverage` run in the same
  image with the Docker socket mounted and host networking, so Testcontainers
  can start containers and reach their ports;
- `make build` runs a pinned Go build image with the repository mounted at
  `/app`, and `make docker-build` runs `docker build`;
- `make run-dev` builds the development image and runs the source baked into
  it, without mounting the repository, so rerun it after each change;
- `make own`, `make servicepack-update*` and `make backup*` run on the host.

Do not replace these with a host `go test` or host linter in project automation:
two toolchains give two chances to disagree.

## Targets

The [reference](reference.md#make-targets) lists every target and what it
does. `make help` prints the ones your copy has.

## CPU limit

Containers started through the development image's `DEV_RUN` and `DEV_RUN_DIND`
commands, which is most targets, run with `--cpus` set to the host thread count
minus two, never below one, so lint and tests leave the machine responsive.
Go sizes `GOMAXPROCS` from that limit. `make build`, `make run-dev` and the
containers tests start themselves are not limited. These Make variables change
it:

| Variable | Default | Effect |
| --- | --- | --- |
| `DEV_CPUS_RESERVED` | `2` | Threads kept free for the host. |
| `DEV_CPUS` | host threads minus `DEV_CPUS_RESERVED`, minimum 1 | Threads the container may use, for example `make lint DEV_CPUS=4`. |
| `DEV_CPU_LIMIT_ARGS` | `--cpus=$(DEV_CPUS)` | The Docker flags added to `DEV_RUN` and `DEV_RUN_DIND`. Set it empty (`DEV_CPU_LIMIT_ARGS=`) to remove the limit. |
| `DEV_RUN_EXTRA_ARGS` | empty | Extra `docker run` arguments for `DEV_RUN`. |
| `DEV_RUN_DIND_EXTRA_ARGS` | empty | Extra `docker run` arguments for `DEV_RUN_DIND`. |

## Tests and coverage

`make test` is the normal full suite: the race detector on, and Docker
available for Testcontainers. `make test-unit` runs the same packages without
the Docker socket, so tests that start containers cannot run there.
`make test-integration` adds the tests behind `//go:build integration`, such as
the shipped boot test in `tests/integration`, and runs without the test cache
and with a ten-minute timeout. `make test` and `make test-unit` leave those
tests out.

`make test-coverage` also uses the race detector and runs with
`-tags=integration`. It instruments every package in the module
(`-coverpkg=<module>/...`), so a test under `tests/` counts toward the package
it drives, and your services count toward the total. The floor leaves out only
code that is not hand-written code under test: `cmd/` mains, the `tests/`
harness, generated `*.gen.go` files, the service-manager mocks, and the
framework's `example-*` and `hello-world` demo services. Raise the floor
deliberately:

```bash
make test-coverage MIN_TEST_COVERAGE=95
```

It writes `coverage-percent.txt` for the badge workflow and removes the
temporary profiles. If Docker cannot start containers, fix Docker access rather
than skipping the integration tests.

The script can also count a binary that runs inside a container during a test.
It exports `SERVICEPACK_COVDATA_DIR` and merges any Go coverage data it finds
there into the total. Nothing produces that data by default: you build that
image with `go build -cover` and mount the directory into the container as
`GOCOVERDIR` in your own test.

## Test your services

Most service tests call the service directly: build it with `New`, run `Run`
with a context you cancel, and call `Stop`.

To test how services behave together, such as ordering, readiness or retries,
run them through the manager without the generated registration. Reset the
singleton and add only the services the test needs:

```go
servicemanager.ResetInstance()
manager := servicemanager.GetInstance()
manager.Add(apiService, databaseService)

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

errCh := make(chan error, 1)
go func() { errCh <- manager.Run(ctx) }()

// drive the services through channels you control, then:
cancel()
err := <-errCh
```

The manager is a process-wide singleton, so these tests cannot use
`t.Parallel()`.

For real infrastructure, use `tests/`. `tests/testinfra` is the harness: its
`Setup` builds the image from the repository's `Dockerfile` and starts it, and
`tests/integration` boots that image to check the application comes up with its
registered services. Grow both: add the databases, caches or brokers your
services need to `Infra` with testcontainers-go, and add tests that drive them.
Updates never overwrite `tests/`.

## Build output and runtime identity

`make build` installs the static-build requirements in a temporary build
container and writes `build/<app>` with your user and group IDs. It sets three
values with linker flags:

- `main.appName`: the last segment of the module path, used as the binary and
  root command name;
- `main.buildCommit`: the `HEAD` commit, when there is one;
- `main.buildVersion`: the exact Git tag at `HEAD`, or `dev` when `HEAD` is not
  tagged.

`cmd/main.go` puts them in the global log scope as `binary`, `commit` and
`version`, so every log line says which build wrote it. A tree without a Git
`HEAD` still builds, without a commit.

`make docker-build` builds the production image from `Dockerfile`, or from
`Dockerfile.servicepack` when the project has no `Dockerfile`. It is a separate
path from `make build`; check the relevant script before changing either.

## Dependencies and code generation

Use the Make targets, so `go.mod`, `go.sum` and `vendor/` stay in step:

```bash
make dep
make pkg-add PKG=example.com/module@v1.2.3
make pkg-update PKG=example.com/module
make pkg-upgrade
make pkg-remove PKG=example.com/module
```

`vendor/` is committed. Do not edit vendored code by hand. `pkg-add`,
`pkg-update`, `pkg-upgrade` and `pkg-add-tool` refuse module versions published
less than seven days ago (see the [module age gate](reference.md#module-age-gate)).

`make service` regenerates service registration; run `make service-registration`
after changing services by hand. `internal/pkg/services/services.gen.go` is
generated output.

## Override a target

`Makefile` includes `Makefile.servicepack`. To replace a framework target,
define a target with the same name in your `Makefile`, and add your own targets
next to it.

Targets that run a script look for `scripts/make/<script>.sh` first and fall
back to `scripts/make/servicepack/<script>.sh`. Every script finds `common.sh`
and the scripts it calls through `SCRIPT_DIR`, its own directory. So when you
copy one, point `SCRIPT_DIR` back at `servicepack/`, then edit the copy:

```bash
cp scripts/make/servicepack/test.sh scripts/make/test.sh
sed -i 's|dirname "${BASH_SOURCE\[0\]}")"|dirname "${BASH_SOURCE[0]}")/servicepack"|' scripts/make/test.sh
```

A copied script then calls the framework's versions of its sibling scripts,
not your overrides. For example, an overridden `service.sh` still runs the
framework's `service_registration.sh`. If you override both, change the call
in your copy.

The [reference](reference.md#make-targets) shows which targets run a script.
The others are defined directly in `Makefile.servicepack`, so a script in
`scripts/make/` does not change them; override them in your `Makefile`.

`Dockerfile`, `Dockerfile.dev`, `cmd/init.go` and `cmd/commands.go` are yours
too. [Framework updates](framework-updates.md) explains what an update replaces.

## Before you hand off a change

Run the narrowest useful target first, then the full checks: `make test`,
`make lint`, and `make test-coverage` when framework behavior changes.
`pre-commit.sh` runs `make lint && make test-coverage`. Git does not run it
until you install it as a hook:

```bash
ln -s ../../pre-commit.sh .git/hooks/pre-commit
```

Read [architecture](architecture.md) for where a change belongs.
