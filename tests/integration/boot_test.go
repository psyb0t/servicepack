//go:build integration

// Package integration holds servicepack's integration tests. The baseline is the
// very bottom of the pyramid: build the servicepack image and run it, confirming
// the application boots with whatever services are registered. It asserts nothing
// about any specific service, so a downstream keeps this test verbatim and it
// keeps passing as it swaps in its own services and Dockerfile.
package integration

import (
	"context"
	"testing"

	"github.com/psyb0t/servicepack/tests/testinfra"
	"github.com/stretchr/testify/require"
)

// TestApp_BootsAndRuns builds servicepack's Docker image and runs it. testinfra
// blocks until the app logs that it is running, so reaching the assertions here
// means the image built and the application came up. The container-state check
// confirms it is still running right after boot.
func TestApp_BootsAndRuns(t *testing.T) {
	ctx := context.Background()

	infra, err := testinfra.Setup(ctx)
	require.NoError(t, err, "build and boot the servicepack image")

	t.Cleanup(func() {
		infra.Teardown(context.Background())
	})

	state, err := infra.App.State(ctx)
	require.NoError(t, err, "inspect app container state")
	require.True(t, state.Running, "app container must be running after boot")
}
