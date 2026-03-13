package exampleapi

import (
	"context"
	"log/slog"
	"time"

	exampledatabase "github.com/psyb0t/servicepack/internal/pkg/services/example-database"
	exampleflaky "github.com/psyb0t/servicepack/internal/pkg/services/example-flaky"
)

const ServiceName = "example-api"

// ExampleAPI demonstrates dependency behavior.
// It depends on "example-database" and "example-flaky"
// so it only starts after both are running. This shows
// how the API waits for the flaky service to recover
// from its retries before starting.
type ExampleAPI struct{}

func New() (*ExampleAPI, error) {
	return &ExampleAPI{}, nil
}

func (a *ExampleAPI) Name() string {
	return ServiceName
}

// Dependencies makes ExampleAPI implement the Dependent
// interface. The service manager will start
// example-database and example-flaky before this one.
func (a *ExampleAPI) Dependencies() []string {
	return []string{
		exampledatabase.ServiceName,
		exampleflaky.ServiceName,
	}
}

func (a *ExampleAPI) Run(ctx context.Context) error {
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

func (a *ExampleAPI) Stop(_ context.Context) error {
	slog.Info("stopping service", "service", ServiceName)

	return nil
}
