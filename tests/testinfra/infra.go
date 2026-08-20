// Package testinfra is the extendable integration-test harness for a
// servicepack app. Its baseline builds the servicepack image from the repo
// Dockerfile and runs it, so a test exercises the real, containerized
// application with whatever services it registers, and nothing more.
//
// Extend it with the external dependencies your services need (a database, a
// cache, a broker via testcontainers-go): give Infra a field, start it in
// Setup, wire the app container to it, and tear it down in Teardown. The DIND
// runner already supports this: DEV_RUN_DIND uses the host network, so
// testcontainers' host-published ports are reachable.
package testinfra

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/psyb0t/ctxerrors"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// appBootLog is what the framework logs the instant app.Run starts, in every
// servicepack app. Waiting on it proves the built image boots and runs its
// services, whatever they are.
const appBootLog = "running app"

const (
	appDockerfile  = "Dockerfile"
	appRunCommand  = "run"
	appBootTimeout = 5 * time.Minute // covers a cold image build plus boot
)

var errNoGoMod = errors.New("go.mod not found above the working directory")

// Infra holds the containers a test package brought up. At the baseline that is
// just the application image; extend it with one field per external dependency
// (for example: Postgres *PostgresResource) as your services grow.
type Infra struct {
	App testcontainers.Container
}

// Setup builds the app image from the repo Dockerfile and starts it with the
// run command, blocking until the app logs that it is running. On failure it
// returns the error with nothing left running to leak.
func Setup(ctx context.Context) (*Infra, error) {
	root, err := repoRoot()
	if err != nil {
		return nil, err
	}

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    root,
			Dockerfile: appDockerfile,
			KeepImage:  false,
		},
		Cmd: []string{appRunCommand},
		WaitingFor: wait.ForLog(appBootLog).
			WithStartupTimeout(appBootTimeout),
	}

	container, err := testcontainers.GenericContainer(ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		},
	)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "build and start servicepack image")
	}

	return &Infra{App: container}, nil
}

// Teardown terminates every started container. Idempotent and best-effort: safe
// to call from a Setup failure path and from a test's cleanup.
func (i *Infra) Teardown(ctx context.Context) {
	if i.App != nil {
		_ = i.App.Terminate(ctx)
	}
}

// repoRoot walks up from the test's working directory to the module root (the
// directory holding go.mod), which is where the Dockerfile and its build
// context live.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", ctxerrors.Wrap(err, "get working directory")
	}

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errNoGoMod
		}

		dir = parent
	}
}
