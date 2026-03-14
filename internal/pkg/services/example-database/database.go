package exampledatabase

import (
	"context"
	"log/slog"
	"time"
)

const ServiceName = "example-database"

// ExampleDatabase demonstrates retry and readiness
// behavior. It simulates a database connection pool
// that retries on failure and signals ready after a
// short startup delay. Dependent services (like
// example-api) won't start until this signals ready.
type ExampleDatabase struct {
	readyCh chan struct{}
}

func New() (*ExampleDatabase, error) {
	return &ExampleDatabase{
		readyCh: make(chan struct{}),
	}, nil
}

func (d *ExampleDatabase) Name() string {
	return ServiceName
}

// MaxRetries makes ExampleDatabase implement the
// Retryable interface.
func (d *ExampleDatabase) MaxRetries() int {
	return 2 //nolint:mnd
}

func (d *ExampleDatabase) RetryDelay() time.Duration {
	return 2 * time.Second //nolint:mnd
}

// Ready makes ExampleDatabase implement the
// ReadyNotifier interface. The service manager waits
// for this before starting dependent services.
func (d *ExampleDatabase) Ready() <-chan struct{} {
	return d.readyCh
}

func (d *ExampleDatabase) Run(
	ctx context.Context,
) error {
	slog.Info("starting service", "service", ServiceName)

	// Simulate connection pool startup
	startupDelay := 2 * time.Second //nolint:mnd

	slog.Info("connecting to database",
		"service", ServiceName,
		"startupDelay", startupDelay,
	)

	select {
	case <-ctx.Done():
		return nil
	case <-time.After(startupDelay):
	}

	slog.Info("database ready, accepting connections",
		"service", ServiceName,
	)

	close(d.readyCh)

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
