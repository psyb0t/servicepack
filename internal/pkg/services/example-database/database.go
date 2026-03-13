package exampledatabase

import (
	"context"
	"log/slog"
	"time"
)

const ServiceName = "example-database"

// ExampleDatabase demonstrates retry behavior.
// It simulates a database connection pool that retries
// on failure.
type ExampleDatabase struct{}

func New() (*ExampleDatabase, error) {
	return &ExampleDatabase{}, nil
}

func (d *ExampleDatabase) Name() string {
	return ServiceName
}

// MaxRetries returns the number of retry attempts.
// This makes ExampleDatabase implement the Retryable
// interface.
func (d *ExampleDatabase) MaxRetries() int {
	return 2 //nolint:mnd
}

func (d *ExampleDatabase) RetryDelay() time.Duration {
	return 2 * time.Second //nolint:mnd
}

func (d *ExampleDatabase) Run(
	ctx context.Context,
) error {
	slog.Info("starting service", "service", ServiceName)

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

func (d *ExampleDatabase) Stop(
	_ context.Context,
) error {
	slog.Info("stopping service", "service", ServiceName)

	return nil
}
