package exampleflaky

import (
	"context"
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxscope"
)

const ServiceName = "example-flaky"

var errFlaky = errors.New("flaky failure")

const (
	attemptsFile     = "/tmp/example-flaky-attempts"
	attemptsFileMode = 0o644
	maxRetries       = 2
)

// ExampleFlaky demonstrates retry behavior with state.
// It tracks how many times it has been called in a temp
// file. It fails on the first attempts and succeeds on
// the last retry. On success it cleans up the temp file
// so the next run starts fresh.
type ExampleFlaky struct{}

func New() (*ExampleFlaky, error) {
	return &ExampleFlaky{}, nil
}

func (f *ExampleFlaky) Name() string {
	return ServiceName
}

// MaxRetries makes ExampleFlaky implement the Retryable
// interface.
func (f *ExampleFlaky) MaxRetries() int {
	return maxRetries
}

func (f *ExampleFlaky) RetryDelay() time.Duration {
	return time.Second
}

func (f *ExampleFlaky) Run(
	ctx context.Context,
) error {
	ctx = ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))
	logger := ctxscope.GetLogger(ctx)

	attempt := readAttempts(ctx) + 1
	if err := writeAttempts(attempt); err != nil {
		return ctxerrors.Wrap(err, "write attempt count")
	}

	logger.Info("starting service",
		"attempt", attempt,
		"max_attempts", maxRetries+1,
	)

	if attempt <= maxRetries {
		logger.Warn("simulating failure",
			"attempt", attempt,
		)

		return ctxerrors.Wrapf(
			errFlaky,
			"attempt %d/%d",
			attempt, maxRetries+1,
		)
	}

	logger.Info("finally stable",
		"attempt", attempt,
	)

	if err := cleanupAttempts(); err != nil {
		return ctxerrors.Wrap(err, "clean up attempt count")
	}

	ticker := time.NewTicker(10 * time.Second) //nolint:mnd
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("context cancelled, stopping service")

			return nil
		case <-ticker.C:
			logger.Info("heartbeat")
		}
	}
}

func (f *ExampleFlaky) Stop(
	ctx context.Context,
) error {
	serviceCtx := ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))

	ctxscope.GetLogger(serviceCtx).Info("stopping service")

	if err := cleanupAttempts(); err != nil {
		return ctxerrors.Wrap(err, "clean up attempt count")
	}

	return nil
}

func readAttempts(ctx context.Context) int {
	data, err := os.ReadFile(attemptsFile)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			ctxscope.GetLogger(ctx).Warn(
				"read attempt count failed",
				"err", err,
			)
		}

		return 0
	}

	n, err := strconv.Atoi(string(data))
	if err != nil {
		ctxscope.GetLogger(ctx).Warn("parse attempt count failed", "err", err)

		return 0
	}

	return n
}

func writeAttempts(n int) error {
	// This example service persists a restart-surviving attempt counter at a
	// FIXED path on purpose so readAttempts and cleanupAttempts can find it
	// again; a random os.CreateTemp name would defeat that. Demo code, not a
	// real attack surface.
	// nosemgrep: go.lang.security.bad_tmp.bad-tmp-file-creation
	if err := os.WriteFile(
		attemptsFile,
		[]byte(strconv.Itoa(n)),
		attemptsFileMode,
	); err != nil {
		return ctxerrors.Wrap(err, "write attempts file")
	}

	return nil
}

func cleanupAttempts() error {
	err := os.Remove(attemptsFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return ctxerrors.Wrap(err, "remove attempts file")
	}

	return nil
}
