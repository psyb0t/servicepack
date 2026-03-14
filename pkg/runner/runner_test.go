package runner

import (
	"context"
	"errors"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTest = errors.New("test error")

type mockRunnable struct {
	runFunc  func(ctx context.Context) error
	stopFunc func(ctx context.Context) error
}

func (m *mockRunnable) Run(ctx context.Context) error {
	if m.runFunc != nil {
		return m.runFunc(ctx)
	}

	<-ctx.Done()

	return nil
}

func (m *mockRunnable) Stop(ctx context.Context) error {
	if m.stopFunc != nil {
		return m.stopFunc(ctx)
	}

	return nil
}

func TestRun_AppError(t *testing.T) {
	r := &mockRunnable{
		runFunc: func(_ context.Context) error {
			return errTest
		},
	}

	err := Run(r)
	assert.Error(t, err)
}

func TestRun_CleanExit(t *testing.T) {
	r := &mockRunnable{
		runFunc: func(_ context.Context) error {
			return nil
		},
	}

	err := Run(r)
	assert.NoError(t, err)
}

func TestGetConfig(t *testing.T) {
	cfg, err := getConfig()
	require.NoError(t, err)
	assert.Equal(
		t, defaultShutdownTimeout, cfg.ShutdownTimeout,
	)
}

func TestGetConfig_Custom(t *testing.T) {
	t.Setenv(envVarNameShutdownTimeout, "5s")

	cfg, err := getConfig()
	require.NoError(t, err)

	expected := 5 * time.Second
	assert.Equal(t, expected, cfg.ShutdownTimeout)
}

func TestHandleShutdownTimeout_DeadlineExceeded(
	t *testing.T,
) {
	r := &appRunner{shutdownTimeout: time.Nanosecond}

	ctx, cancel := context.WithTimeout(
		context.Background(), time.Nanosecond,
	)
	defer cancel()

	time.Sleep(time.Millisecond)

	err := r.handleShutdownTimeout(ctx, nil)
	assert.ErrorIs(t, err, ErrShutdownTimeout)
}

func TestHandleShutdownTimeout_WithShutdownErr(
	t *testing.T,
) {
	r := &appRunner{shutdownTimeout: time.Nanosecond}

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	err := r.handleShutdownTimeout(ctx, errTest)
	assert.Error(t, err)
}

func TestHandleShutdownTimeout_NilCtxErr(
	t *testing.T,
) {
	r := &appRunner{shutdownTimeout: time.Second}

	err := r.handleShutdownTimeout(
		context.Background(), errTest,
	)
	assert.Equal(t, errTest, err)
}

func TestWaitForShutdown_Signal(t *testing.T) {
	r := &appRunner{}

	sigCh := make(chan os.Signal, 1)
	errCh := make(chan error, 1)

	sigCh <- syscall.SIGTERM

	err := r.waitForShutdown(sigCh, errCh)
	assert.NoError(t, err)
}

func TestWaitForShutdown_Error(t *testing.T) {
	r := &appRunner{}

	sigCh := make(chan os.Signal, 1)
	errCh := make(chan error, 1)

	errCh <- errTest

	err := r.waitForShutdown(sigCh, errCh)
	assert.ErrorIs(t, err, errTest)
}

func TestWaitForShutdown_NilError(t *testing.T) {
	r := &appRunner{}

	sigCh := make(chan os.Signal, 1)
	errCh := make(chan error, 1)

	errCh <- nil

	err := r.waitForShutdown(sigCh, errCh)
	assert.NoError(t, err)
}
