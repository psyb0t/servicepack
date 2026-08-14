package exampleapi

import (
	"context"
	"time"

	"github.com/psyb0t/ctxscope"
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
	ctx = ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))
	logger := ctxscope.GetLogger(ctx)
	logger.Info("starting service")

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

func (a *ExampleAPI) Stop(ctx context.Context) error {
	serviceCtx := ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))

	ctxscope.GetLogger(serviceCtx).Info("stopping service")

	return nil
}
