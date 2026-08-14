package examplecrasher

import (
	"context"
	"errors"
	"time"

	"github.com/psyb0t/ctxscope"
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
	ctx = ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))
	logger := ctxscope.GetLogger(ctx)
	logger.Info("starting service")

	crashDelay := 10 * time.Second //nolint:mnd
	timer := time.NewTimer(crashDelay)

	defer timer.Stop()

	select {
	case <-ctx.Done():
		logger.Info("context cancelled, stopping service")

		return nil
	case <-timer.C:
		logger.Error("crashing on purpose")

		return errCrash
	}
}

func (c *ExampleCrasher) Stop(
	ctx context.Context,
) error {
	serviceCtx := ctxscope.Set(ctx, ctxscope.Attr("service", ServiceName))

	ctxscope.GetLogger(serviceCtx).Info("stopping service")

	return nil
}
