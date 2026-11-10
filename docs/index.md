---
title: "Home"
description: "servicepack is a clone-and-own Go service framework: run related services in one process for debugging, then deploy one binary or split them."
nav_order: 1
permalink: /
seo:
  type: WebSite
---

# servicepack

A clone-and-own Go service framework. Run related services together locally so you can debug the whole thing in one process, then deploy it as one binary or split those services into separate microservices when that makes sense.

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
| [Services and lifecycle](services-and-lifecycle.md) | Interface contracts, dependencies, retries, commands, filtering, logging and shutdown. |
| [Development](development.md) | Docker-first Make targets, Testcontainers, coverage, builds and overrides. |
| [Architecture](architecture.md) | The process topology, ownership boundaries, generated registration and deployment choices. |
| [Framework updates](framework-updates.md) | Updating a clone safely and keeping project customizations out of the blast radius. |

The code-level deep dives live next to the code: the [service manager](https://github.com/psyb0t/servicepack/blob/main/internal/pkg/service-manager/README.md), the [runner](https://github.com/psyb0t/servicepack/blob/main/pkg/runner/README.md) and the [framework Make scripts](https://github.com/psyb0t/servicepack/blob/main/scripts/make/servicepack/README.md).
