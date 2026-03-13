package examplecrasher

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

const ServiceName = "example-crasher"

var errCrash = errors.New("shit hit the fan")

// ExampleCrasher demonstrates retry and failure behavior.
// It crashes after 5 seconds, retries up to 2 times,
// and is NOT an allowed failure - so it takes down the
// whole service manager when retries are exhausted.
type ExampleCrasher struct{}

func New() (*ExampleCrasher, error) {
	return &ExampleCrasher{}, nil
}

func (c *ExampleCrasher) Name() string {
	return ServiceName
}

// MaxRetries makes ExampleCrasher implement the Retryable
// interface. The service manager will restart it up to 2
// times before giving up.
func (c *ExampleCrasher) MaxRetries() int {
	return 2 //nolint:mnd
}

func (c *ExampleCrasher) RetryDelay() time.Duration {
	return 3 * time.Second //nolint:mnd
}

func (c *ExampleCrasher) Run(
	ctx context.Context,
) error {
	slog.Info("starting service", "service", ServiceName)

	crashDelay := 10 * time.Second //nolint:mnd
	timer := time.NewTimer(crashDelay)

	defer timer.Stop()

	select {
	case <-ctx.Done():
		slog.Info(
			"context cancelled, stopping service",
			"service", ServiceName,
		)

		return nil
	case <-timer.C:
		slog.Error("crashing on purpose",
			"service", ServiceName,
		)

		return errCrash
	}
}

func (c *ExampleCrasher) Stop(
	_ context.Context,
) error {
	slog.Info("stopping service", "service", ServiceName)

	return nil
}
