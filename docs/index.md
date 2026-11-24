---
title: "Home"
description: "servicepack is a clone-and-own Go service framework: run related services in one process for debugging, then deploy one binary or split them."
sidebar_position: 1
slug: /
---

# servicepack

A clone-and-own Go service framework. Run related services together so you can debug the whole thing in one process, then deploy it as one binary or run each service as its own deployment when that makes sense.

It is a template repository, not a `go get` library. A new project gets a concurrent service supervisor, a lifecycle runner, a Docker-first build and test workflow, generated service registration, and clear places to customize it.

```bash
git clone https://github.com/psyb0t/servicepack.git my-service
cd my-service
make own MODNAME=github.com/yourname/my-service
make service NAME=worker
make build
./build/my-service run
```

| Read this | For |
| --- | --- |
| [Getting started](getting-started.md) | Turning a fresh clone into your project, then adding and running a service. |
| [Services and lifecycle](services-and-lifecycle.md) | The service contract, dependencies, readiness, retries, commands, filtering and shutdown. |
| [Architecture](architecture.md) | The process layout, generated registration, logging, and running services in one deployment or several. |
| [Development](development.md) | The Docker-first Make targets, tests and coverage, builds, dependencies and overrides. |
| [Framework updates](framework-updates.md) | What `make servicepack-update` changes, what it leaves alone, and how to keep your changes out of its way. |
| [Reference](reference.md) | Environment variables, interface signatures, service package rules and every Make target. |

The code-level deep dives live next to the code: the [service manager](https://github.com/psyb0t/servicepack/blob/main/internal/pkg/service-manager/README.md), the [runner](https://github.com/psyb0t/servicepack/blob/main/pkg/runner/README.md) and the [framework Make scripts](https://github.com/psyb0t/servicepack/blob/main/scripts/make/servicepack/README.md).
