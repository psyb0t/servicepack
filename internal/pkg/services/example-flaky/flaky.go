package exampleflaky

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/psyb0t/ctxerrors"
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
	attempt := readAttempts() + 1
	writeAttempts(attempt)

	slog.Info("starting service",
		"service", ServiceName,
		"attempt", attempt,
		"maxAttempts", maxRetries+1,
	)

	if attempt <= maxRetries {
		slog.Warn("simulating failure",
			"service", ServiceName,
			"attempt", attempt,
		)

		return ctxerrors.Wrapf(
			errFlaky,
			"attempt %d/%d",
			attempt, maxRetries+1,
		)
	}

	slog.Info("finally stable",
		"service", ServiceName,
		"attempt", attempt,
	)

	cleanupAttempts()

	ticker := time.NewTicker(10 * time.Second) //nolint:mnd
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info(
				"context cancelled, stopping service",
				"service", ServiceName,
			)

			return nil
		case <-ticker.C:
			slog.Info("heartbeat",
				"service", ServiceName,
			)
		}
	}
}

func (f *ExampleFlaky) Stop(
	_ context.Context,
) error {
	slog.Info("stopping service", "service", ServiceName)

	cleanupAttempts()

	return nil
}

func readAttempts() int {
	data, err := os.ReadFile(attemptsFile)
	if err != nil {
		return 0
	}

	n, err := strconv.Atoi(string(data))
	if err != nil {
		return 0
	}

	return n
}

func writeAttempts(n int) {
	_ = os.WriteFile(
		attemptsFile,
		[]byte(strconv.Itoa(n)),
		attemptsFileMode,
	)
}

func cleanupAttempts() {
	_ = os.Remove(attemptsFile)
}
