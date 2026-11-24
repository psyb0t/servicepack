---
title: "Framework updates"
description: "How make servicepack-update brings framework fixes into your clone, what it never touches, and how to keep your customizations out of its way."
sidebar_position: 6
---

# Framework updates

A project made from servicepack owns its services and extension points and
keeps a copy of the framework. `make servicepack-update` brings that copy
forward without touching your code.

## Before updating

The update runs on the host, not in the development image, so it needs `git`,
`tar`, `rsync`, `jq`, `docker` and `go` on the `PATH`, and network access to
GitHub. It edits `go.mod` with the host's `go mod edit`, so host Go must be
1.24 or newer and able to handle the `go` line in your `go.mod`, either by
being that version or newer, or by downloading it with the default
`GOTOOLCHAIN=auto`.

Start from a clean working tree. The update refuses to run with uncommitted
or untracked files, because it makes a branch and needs to tell its changes
apart from yours.

```bash
make servicepack-update
```

It finds the latest servicepack tag (or the head of `main` when there are no
tags), downloads that copy, and runs the downloaded copy's update logic, so a
release can fix the update process in the same run that installs it.

The update then:

1. makes a backup;
2. creates the branch `servicepack_update_to_<version>` from your current
   branch;
3. copies the framework files onto it;
4. keeps your module path and merges the framework's minimum dependency
   versions into your `go.mod` instead of replacing it; and
5. commits the result on that branch and leaves it checked out.

Review it, then merge it or throw it away:

```bash
make servicepack-update-review
make servicepack-update-merge
make servicepack-update-revert
```

Treat the branch like any other dependency upgrade: read the diff and run your
checks before you merge it. The
[framework Make-script README](https://github.com/psyb0t/servicepack/blob/main/scripts/make/servicepack/README.md)
documents the scripts behind these targets.

## What an update changes

The update copies servicepack's whole tree over yours, except for a fixed list
of paths and the patterns in your `.servicepackupdateignore`. Anything not
excluded is overwritten, root files included: `Makefile.servicepack`,
`.golangci.yml`, `.gitattributes`, `pre-commit.sh`, `scripts/make/servicepack/`,
`scripts/check-go-age.sh`, `cmd/main.go`, `internal/app/`,
`internal/pkg/service-manager/`, `pkg/runner/` and the framework's GitHub
workflows. Your script overrides in `scripts/make/` are not framework files, so
they are never touched. Files you add in your own directories are never
touched.

Nothing is deleted. A file servicepack removes in a later release stays in your
project until you remove it.

The update never copies these paths, all relative to the project root:

- your services, `internal/pkg/services/*`;
- `docs/` and `tests/`;
- `README.md`, `LICENSE`, `CHANGELOG.md`, `.gitignore`, `Makefile`,
  `Dockerfile` and `Dockerfile.dev`;
- your agent instructions: `AGENTS.md`, `CLAUDE.md`, `.claude` and the whole
  `.agents` tree;
- `.servicepackupdateignore` itself;
- `go.mod`, `go.sum` and `vendor/`, which get the dependency merge above
  instead;
- `build/`, `coverage.txt` and Git internals.

servicepack's documentation site (`website/` and `.github/workflows/docs.yml`)
and its `.github/workflows/collaborators-only.yml` are also excluded. `make own`
deletes them, and an update never brings them back.

Only those root paths are skipped. A framework file with the same name deeper
in the tree, such as the READMEs in `internal/pkg/service-manager/`,
`pkg/runner/` and `scripts/make/servicepack/`, is updated like any other
framework file.

## Your ignore file

The `.servicepackupdateignore` that `make own` leaves you already excludes
`cmd/init.go`, `cmd/commands.go`, your Dependabot config, `.dockerignore`,
`.gitleaks.toml`, `.github/FUNDING.yml`, and servicepack's pipeline,
mirror-and-archive and issue-pull workflows. The mirror-and-archive workflow
would push a private project to public mirrors.

Each line is an rsync exclude pattern. A pattern without a leading `/` matches
at any depth: `config.yml` excludes every file of that name in the tree, and
`/config.yml` only the one at the root.

The file is yours after `make own`, and updates never change it. When a later
servicepack release adds a default entry, compare your file with the new one
and copy what you want.

Exclude a framework file only when you must keep your own version of it. From
then on, merging the framework's fixes into that file is your job. Use the
extension points first:

- lifecycle hooks in `cmd/init.go`;
- application commands in `cmd/commands.go`;
- targets in `Makefile` and script overrides in `scripts/make/`;
- `Dockerfile` and `Dockerfile.dev`.

Do not exclude broadly to make an update look clean. Skipping the framework's
changes is not updating.

## The Dockerfiles

`Dockerfile` and `Dockerfile.dev` are yours, so updates never change them,
including the Go version and base image digests they pin. The framework's
current images are `Dockerfile.servicepack` and `Dockerfile.servicepack.dev`,
which updates do bring forward.

- Delete `Dockerfile.dev` unless you changed it. The development targets then
  build `Dockerfile.servicepack.dev`.
- Keep `Dockerfile`: the test harness in `tests/testinfra` builds it by name.
  After each update, compare it with `Dockerfile.servicepack` and copy the
  changes you want, which can go beyond the base image.

## Version tracking and backups

`make own` writes `servicepack.version` when the project has none, recording
the servicepack tag you cloned, or its commit when the clone was not on a tag.
Each update compares it with the latest release to decide whether to run, and
writes the release it installed.

The update makes a backup first. You can also manage backups yourself:

```bash
make backup
make backup-restore
make backup-clear
```

`make backup-restore` restores the latest backup unless you pass
`BACKUP=<file>`. It first deletes everything in the project directory except
`.backup/`, and backups hold only files Git tracks or would track, so a restore
permanently removes ignored files such as `.env`. Copy those somewhere safe
first. `make backup-clear` deletes every backup.

## After updating

Review the update branch, settle any place where your code and the framework's
changes meet, then run the checks from [development](development.md):

```bash
make test
make lint
make test-coverage
```

If the update changes lifecycle code, also start and stop the application for
real. [Architecture](architecture.md) describes those parts.
