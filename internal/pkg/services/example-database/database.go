package exampledatabase

import (
	"context"
	"time"

	"github.com/psyb0t/ctxscope"
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
	ctx = ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))
	logger := ctxscope.GetLogger(ctx)
	logger.Info("starting service")

	// Simulate connection pool startup
	startupDelay := 2 * time.Second //nolint:mnd

	logger.Info("connecting to database",
		"startup_delay", startupDelay,
	)

	select {
	case <-ctx.Done():
		return nil
	case <-time.After(startupDelay):
	}

	logger.Info("database ready, accepting connections")

	close(d.readyCh)

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

func (d *ExampleDatabase) Stop(
	ctx context.Context,
) error {
	serviceCtx := ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))

	ctxscope.GetLogger(serviceCtx).Info("stopping service")

	return nil
}
